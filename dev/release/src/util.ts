import * as readline from 'readline'
import { readFile, writeFile, mkdir } from 'mz/fs'
import * as path from 'path'

/* eslint-disable @typescript-eslint/consistent-type-assertions */
export function formatDate(date: Date): string {
    return `${date.toLocaleString('en-US', {
        timeZone: 'UTC',
        dateStyle: 'medium',
        timeStyle: 'short',
    } as Intl.DateTimeFormatOptions)} (UTC)`
}
/* eslint-enable @typescript-eslint/consistent-type-assertions */

const addZero = (index: number): string => (index < 10 ? `0${index}` : `${index}`)

/**
 * Generates a link for comparing given Date with local time.
 */
export function timezoneLink(date: Date, linkName: string): string {
    const timeString = `${addZero(date.getUTCHours())}${addZero(date.getUTCMinutes())}`
    return `https://time.is/${timeString}_${date.getUTCDate()}_${date.toLocaleString('en-US', {
        month: 'short',
    })}_${date.getUTCFullYear()}_in_UTC?${encodeURI(linkName)}`
}

export async function readLine(prompt: string, cacheFile?: string): Promise<string> {
    if (!cacheFile) {
        return readLineNoCache(prompt)
    }

    try {
        return await readFile(cacheFile, { encoding: 'utf8' })
    } catch {
        const userInput = await readLineNoCache(prompt)
        await mkdir(path.dirname(cacheFile), { recursive: true })
        await writeFile(cacheFile, userInput)
        return userInput
    }
}

async function readLineNoCache(prompt: string): Promise<string> {
    const readlineInterface = readline.createInterface({
        input: process.stdin,
        output: process.stdout,
    })
    const userInput = await new Promise<string>(resolve => readlineInterface.question(prompt, resolve))
    readlineInterface.close()
    return userInput
}
