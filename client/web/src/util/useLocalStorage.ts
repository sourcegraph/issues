import { useState } from 'react'

/**
 * A React hook to use and set state that is persisted in localStorage.
 *
 * @param key The localStorage key to use for persistence.
 * @param initialValue The initial value to use when there is no value in localStorage for the key.
 * @returns A getter and setter for the value (`const [foo, setFoo] = useLocalStorage('key', 123)`).
 */
export const useLocalStorage = <T>(
    key: string,
    initialValue: T
): [T, (value: T | ((previousValue: T) => T)) => void] => {
    const [storedValue, setStoredValue] = useState<T>(() => {
        const item = localStorage.getItem(key)
        return item ? (JSON.parse(item) as T) : initialValue
    })

    const setValue = (value: T | ((previousValue: T) => T)): void => {
        // We need to cast here because T could be a function type itself,
        // but we cannot tell TypeScript that functions are not allowed as T.
        const valueToStore = typeof value === 'function' ? (value as (previousValue: T) => T)(storedValue) : value
        window.localStorage.setItem(key, JSON.stringify(valueToStore))
        setStoredValue(valueToStore)
    }

    return [storedValue, setValue]
}
