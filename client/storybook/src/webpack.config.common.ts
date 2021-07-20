import fs from 'fs'
import path from 'path'

import { parse } from '@sqs/jsonc-parser'
import MonacoWebpackPlugin from 'monaco-editor-webpack-plugin'
import { WebpackPluginInstance, RuleSetRule, RuleSetUseItem } from 'webpack'

export const rootPath = path.resolve(__dirname, '../../..')
export const nodeModulesPath = path.resolve(rootPath, 'node_modules')
export const monacoEditorPath = path.resolve(nodeModulesPath, 'monaco-editor')
export const storybookWorkspacePath = path.resolve(rootPath, 'client/storybook')
export const dllBuildPath = path.resolve(storybookWorkspacePath, 'assets/dll-bundle')
export const dllBundleManifestPath = path.resolve(dllBuildPath, 'dll-bundle.manifest.json')

// eslint-disable-next-line no-sync
export const readJsonFile = (path: string): unknown => parse(fs.readFileSync(path, 'utf-8')) as unknown

// CSS rule for monaco-editor and other external plain CSS (skip SASS and PostCSS for build perf)
export const getMonacoCSSRule = (): RuleSetRule => ({
    test: /\.css$/,
    include: [monacoEditorPath],
    use: ['style-loader', { loader: 'css-loader' }],
})

// TTF rule for monaco-editor
export const getMonacoTTFRule = (): RuleSetRule => ({
    test: /\.ttf$/,
    include: [monacoEditorPath],
    type: 'asset/resource',
})

export const getBasicCSSLoader = (): RuleSetUseItem => ({
    loader: 'css-loader',
    options: { url: false },
})

export const getMonacoWebpackPlugin = (): WebpackPluginInstance =>
    new MonacoWebpackPlugin({
        languages: ['json'],
        features: [
            'bracketMatching',
            'clipboard',
            'coreCommands',
            'cursorUndo',
            'find',
            'format',
            'hover',
            'inPlaceReplace',
            'iPadShowKeyboard',
            'links',
            'suggest',
        ],
    })

export const dllPluginConfig = {
    context: dllBuildPath,
    name: 'dll_lib',
    path: path.resolve(dllBuildPath, 'dll-plugin.manifest.json'),
}
