import * as comlink from 'comlink'
import { Unsubscribable } from 'rxjs'
import { QueryTransformer } from 'sourcegraph'
import { ClientSearchAPI } from '../../client/api/search'
import { synchronousSubscription } from '../../util'

export class ExtSearch {
    constructor(private proxy: comlink.Remote<ClientSearchAPI>) {}

    public registerQueryTransformer(provider: QueryTransformer): Unsubscribable {
        return synchronousSubscription(this.proxy.$registerQueryTransformer(comlink.proxy(provider)))
    }
}
