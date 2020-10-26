import React from 'react'
import { IconProps } from './icons'
import LanguageCIcon from 'mdi-react/LanguageCIcon'
import LanguageCppIcon from 'mdi-react/LanguageCppIcon'
import LanguageCsharpIcon from 'mdi-react/LanguageCsharpIcon'
import LanguageCss3Icon from 'mdi-react/LanguageCss3Icon'
import LanguageGoIcon from 'mdi-react/LanguageGoIcon'
import LanguageHaskellIcon from 'mdi-react/LanguageHaskellIcon'
import LanguageHtml5Icon from 'mdi-react/LanguageHtml5Icon'
import LanguageJavaIcon from 'mdi-react/LanguageJavaIcon'
import LanguageJavascriptIcon from 'mdi-react/LanguageJavascriptIcon'
import LanguageLuaIcon from 'mdi-react/LanguageLuaIcon'
import LanguagePhpIcon from 'mdi-react/LanguagePhpIcon'
import LanguagePythonIcon from 'mdi-react/LanguagePythonIcon'
import LanguageRIcon from 'mdi-react/LanguageRIcon'
import LanguageSwiftIcon from 'mdi-react/LanguageSwiftIcon'
import SassIcon from 'mdi-react/SassIcon'
import LanguageTypescriptIcon from 'mdi-react/LanguageTypescriptIcon'
import GraphqlIcon from 'mdi-react/GraphqlIcon'
import PowershellIcon from 'mdi-react/PowershellIcon'
import WebIcon from 'mdi-react/WebIcon'
import MarkdownIcon from 'mdi-react/LanguageMarkdownIcon'
import RubyIcon from 'mdi-react/LanguageRubyIcon'
import JsonIcon from 'mdi-react/CodeJsonIcon'
import { MdiReactIconComponentType } from 'mdi-react'

interface Props extends IconProps {
    language: string
}

/**
 * Record of known valid language values for the `lang:` filter to their icon in suggestions.
 */
export const languageIcons: Record<string, MdiReactIconComponentType | undefined> = {
    __proto__: null as any,

    bash: undefined,
    c: LanguageCIcon,
    cobol: undefined,
    clojure: undefined,
    cpp: LanguageCppIcon,
    csharp: LanguageCsharpIcon,
    css: LanguageCss3Icon,
    dart: undefined,
    go: LanguageGoIcon,
    graphql: GraphqlIcon,
    erlang: undefined,
    elixir: undefined,
    haskell: LanguageHaskellIcon,
    html: LanguageHtml5Icon,
    java: LanguageJavaIcon,
    javascript: LanguageJavascriptIcon,
    json: JsonIcon,
    julia: undefined,
    kotlin: undefined,
    lua: LanguageLuaIcon,
    markdown: MarkdownIcon,
    ocaml: undefined,
    objectivec: undefined,
    php: LanguagePhpIcon,
    protobuf: undefined,
    powershell: PowershellIcon,
    python: LanguagePythonIcon,
    r: LanguageRIcon,
    rust: undefined,
    ruby: RubyIcon,
    sass: SassIcon,
    scala: undefined,
    sql: undefined,
    swift: LanguageSwiftIcon,
    typescript: LanguageTypescriptIcon,
    webassembly: undefined,
}

export const LanguageIcon: React.FunctionComponent<Props> = ({ language, size }) => {
    const LanguageIconComponent = languageIcons[language] || WebIcon
    return <LanguageIconComponent size={size} />
}
