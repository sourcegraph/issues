import { android } from './Android'
import { cncf } from './cncf'
import { golang } from './Golang'
import { kubernetes } from './Kubernetes'
import { o3de } from './o3de'
import { python2To3Metadata } from './Python2To3'
import { reactHooks } from './ReactHooks'
import { stackStorm } from './StackStorm'
import { stanford } from './Stanford'
import { temporal } from './Temporal'
import { RepogroupMetadata } from './types'

export const repogroupList: RepogroupMetadata[] = [
    cncf,
    python2To3Metadata,
    android,
    temporal,
    o3de,
    stackStorm,
    kubernetes,
    golang,
    reactHooks,
    stanford,
]
