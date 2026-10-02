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
  const requestedProject = url.searchParams.get('project')
  const project = requestedProject === 'memory' || requestedProject === 'showcase' ? requestedProject : 'website'
  const environment = url.searchParams.get('environment')
  const production = environment === 'production'
  const preview = environment === 'preview'
  const policy = createPreferencePolicy({
    environment: production ? 'production' : preview ? 'preview' : 'development',
    namespace: project,
    hostname: production || preview ? url.searchParams.get('hostname') ?? 'ztd.me' : '127.0.0.1',
    protocol: production || preview ? 'https:' : 'http:',
  })
  const initialPreferences = resolveInitialPreferences({
    policy, cookieHeader: request.headers.cookie, acceptLanguage: request.headers['accept-language'],
  })
  const styleNonce = randomBytes(18).toString('base64')
  const props = { policy, initialPreferences, styleNonce, application: url.pathname === '/application' }
  const root = Object.entries(frontendRootAttributes(initialPreferences)).map(([key, value]) => `${key}="${value}"`).join(' ')
  const html = template.replace('lang="en"', root).replace('<!--content-->', renderToString(<Fixture {...props} />))
  const payload = JSON.stringify(props).replaceAll('<', '\u003c')
  response.setHeader('Content-Type', 'text/html; charset=utf-8')
  response.setHeader('Content-Security-Policy', `default-src 'self'; script-src 'self'; style-src-elem 'self' 'nonce-${styleNonce}'; style-src-attr 'unsafe-inline'; font-src 'self'`)
  response.end(html.replace('<!--initial-->', `<script type="application/json" id="initial">${payload}</script>`))
})
server.listen(4317, '127.0.0.1')
