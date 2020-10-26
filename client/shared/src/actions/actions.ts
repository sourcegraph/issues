import { Contributions, Evaluated } from '../api/protocol'

export interface ActionsState {
    /** The contributions, merged from all extensions, or undefined before the initial emission. */
    contributions?: Evaluated<Contributions>
}
