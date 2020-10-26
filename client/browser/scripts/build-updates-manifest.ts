/* eslint "no-sync": "warn" */
import * as fs from 'fs'
import * as path from 'path'

const updatesManifestPath = path.resolve(__dirname, '..', 'src/browser-extension/updates.manifest.json')

interface Update {
    version: string
    update_link: string
}

function addVersionsToManifest(links: string[]): void {
    const updatesManifest = JSON.parse(fs.readFileSync(updatesManifestPath, 'utf8'))

    const updates: Update[] = []

    for (const link of links) {
        // `link` looks like gs://sourcegraph-for-firefox/sourcegraph_for_firefox-18.11.17.46-an+fx.xpi
        const match = link.match(/_firefox-(.*?)-/)
        if (!match) {
            throw new Error(`could not get version from ${link}`)
        }

        const version = match[1]

        updates.push({
            version,
            update_link: link.replace(/^gs:\/\//, 'https://storage.googleapis.com/'),
        })
    }

    ;(updatesManifest.addons['sourcegraph-for-firefox@sourcegraph.com'].updates as Update[]) = updates

    fs.writeFileSync(updatesManifestPath, JSON.stringify(updatesManifest, null, 2), 'utf8')
}

const links = process.argv.slice(2).filter(link => !link.match(/latest.xpi$/) && !link.match(/updates.json$/))

addVersionsToManifest(links)
