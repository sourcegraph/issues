declare module '@storybook/addons' {
    interface Parameters {
        chromatic?: ChromaticParameters
    }
}

export interface ChromaticParameters {
    /** You can delay capture for a fixed time to allow your story to get into the intended state. */
    delay?: number
    disable?: boolean
    viewports?: number[]
}
