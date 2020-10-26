const fetch = require('node-fetch')
globalThis.fetch = fetch
const { Request, Response, Headers } = fetch
Object.assign(globalThis, { Request, Response, Headers })
