import * as React from 'react'

interface Props {
    title?: string
}

export class PageTitle extends React.Component<Props, {}> {
    public static titleSet = false

    public componentDidMount(): void {
        if (PageTitle.titleSet) {
            console.error('more than one PageTitle used at the same time')
        }
        PageTitle.titleSet = true
        this.updateTitle(this.props.title)
    }

    public componentDidUpdate(): void {
        this.updateTitle(this.props.title)
    }

    public componentWillUnmount(): void {
        PageTitle.titleSet = false
        document.title = this.brandName()
    }

    public render(): JSX.Element | null {
        return null
    }

    private brandName(): string {
        if (!window.context) {
            return 'Sourcegraph'
        }
        const { branding } = window.context
        return branding ? branding.brandName : 'Sourcegraph'
    }

    private updateTitle(title?: string): void {
        document.title = title ? `${title} - ${this.brandName()}` : this.brandName()
    }
}
