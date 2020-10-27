/**
 * Replaces `T` with `R` if it is assignable to `M`.
 */
type Replace<T, M, R> = T extends M ? R : T

/**
 * Recursively replaces all values in `T` assignable to `M` with `R`.
 */
export type DeepReplace<T, M, R> = {
    [K in keyof T]: DeepReplace<Replace<T[K], M, R>, M, R>
}

/**
 * Identity-function helper to ensure a value `T` is a subtype of `U`.
 *
 * @template U The type to check for (explicitly specify this)
 * @template T The actual type (inferred, don't specify this)
 */
// needed for type parameter
// eslint-disable-next-line unicorn/consistent-function-scoping
export const subtypeOf = <U>() => <T extends U>(value: T): T => value

/**
 * Returns true if `val` is not `null` or `undefined`
 */
export const isDefined = <T>(value: T): value is NonNullable<T> => value !== undefined && value !== null

/**
 * Returns a type guard that checks whether the given value is strictly equal to a specific value.
 * This can for example be used with `isNot()` to exclude string literals like `"loading"`.
 *
 * @param constant The value to compare to. Pass this with `as const` to improve type inference.
 */
export const isExactly = <T, C extends T>(constant: C) => (value: T): value is C => value === constant

/**
 * Negates a type guard.
 * Returns a function that returns `true` when the input is **not** the type checked for by the given type guard.
 * It therefor excludes a type from a union type.
 *
 * @param isType The type guard that checks whether the given input value should be excluded.
 */
export const isNot = <TInput, TExclude extends TInput>(isType: (value: TInput) => value is TExclude) => (
    value: TInput
): value is Exclude<TInput, TExclude> => !isType(value)

/**
 * Returns a function that returns `true` if the given `key` exists in the given object, narrowing down the type of the _object_.
 *
 * @param key The key of the property to check.
 */
export const hasProperty = <O extends object, K extends string | number | symbol>(key: K) => (
    object: O
): object is O & { [k in K]: unknown } => key in object

/**
 * Returns a function that returns `true` if the given `key` exists in the given object, narrowing down the type of the _key_.
 *
 * @param key The key of the property to check.
 */
export const keyExistsIn = <O extends object>(key: string | number | symbol, object: O): key is keyof O => key in object

/**
 * Returns a function that returns `true` if the given `key` of the object passes the given type guard.
 *
 * @param key The key of the property to check.
 * @param isType The type guard to evalute on the property value.
 */
export const property = <O extends object, K extends keyof O, T extends O[K]>(
    key: K,
    isType: (value: O[K]) => value is T
) => (object: O): object is O & Record<K, T> => isType(object[key])

/**
 * Resolves a tagged union type to a specific member of the union identified by the given tag value.
 */
export const isTaggedUnionMember = <O extends object, K extends keyof O, V extends O[K]>(key: K, tagValue: V) => (
    object: O
): object is Extract<O, Record<K, V>> => object[key] === tagValue

/**
 * Returns a function that returns `true` if the given value is an instance of the given class.
 *
 * @param constructor A reference to a class, e.g. `HTMLElement`
 */
export const isInstanceOf = <C extends new () => object>(constructor: C) => (
    value: unknown
): value is InstanceType<C> => value instanceof constructor

/**
 * Combines multiple type guards into one type guard that checks if the value passes any of the provided type guards.
 */
export function anyOf<T0, T1 extends T0, T2 extends Exclude<T0, T1>>(
    t1: (value: T0) => value is T1,
    t2: (value: Exclude<T0, T1>) => value is T2
): (value: T0) => value is T1 | T2
export function anyOf<T0, T1 extends T0, T2 extends Exclude<T0, T1>, T3 extends Exclude<T1, T2>>(
    t1: (value: T0) => value is T1,
    t2: (value: Exclude<T0, T1>) => value is T2,
    t3: (value: Exclude<T1, T2>) => value is T3
): (value: T0) => value is T1 | T2 | T3
export function anyOf<
    T0,
    T1 extends T0,
    T2 extends Exclude<T0, T1>,
    T3 extends Exclude<T1, T2>,
    T4 extends Exclude<T2, T3>
>(
    t1: (value: T0) => value is T1,
    t2: (value: Exclude<T0, T1>) => value is T2,
    t3: (value: Exclude<T1, T2>) => value is T3,
    t4: (value: Exclude<T2, T3>) => value is T4
): (value: T0) => value is T1 | T2 | T3 | T4
export function anyOf(...typeGuards: any[]): any {
    return (value: unknown) => typeGuards.some((guard: (value: unknown) => boolean) => guard(value))
}

/**
 * Combines multiple type guards into one type guard that checks if the value passes all of the provided type guards.
 */
export function allOf<T0, T1 extends T0, T2 extends T1>(
    t1: (value: T0) => value is T1,
    t2: (value: T1) => value is T2
): (value: T0) => value is T1 & T2
export function allOf<T0, T1 extends T0, T2 extends T1, T3 extends T2>(
    t1: (value: T0) => value is T1,
    t2: (value: T1) => value is T2,
    t3: (value: T2) => value is T3
): (value: T0) => value is T1 & T2 & T3
export function allOf<T0, T1 extends T0, T2 extends T1, T3 extends T2, T4 extends T3>(
    t1: (value: T0) => value is T1,
    t2: (value: T1) => value is T2,
    t3: (value: T2) => value is T3,
    t4: (value: T3) => value is T4
): (value: T0) => value is T1 & T2 & T3 & T4
export function allOf(...typeGuards: any[]): any {
    return (value: unknown) => typeGuards.every((guard: (value: unknown) => boolean) => guard(value))
}

/**
 * Returns a type guard for a simple condition that does not check the type of the argument (but something about the value).
 */
export const check = <T>(simpleCondition: (value: T) => boolean) => (value: T): value is T => simpleCondition(value)
