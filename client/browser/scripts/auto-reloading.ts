import signale from 'signale'
import socketIo from 'socket.io'

/**
 * Returns a trigger function that notifies the extension to reload itself.
 */
export const initializeServer = (): (() => void) => {
    const logger = new signale.Signale({ scope: 'Auto reloading' })
    logger.config({ displayTimestamp: true })

    // Since this port is hard-coded, it must match background.ts
    const socketIOServer = socketIo.listen(8890)
    logger.await('Ready for a browser extension to connect')
    socketIOServer.on('connect', () => {
        logger.info('Browser extension connected')
    })
    socketIOServer.on('disconnect', () => {
        logger.info('Browser extension disconnected')
    })

    return () => {
        if (Object.keys(socketIOServer.clients().connected).length === 0) {
            logger.warn('No browser extension has connected yet, so no reload was triggered')
            logger.warn("- Make sure it's enabled")
            logger.warn("- Make sure it's in developer mode (unpacked extension)")
            logger.warn('- Try manually reloading it 🔄')
        } else {
            logger.info('Triggering a reload of browser extensions')
            socketIOServer.emit('file.change', {})
        }
    }
}
