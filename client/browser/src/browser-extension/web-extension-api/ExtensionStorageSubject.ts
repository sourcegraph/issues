import { BehaviorSubject, NextObserver, Observable } from 'rxjs'
import { observeStorageKey, storage } from './storage'
import { LocalStorageItems } from './types'

/**
 * An RxJS subject that is backed by an extension storage item.
 */
export class ExtensionStorageSubject<T extends keyof LocalStorageItems>
    // eslint-disable-next-line rxjs/no-subclass
    extends Observable<LocalStorageItems[T]>
    implements NextObserver<LocalStorageItems[T]>, Pick<BehaviorSubject<LocalStorageItems[T]>, 'value'> {
    constructor(private key: T, defaultValue: LocalStorageItems[T]) {
        super(subscriber => {
            subscriber.next(this.value)
            return observeStorageKey('local', this.key).subscribe((item = defaultValue) => {
                this.value = item
                subscriber.next(item)
            })
        })
        this.value = defaultValue
    }

    public async next(value: LocalStorageItems[T]): Promise<void> {
        await storage.local.set({ [this.key]: value })
    }

    public value: LocalStorageItems[T]
}
