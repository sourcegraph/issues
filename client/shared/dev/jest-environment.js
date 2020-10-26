// Fork of https://github.com/facebook/jest/blob/19b6292dac229018b54a6bd7a0b1a1e2942952f0/packages/jest-environment-jsdom/src/index.ts
// But exposes jsdom as a global variable
// and uses latest JSDOM

'use strict'

/**
 * Copyright (c) Facebook, Inc. and its affiliates. All Rights Reserved.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */
const { JestFakeTimers } = require('@jest/fake-timers')
const { ModuleMocker } = require('jest-mock')
const { installCommonGlobals } = require('jest-util')
const { JSDOM, VirtualConsole } = require('jsdom')
function isWin(globals) {
  return globals.document !== undefined
}
// A lot of the globals expected by other APIs are `NodeJS.Global` and not
// `Window`, so we need to cast here and there
class JSDOMEnvironment {
  constructor(config, options = {}) {
    this.dom = new JSDOM('<!DOCTYPE html>', {
      pretendToBeVisual: true,
      runScripts: 'dangerously',
      url: config.testURL,
      virtualConsole: new VirtualConsole().sendTo(options.console || console),
      ...config.testEnvironmentOptions,
    })
    const global = (this.global = this.dom.window.document.defaultView)

    // JSDOM does not have SVGAElement implemented. Use a quick and dirty polyfill.
    // This does not implement href and target, which is impossible without mofifying JSDOM.
    global.SVGAElement = class SVGAElement extends global.SVGGraphicsElement {}

    // jsdom doesn't support document.queryCommandSupported(), needed for monaco-editor.
    // https://github.com/testing-library/react-testing-library/issues/546
    // eslint-disable-next-line @typescript-eslint/unbound-method
    this.dom.window.document.queryCommandSupported = () => false

    if (!global) {
      throw new Error('JSDOM did not return a Window object')
    }
    // Node's error-message stack size is limited at 10, but it's pretty useful
    // to see more than that when a test fails.
    this.global.Error.stackTraceLimit = 100
    installCommonGlobals(global, config.globals)
    // Report uncaught errors.
    let userErrorListenerCount = 0
    this.errorEventListener = event => {
      if (userErrorListenerCount === 0 && event.error) {
        process.emit('uncaughtException', event.error)
      }
    }
    global.addEventListener('error', this.errorEventListener)
    // However, don't report them as uncaught if the user listens to 'error' event.
    // In that case, we assume the might have custom error handling logic.
    /* eslint-disable @typescript-eslint/unbound-method */
    const originalAddListener = global.addEventListener
    const originalRemoveListener = global.removeEventListener
    global.addEventListener = function (...args) {
      if (args[0] === 'error') {
        userErrorListenerCount++
      }
      return originalAddListener.apply(this, args)
    }
    global.removeEventListener = function (...args) {
      if (args[0] === 'error') {
        userErrorListenerCount--
      }
      return originalRemoveListener.apply(this, args)
    }
    /* eslint-enable @typescript-eslint/unbound-method */
    this.moduleMocker = new ModuleMocker(global)
    const timerConfig = {
      idToRef: id => id,
      refToId: reference => reference,
    }
    this.fakeTimers = new JestFakeTimers({
      config,
      global,
      moduleMocker: this.moduleMocker,
      timerConfig,
    })
    this.global.jsdom = this.dom
  }
  setup() {
    return Promise.resolve()
  }
  teardown() {
    if (this.fakeTimers) {
      this.fakeTimers.dispose()
    }
    if (this.global) {
      if (this.errorEventListener && isWin(this.global)) {
        this.global.removeEventListener('error', this.errorEventListener)
      }
      // Dispose "document" to prevent "load" event from triggering.
      Object.defineProperty(this.global, 'document', { value: null })
      if (isWin(this.global)) {
        this.global.close()
      }
    }
    this.errorEventListener = null
    this.global = null
    this.dom = null
    this.fakeTimers = null
    return Promise.resolve()
  }
  runScript(script) {
    if (this.dom) {
      return this.dom.runVMScript(script)
    }
    return null
  }
}
module.exports = JSDOMEnvironment
