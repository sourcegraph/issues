declare module 'string-score' {
    function score(target: string, query: string, fuzzyFactor?: number): number

    export = score
}
