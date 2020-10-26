declare module '@storybook/addons' {
    interface Parameters {
        design?: DesignParameters | DesignParameters[]
    }
}

export interface DesignParameters {
    type: 'figma' | 'iframe' | 'image' | 'link' | 'pdf'
    url: string

    /**
     * Change the name of the tab
     */
    name?: string
    options?: {
        /**
         * @default 'panel'
         */
        renderTarget?: 'tab' | 'panel'
    }
}
