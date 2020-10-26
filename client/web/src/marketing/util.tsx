export const IS_CHROME = !!window.chrome

let lastDayActive = localStorage.getItem('last-day-active')
export let daysActiveCount = parseInt(localStorage.getItem('days-active-count') || '', 10) || 0

/**
 * Function called upon initial app load that checks if the user is visiting
 * on a new day, and if so, updates persistent local storage with that information
 */
export function updateUserSessionStores(): void {
    if (new Date().toDateString() !== lastDayActive) {
        daysActiveCount++
        lastDayActive = new Date().toDateString()
        localStorage.setItem('days-active-count', daysActiveCount.toString())
        localStorage.setItem('last-day-active', lastDayActive)
    }
}
