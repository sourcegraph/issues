import * as constants from './constants'
import * as fs from 'mz/fs'
import * as path from 'path'

/**
 * Construct the path of the SQLite database file for the given dump.
 *
 * @param storageRoot The path where SQLite databases are stored.
 * @param id The ID of the dump.
 */
export function dbFilename(storageRoot: string, id: number): string {
    return path.join(storageRoot, constants.DBS_DIR, `${id}.lsif.db`)
}

/**
 * Returns the identifier of the database file. Handles both of the
 * following formats:
 *
 * - `{id}.lsif.db`
 * - `{id}-{repo}-{commit}.lsif.db`
 *
 * @param filename The filename.
 */
export function idFromFilename(filename: string): number | undefined {
    const id = parseInt(path.parse(filename).name.split('-')[0], 10)
    if (!isNaN(id)) {
        return id
    }

    return undefined
}

/**
 * Ensure the directory exists.
 *
 * @param directoryPath The directory path.
 */
export async function ensureDirectory(directoryPath: string): Promise<void> {
    try {
        await fs.mkdir(directoryPath)
    } catch (error) {
        if (!(error && error.code === 'EEXIST')) {
            throw error
        }
    }
}

/**
 * Delete the file if it exists. Throws errors that are not ENOENT.
 *
 * @param filePath The path to delete.
 */
export async function tryDeleteFile(filePath: string): Promise<boolean> {
    try {
        await fs.unlink(filePath)
        return true
    } catch (error) {
        if (!(error && error.code === 'ENOENT')) {
            throw error
        }

        return false
    }
}
