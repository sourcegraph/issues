/* eslint-disable rxjs/no-subclass */
/* eslint-disable @typescript-eslint/no-use-before-define */
import {
    asyncScheduler,
    MonoTypeOperatorFunction,
    Observable,
    Operator,
    SchedulerLike,
    Subscriber,
    Subscription,
    TeardownLogic,
} from 'rxjs'

/**
 * Emits `valuesPerWindow` values from the source Observable, then ignores subsequent source values
 * for `windowDuration` milliseconds, then repeats this process.
 *
 * It is useful for throttling user input when the user wants immediate responses for the first
 * several inputs, but if there is a repeated input (such as a depressed key that repeats), it is
 * acceptable to ignore intermediate inputs.
 *
 * @see {@link throttleTime}
 *
 * @param windowDuration Time to wait before emitting another value after emitting the last value
 * allowed by `valuesPerWindow`, measured in milliseconds or the time unit determined internally by
 * the optional `scheduler`.
 * @param valuesPerWindow The number of values to allow per time window.
 * @param The {@link SchedulerLike} to use for managing the timers that handle the throttling.
 * @returns An Observable that performs the throttle operation to limit the rate of emissions from
 * the source.
 */
export function throttleTimeWindow<T>(
    windowDuration: number,
    valuesPerWindow = 1,
    scheduler: SchedulerLike = asyncScheduler
): MonoTypeOperatorFunction<T> {
    return (source: Observable<T>) =>
        source.lift(new ThrottleTimeWindowOperator(windowDuration, valuesPerWindow, scheduler))
}

class ThrottleTimeWindowOperator<T> implements Operator<T, T> {
    constructor(private windowDuration: number, private valuesPerWindow: number, private scheduler: SchedulerLike) {}

    public call(subscriber: Subscriber<T>, source: any): TeardownLogic {
        return source.subscribe(
            new ThrottleTimeWindowSubscriber(subscriber, this.windowDuration, this.valuesPerWindow, this.scheduler)
        )
    }
}

class ThrottleTimeWindowSubscriber<T> extends Subscriber<T> {
    private throttled: Subscription | undefined
    private _hasTrailingValue = false
    private _trailingValue: T | undefined
    private intervalEnds: number | undefined
    private valuesInCurrentWindow = 0

    constructor(
        destination: Subscriber<T>,
        private windowDuration: number,
        private valuesPerWindow: number,
        private scheduler: SchedulerLike
    ) {
        super(destination)
    }

    protected _next(value: T): void {
        if (this.throttled) {
            this._trailingValue = value
            this._hasTrailingValue = true
        } else {
            if (
                this.valuesInCurrentWindow === 0 ||
                (this.intervalEnds !== undefined && this.scheduler.now() > this.intervalEnds)
            ) {
                this.intervalEnds = this.scheduler.now() + this.windowDuration
            }
            this.valuesInCurrentWindow++
            if (this.valuesInCurrentWindow === this.valuesPerWindow) {
                this.throttled = this.scheduler.schedule<DispatchArgument<T>>(
                    dispatchNext,
                    this.intervalEnds! - this.scheduler.now(),
                    {
                        subscriber: this,
                    }
                )
                this.add(this.throttled)
            }
            this.destination.next!(value)
        }
    }

    protected _complete(): void {
        if (this._hasTrailingValue) {
            this.destination.next!(this._trailingValue)
            this.destination.complete!()
        } else {
            this.destination.complete!()
        }
    }

    public clearThrottle(): void {
        const throttled = this.throttled
        if (throttled) {
            if (this._hasTrailingValue) {
                this.destination.next!(this._trailingValue)
                this._trailingValue = undefined
                this._hasTrailingValue = false
            }
            throttled.unsubscribe()
            this.remove(throttled)
            this.throttled = undefined
        }
        this.valuesInCurrentWindow = 0
        this.intervalEnds = undefined
    }
}

interface DispatchArgument<T> {
    subscriber: ThrottleTimeWindowSubscriber<T>
}

function dispatchNext<T>(argument: DispatchArgument<T> | undefined): void {
    if (argument) {
        argument.subscriber.clearThrottle()
    }
}
