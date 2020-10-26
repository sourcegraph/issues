import { basename } from 'path'

/** The LSP mode used for plain text files and all other unrecognized files. */
const PLAINTEXT_MODE = 'plaintext'

/**
 * getModeFromPath returns the LSP mode for the provided file path. If the file path does not correspond to any
 * known mode, 'plaintext' is returned.
 */
export function getModeFromPath(path: string): string {
    const fileName = basename(path)
    const extension = getPathExtension(path)

    return getModeFromExactFilename(fileName) || getModeFromExtension(extension) || PLAINTEXT_MODE
}

/**
 * getModeFromExactFilename returns the LSP mode for the
 * provided file name (e.g. "dockerfile")
 *
 * Cherry picked from https://github.com/github/linguist/blob/master/lib/linguist/languages.yml
 */
function getModeFromExactFilename(fileName: string): string | undefined {
    switch (fileName.toLowerCase()) {
        case 'dockerfile':
            return 'dockerfile'

        default:
            return undefined
    }
}

/**
 * getModeFromExtension returns the LSP mode for the
 * provided file extension (e.g. "jsx")
 *
 * Cherry picked from https://github.com/isagalaev/highlight.js/tree/master/src/languages
 * and https://github.com/github/linguist/blob/master/lib/linguist/languages.yml.
 */
function getModeFromExtension(extension: string): string | undefined {
    switch (extension.toLowerCase()) {
        // Ada
        case 'adb':
        case 'ada':
        case 'ads':
            return 'ada'

        // Actionscript
        case 'as':
            return 'actionscript'

        // Apache
        case 'apacheconf':
            return 'apache'

        // Applescript
        case 'applescript':
        case 'scpt':
            return 'applescript'

        // Bash
        case 'sh':
        case 'bash':
        case 'zsh':
            return 'shell'

        // Clojure
        case 'clj':
        case 'cljs':
        case 'cljx':
            return 'clojure'

        // CSS
        case 'css':
            return 'css'

        // CMake
        case 'cmake':
        case 'cmake.in':
        case 'in': // TODO(john): hack b/c we don't properly parse extensions w/ '.' in them
            return 'cmake'

        // Coffeescript
        case 'coffee':
        case 'cake':
        case 'cson':
        case 'cjsx':
        case 'iced':
            return 'coffescript'

        // C#
        case 'cs':
        case 'csx':
            return 'csharp'

        // C++
        case 'c':
        case 'cc':
        case 'cpp':
        case 'cxx':
        case 'c++':
        case 'h++':
        case 'hh':
        case 'h':
        case 'hpp':
            return 'cpp'

        // CUDA
        case 'cu':
        case 'cuh':
            return 'cuda'

        // Dart
        case 'dart':
            return 'dart'

        // Diff
        case 'diff':
        case 'patch':
            return 'diff'

        // Django
        case 'jinja':
            return 'django'

        // DOS
        case 'bat':
        case 'cmd':
            return 'dos'

        // Elixir
        case 'ex':
        case 'exs':
            return 'elixir'

        // Elm
        case 'elm':
            return 'elm'

        // Erlang
        case 'erl':
            return 'erlang'

        // Fortran
        case 'f':
        case 'for':
        case 'frt':
        case 'fr':
        // case 'fs':
        case 'forth':
        case '4th':
        case 'fth':
            return 'fortran'

        // F#
        case 'fs':
            return 'fsharp'

        // Go
        case 'go':
            return 'go'

        // GraphQL
        case 'graphql':
            return 'graphql'

        // Groovy
        case 'groovy':
            return 'groovy'

        // HAML
        case 'haml':
            return 'haml'

        // Handlebars
        case 'hbs':
        case 'handlebars':
            return 'handlebars'

        // Haskell
        case 'hs':
        case 'hsc':
            return 'haskell'

        // HTML
        case 'htm':
        case 'html':
        case 'xhtml':
            return 'html'

        // INI
        case 'ini':
        case 'cfg':
        case 'prefs':
        case 'pro':
        case 'properties':
            return 'ini'

        // Java
        case 'java':
            return 'java'

        // JavaScript
        case 'js':
        case 'jsx':
        case 'es':
        case 'es6':
        case 'jss':
        case 'jsm':
            return 'javascript'

        // JSON
        case 'json':
        case 'sublime_metrics':
        case 'sublime_session':
        case 'sublime-keymap':
        case 'sublime-mousemap':
        case 'sublime-project':
        case 'sublime-settings':
        case 'sublime-workspace':
            return 'json'

        // Jsonnet
        case 'jsonnet':
        case 'libsonnet':
            return 'jsonnet'

        // Julia
        case 'jl':
            return 'julia'

        // Kotlin
        case 'kt':
        case 'ktm':
        case 'kts':
            return 'kotlin'

        // Less
        case 'less':
            return 'less'

        // Lisp
        case 'lisp':
        case 'asd':
        case 'cl':
        case 'lsp':
        case 'l':
        case 'ny':
        case 'podsl':
        case 'sexp':
        case 'el':
            return 'lisp'

        // Lua
        case 'lua':
        case 'fcgi':
        case 'nse':
        case 'pd_lua':
        case 'rbxs':
        case 'wlua':
            return 'lua'

        // Makefile
        case 'mk':
        case 'mak':
            return 'makefile'

        // Markdown
        case 'md':
        case 'mkdown':
        case 'mkd':
            return 'markdown'

        // nginx
        case 'nginxconf':
            return 'nginx'

        // Objective-C
        case 'm':
        case 'mm':
            return 'objectivec'

        // OCaml
        case 'ml':
        case 'eliom':
        case 'eliomi':
        case 'ml4':
        case 'mli':
        case 'mll':
        case 'mly':
        case 're': // reason has the same language server as ocaml
            return 'ocaml'

        // Pascal
        case 'p':
        case 'pas':
        case 'pp':
            return 'pascal'

        // Perl
        case 'pl':
        case 'al':
        case 'cgi':
        // case 'fcgi':
        case 'perl':
        case 'ph':
        case 'plx':
        case 'pm':
        case 'pod':
        case 'psgi':
        case 't':
            return 'perl'

        // PHP
        case 'php':
        case 'phtml':
        case 'php3':
        case 'php4':
        case 'php5':
        case 'php6':
        case 'php7':
        case 'phps':
            return 'php'

        // Powershell
        case 'ps1':
        case 'psd1':
        case 'psm1':
            return 'powershell'

        // Proto
        case 'proto':
            return 'protobuf'

        // Python
        case 'py':
        case 'pyc':
        case 'pyd':
        case 'pyo':
        case 'pyw':
        case 'pyz':
            return 'python'

        // R
        case 'r':
        case 'rd':
        case 'rsx':
            return 'r'

        // Ruby
        case 'rb':
        case 'builder':
        case 'eye':
        // case 'fcgi':
        case 'gemspec':
        case 'god':
        case 'jbuilder':
        case 'mspec':
        case 'pluginspec':
        case 'podspec':
        case 'rabl':
        case 'rake':
        case 'rbuild':
        case 'rbw':
        case 'rbx':
        case 'ru':
        case 'ruby':
        case 'spec':
        case 'thor':
        case 'watchr':
            return 'ruby'

        // Rust
        case 'rs':
        case 'rs.in':
            return 'rust'

        // SASS
        case 'sass':
        case 'scss':
            return 'scss'

        // Scala
        case 'sbt':
        case 'sc':
        case 'scala':
            return 'scala'

        // Scheme
        case 'scm':
        case 'sch':
        case 'sls':
        case 'sps':
        case 'ss':
            return 'scheme'

        // Smalltalk
        case 'st':
            return 'smalltalk'

        // SQL
        case 'sql':
            return 'sql'

        // Stylus
        case 'styl':
            return 'stylus'

        // Swift
        case 'swift':
            return 'swift'

        // Thrift
        case 'thrift':
            return 'thrift'

        // TypeScript
        case 'ts':
        case 'tsx':
            return 'typescript'

        // Twig
        case 'twig':
            return 'twig'

        // Visual Basic
        case 'vb':
            return 'vbnet'
        case 'vbs':
            return 'vbscrip'

        // Verilog, including SystemVerilog
        case 'v':
        case 'veo':
        case 'sv':
        case 'svh':
        case 'svi':
            return 'verilog'

        // VHDL
        case 'vhd':
        case 'vhdl':
            return 'vhdl'

        // VIM
        case 'vim':
            return 'vim'

        // XML
        case 'xml':
        case 'adml':
        case 'admx':
        case 'ant':
        case 'axml':
        case 'builds':
        case 'ccxml':
        case 'clixml':
        case 'cproject':
        case 'csl':
        case 'csproj':
        case 'ct':
        case 'dita':
        case 'ditamap':
        case 'ditaval':
        case 'dll.config':
        case 'dotsettings':
        case 'filters':
        case 'fsproj':
        case 'fxml':
        case 'glade':
        case 'gml':
        case 'grxml':
        case 'iml':
        case 'ivy':
        case 'jelly':
        case 'jsproj':
        case 'kml':
        case 'launch':
        case 'mdpolicy':
        case 'mjml':
        case 'mod':
        case 'mxml':
        case 'nproj':
        case 'nuspec':
        case 'odd':
        case 'osm':
        case 'pkgproj':
        case 'plist':
        // case 'pluginspec':
        case 'props':
        case 'ps1xml':
        case 'psc1':
        case 'pt':
        case 'rdf':
        case 'resx':
        case 'rss':
        // case 'sch':
        case 'scxml':
        case 'sfproj':
        case 'srdf':
        case 'storyboard':
        case 'stTheme':
        case 'sublime-snippet':
        case 'targets':
        case 'tmCommand':
        case 'tml':
        case 'tmLanguage':
        case 'tmPreferences':
        case 'tmSnippet':
        case 'tmTheme':
        // case 'ts':
        // case 'tsx':
        case 'ui':
        case 'urdf':
        case 'ux':
        case 'vbproj':
        case 'vcxproj':
        case 'vsixmanifest':
        case 'vssettings':
        case 'vstemplate':
        case 'vxml':
        case 'wixproj':
        case 'wsdl':
        case 'wsf':
        case 'wxi':
        case 'wxl':
        case 'wxs':
        case 'x3d':
        case 'xacro':
        case 'xaml':
        case 'xib':
        case 'xlf':
        case 'xliff':
        case 'xmi':
        case 'xml.dist':
        case 'xproj':
        case 'xsd':
        case 'xspec':
        case 'xul':
        case 'zcml':
            return 'xml'

        // YAML
        case 'yml':
        case 'yaml':
            return 'yaml'

        default:
            return undefined
    }
}

export function getPathExtension(path: string): string {
    const pathSplit = path.split('.')
    if (pathSplit.length === 1) {
        return ''
    }
    if (pathSplit.length === 2 && pathSplit[0] === '') {
        return '' // e.g. .gitignore
    }
    return pathSplit[pathSplit.length - 1].toLowerCase()
}
