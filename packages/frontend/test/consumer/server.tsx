import { createPreferencePolicy, frontendRootAttributes, resolveInitialPreferences } from '@ztd-me/frontend'
import { createServer } from 'node:http'
import { randomBytes } from 'node:crypto'
import { readFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { renderToString } from 'react-dom/server'
import { Fixture } from './Fixture.js'

const client = fileURLToPath(new URL('../client/', import.meta.url))
const template = await readFile(path.join(client, 'index.html'), 'utf8')
const server = createServer(async (request, response) => {
  const url = new URL(request.url ?? '/', 'http://127.0.0.1:4317')
  if (url.pathname.startsWith('/assets/')) {
    const filename = path.basename(url.pathname)
    const type = filename.endsWith('.css') ? 'text/css' : filename.endsWith('.woff2') ? 'font/woff2' : 'text/javascript'
    response.setHeader('Content-Type', type)
    response.end(await readFile(path.join(client, 'assets', filename)))
    return
  }
  const domain = url.searchParams.get('cookie-domain')
  const policy = createPreferencePolicy({
    name: url.searchParams.get('cookie-name') ?? 'harbor.ui.v1',
    secure: url.searchParams.get('secure') === 'true',
    ...(domain === null ? {} : { domain }),
    ...(url.searchParams.get('mirror') === 'off' ? {} : { mirrorKey: 'harbor.ui.notification' }),
  })
  const initialPreferences = resolveInitialPreferences({
    policy, cookieHeader: request.headers.cookie, acceptLanguage: request.headers['accept-language'],
  })
  const styleNonce = randomBytes(18).toString('base64')
  const props = {
    policy,
    initialPreferences,
    styleNonce,
    application: url.pathname === '/application',
    footer: url.searchParams.get('footer') !== 'none',
    review: url.pathname === '/review',
  }
  const root = Object.entries(frontendRootAttributes(initialPreferences)).map(([key, value]) => `${key}="${value}"`).join(' ')
  const html = template.replace('lang="en"', root).replace('<!--content-->', renderToString(<Fixture {...props} />))
  const payload = JSON.stringify(props).replaceAll('<', '\u003c')
  response.setHeader('Content-Type', 'text/html; charset=utf-8')
  response.setHeader('Content-Security-Policy', `default-src 'self'; script-src 'self'; style-src-elem 'self' 'nonce-${styleNonce}'; style-src-attr 'unsafe-inline'; font-src 'self'`)
  response.end(html.replace('<!--initial-->', `<script type="application/json" id="initial">${payload}</script>`))
})
server.listen(4317, '127.0.0.1')
