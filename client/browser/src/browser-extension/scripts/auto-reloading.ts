import '../../shared/polyfills'

import socketIoClient from 'socket.io-client'

/**
 * Reloads the extension when notified from the development server. Only enabled
 * during development when `process.env.AUTO_RELOAD !== 'false'.
 */
async function main(): Promise<void> {
    const self = await browser.management.getSelf()
    if (self.installType === 'development') {
        // Since the port is hard-coded, it must match scripts/dev.ts
        socketIoClient.connect('http://localhost:8890').on('file.change', () => browser.runtime.reload())
    }
}

main().catch(console.error.bind(console))
