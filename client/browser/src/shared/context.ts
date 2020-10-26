enum AppEnvironment {
    Extension,
    Page,
}

enum ScriptEnvironment {
    Content,
    Background,
    Options,
}

interface AppContext {
    appEnvironment: AppEnvironment
    scriptEnvironment: ScriptEnvironment
}

function getContext(): AppContext {
    const appEnvironment = window.SG_ENV === 'EXTENSION' ? AppEnvironment.Extension : AppEnvironment.Page

    let scriptEnvironment: ScriptEnvironment = ScriptEnvironment.Content
    if (appEnvironment === AppEnvironment.Extension) {
        if (window.location.pathname.includes('options.html')) {
            scriptEnvironment = ScriptEnvironment.Options
        } else if (globalThis.browser && browser.runtime.getBackgroundPage) {
            scriptEnvironment = ScriptEnvironment.Background
        }
    }

    return {
        appEnvironment,
        scriptEnvironment,
    }
}

const context = getContext()

export const isBackground = context.scriptEnvironment === ScriptEnvironment.Background
export const isOptions = context.scriptEnvironment === ScriptEnvironment.Options

export const isExtension = context.appEnvironment === AppEnvironment.Extension
export const isInPage = !isExtension

export const isPhabricator = Boolean(document.querySelector('.phabricator-wordmark'))
