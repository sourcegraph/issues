/**
 * Returns all but the last element of path, or "." if that would be the empty path.
 */
export function dirname(path: string): string {
    return path.split('/').slice(0, -1).join('/') || '.'
}

/**
 * Returns the last element of path, or "." if path is empty.
 */
export function basename(path: string): string {
    return path.split('/').slice(-1)[0] || '.'
}
