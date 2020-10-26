export function getFilepathFromFileForDiff(fileContainer: HTMLElement): { filePath: string; baseFilePath?: string } {
    const filePath = fileContainer.children[3].textContent as string
    const metas = fileContainer.querySelectorAll('.differential-meta-notice')
    let baseFilePath: string | undefined
    const movedFilePrefix = 'This file was moved from '
    for (const meta of metas) {
        let metaText = meta.textContent!
        if (metaText.startsWith(movedFilePrefix)) {
            metaText = metaText.slice(0, -1) // remove trailing '.'
            baseFilePath = metaText.split(movedFilePrefix)[1]
            break
        }
    }
    return { filePath, baseFilePath }
}

export function getFilePathFromFileForRevision(codeView: HTMLElement): string {
    const filePathContainer = document.querySelector('.differential-file-icon-header')
    if (!filePathContainer) {
        throw new Error('Unable to find file path container for revision code view')
    }

    if (!filePathContainer.textContent) {
        throw new Error('`textContent` is undefined for revision file path container')
    }

    return filePathContainer.textContent
}
