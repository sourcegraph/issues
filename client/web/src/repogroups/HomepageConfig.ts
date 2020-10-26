import { python2To3Metadata } from './Python2To3'
import { android } from './Android'
import { kubernetes } from './Kubernetes'
import { golang } from './Golang'
import { reactHooks } from './ReactHooks'
import { stanford } from './Stanford'
import { cncf } from './cncf'
import { RepogroupMetadata } from './types'

export const repogroupList: RepogroupMetadata[] = [
    cncf,
    python2To3Metadata,
    android,
    kubernetes,
    golang,
    reactHooks,
    stanford,
]

export const homepageLanguageList: { name: string; filterName: string }[] = [
    { name: 'C', filterName: 'c' },
    { name: 'C++', filterName: 'cpp' },
    { name: 'C#', filterName: 'csharp' },
    { name: 'CSS', filterName: 'css' },
    { name: 'Go', filterName: 'go' },
    { name: 'GraphQL', filterName: 'graphql' },
    { name: 'Haskell', filterName: 'haskell' },
    { name: 'HTML', filterName: 'html' },
    { name: 'Java', filterName: 'java' },
    { name: 'JavaScript', filterName: 'javascript' },
    { name: 'JSON', filterName: 'json' },
    { name: 'Lua', filterName: 'lua' },
    { name: 'Markdown', filterName: 'markdown' },
    { name: 'PHP', filterName: 'php' },
    { name: 'PowerShell', filterName: 'powershell' },
    { name: 'Python', filterName: 'python' },
    { name: 'R', filterName: 'r' },
    { name: 'Ruby', filterName: 'ruby' },
    { name: 'Sass', filterName: 'sass' },
    { name: 'Swift', filterName: 'swift' },
    { name: 'TypeScript', filterName: 'typescript' },
]
