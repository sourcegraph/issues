/* eslint-disable @typescript-eslint/no-use-before-define */
/* eslint rxjs/no-internal: warn */
import { Observable, ObservableInput, of, Operator, PartialObserver, Subscriber, TeardownLogic, zip } from 'rxjs'
import { fromArray } from 'rxjs/internal/observable/fromArray'
import { OuterSubscriber } from 'rxjs/internal/OuterSubscriber'
import { asap } from 'rxjs/internal/scheduler/asap'
import { subscribeToResult } from 'rxjs/internal/util/subscribeToResult'

/**
 * Like {@link combineLatest}, except that it does not wait for all Observables to emit before emitting an initial
 * value. It emits whenever any of the source Observables emit.
 *
 * If {@link defaultValue} is provided, it will be used to represent any source Observables
 * that have not yet emitted in the emitted array. If it is not provided, source Observables
 * that have not yet emitted will not be represented in the emitted array.
 *
 * Also unlike {@link combineLatest}, if the source Observables array is empty, it emits an empty array and
 * completes.
 *
 * This behavior is useful for the common pattern of combining providers: we don't want to block on the slowest
 * provider for the initial emission, and an empty array of providers should yield an empty array (instead of
 * yielding an Observable that never completes).
 *
 * @see {@link combineLatest}
 *
 * @todo Consider renaming this to combineProviders and making it also catchError from each Observable (and return
 * the error as a value).
 *
 * @param observables The source Observables.
 * @param defaultValue The value to emit for a source Observable if it has not yet emitted a value by the time
 * another Observable has emitted a value.
 * @returns An Observable of an array of the most recent values from each input Observable (or
 * {@link defaultValue}).
 */
export function combineLatestOrDefault<T>(observables: ObservableInput<T>[], defaultValue?: T): Observable<T[]> {
    switch (observables.length) {
        case 0:
            // No source observables: emit an empty array and complete
            return of([])
        case 1:
            // Only one source observable: no need to handle emission accumulation or default values
            return zip(...observables)
        default:
            return fromArray(observables).lift(new CombineLatestOperator(defaultValue))
    }
}

class CombineLatestOperator<T> implements Operator<T, T[]> {
    constructor(private defaultValue?: T) {}

    public call(subscriber: Subscriber<T[]>, source: any): TeardownLogic {
        return source.subscribe(new CombineLatestSubscriber(subscriber, this.defaultValue))
    }
}

class CombineLatestSubscriber<T> extends OuterSubscriber<T, T[]> {
    private activeObservables = 0
    private values: any[] = []
    private observables: Observable<any>[] = []
    private scheduled = false

    constructor(observer: PartialObserver<T[]>, private defaultValue?: T) {
        super(observer)
    }

    protected _next(observable: any): void {
        if (this.defaultValue !== undefined) {
            this.values.push(this.defaultValue)
        }
        this.observables.push(observable)
    }

    protected _complete(): void {
        this.activeObservables = this.observables.length
        for (let index = 0; index < this.observables.length; index++) {
            this.add(subscribeToResult(this, this.observables[index], this.observables[index], index))
        }
    }

    public notifyComplete(): void {
        this.activeObservables--
        if (this.activeObservables === 0 && this.destination.complete) {
            this.destination.complete()
        }
    }

    public notifyNext(_outerValue: T, innerValue: T[], outerIndex: number): void {
        const values = this.values
        values[outerIndex] = innerValue

        if (this.activeObservables === 1) {
            // Only 1 observable is active, so no need to buffer.
            //
            // This makes it possible to use RxJS's `of` in tests without specifying an explicit scheduler.
            if (this.destination.next) {
                this.destination.next(this.values.slice())
            }
            return
        }

        // Buffer all next values that are emitted at the same time into one emission.
        //
        // This makes tests (using expectObservable) easier to write.
        if (!this.scheduled) {
            this.scheduled = true
            asap.schedule(() => {
                if (this.scheduled && this.destination.next) {
                    this.destination.next(this.values.slice())
                }
                this.scheduled = false
            })
        }
    }
}
