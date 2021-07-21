import React from 'react'

import terminalStyles from './Terminal.module.scss'

// 73 '=' characters are the 100% of the progress bar
const CHARACTERS_LENGTH = 73

export const Terminal: React.FunctionComponent = ({ children }) => (
    <section className={terminalStyles.terminalWrapper}>
        <ul className={terminalStyles.downloadProgressWrapper}>{children}</ul>
    </section>
)

export const TerminalTitle: React.FunctionComponent = ({ children }) => (
    <header className={terminalStyles.terminalTitle}>
        <code>{children}</code>
    </header>
)

export const TerminalLine: React.FunctionComponent = ({ children }) => (
    <li className={terminalStyles.terminalLine}>{children}</li>
)

export const TerminalDetails: React.FunctionComponent = ({ children }) => (
    <div>
        <code>{children}</code>
    </div>
)

export const TerminalProgress: React.FunctionComponent<{ progress: number; character: string }> = ({
    progress = 0,
    character = '#',
}) => {
    const numberOfChars = Math.ceil((progress / 100) * CHARACTERS_LENGTH)

    return <code className={terminalStyles.downloadProgress}>{character.repeat(numberOfChars)}</code>
}
