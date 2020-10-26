declare module 'terser-webpack-plugin' {
    import { Plugin } from 'webpack'

    declare class TerserPlugin extends Plugin {
        constructor(options?: TerserPlugin.TerserPluginOptions)
    }

    declare namespace TerserPlugin {
        interface TerserPluginOptions {
            test?: RegExp | RegExp[]
            include?: RegExp | RegExp[]
            exclude?: RegExp | RegExp[]
            cache?: boolean | string
            parallel?: boolean | number
            sourceMap?: boolean
            minify?: boolean
            terserOptions?: TerserOptions
            extractComments?: boolean | RegExp | ((node: object, comment: string) => boolean) | ExtractCommentsOptions
            warningsFilter?: (warning: string, source: string) => boolean
        }

        interface TerserOptions {
            ie8?: boolean
            ecma?: number
            parse?: object
            mangle?: boolean | object
            output?: object
            compress?: boolean | object
            warnings?: boolean
            module?: boolean
            toplevel?: boolean
            nameCache?: object
            keep_classnames?: boolean
            keep_fnames?: boolean
            safari10?: boolean
        }

        interface ExtractCommentsOptions {
            condition?: RegExp | ((node: object, comment: string) => boolean)
            filename?: string | ((originalFileName: string) => string)
            banner?: boolean | string | ((fileName: string) => string)
        }
    }

    export = TerserPlugin
}
