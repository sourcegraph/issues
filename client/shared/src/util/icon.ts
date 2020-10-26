/**
 * Determines if the value describes an encoded SVG image.
 *
 * @param value The raw value.
 */

export function isEncodedSVG(value: string): boolean {
    return /^data:image\/svg\+xml(;base64)?,/.test(value)
}

/**
 * Determines if the value describes an encoded PNG image.
 *
 * @param value The raw value.
 */
export function isEncodedPNG(value: string): boolean {
    return /^data:image\/png(;base64)?,/.test(value)
}

const imageValidators = [isEncodedSVG, isEncodedPNG]

/**
 * Determines if an icon can be used as the src of an image element.
 *
 * @param value The raw icon value.
 */
export function isEncodedImage(value: string): boolean {
    return imageValidators.some(validator => validator(value))
}
