function isFontCss(url) {
  return new URL(url).origin === 'https://fonts.googleapis.com'
    || url.includes('/__local-noto-preview/fonts.css')
}

function readFamilies(css, url, families) {
  for (const [, body] of css.matchAll(/@font-face\s*\{([^}]+)\}/gu)) {
    const family = body.match(/font-family:\s*['"]([^'"]+)['"]/u)?.[1]
    const source = body.match(/url\(([^)]+)\)/u)?.[1]?.replace(/^['"]|['"]$/gu, '')
    if (!family || !source) {
      continue
    }
    const href = new URL(source, url).href
    const records = families.get(href) ?? []
    const weight = body.match(/font-weight:\s*([^;]+)/u)?.[1]
    if (!records.some(record => record.family === family && record.weight === weight)) {
      records.push({ family, weight })
    }
    families.set(href, records)
  }
}

export async function transferProfile(page) {
  const session = await page.context().newCDPSession(page)
  await session.send('Network.enable')
  let active
  const families = new Map()
  session.on('Network.requestServedFromCache', ({ requestId }) => active?.cached.add(requestId))
  session.on('Network.responseReceived', ({ requestId, type, response }) => {
    if (!active || (type !== 'Font' && !isFontCss(response.url))) {
      return
    }
    active.responses.set(requestId, {
      url: response.url,
      kind: type === 'Font' ? 'font' : 'font-css',
      status: response.status,
      cached: Boolean(response.fromDiskCache || response.fromServiceWorker),
      httpResponseBytes: 0,
    })
  })
  session.on('Network.loadingFinished', ({ requestId, encodedDataLength }) => {
    const response = active?.responses.get(requestId)
    if (response) {
      response.httpResponseBytes = encodedDataLength
    }
  })
  page.on('response', (response) => {
    if (!active || (response.request().resourceType() !== 'font' && !isFontCss(response.url()))) {
      return
    }
    const phase = active
    phase.bodies.push(response.body().then((bytes) => {
      phase.decoded.set(response.url(), bytes.length)
      if (isFontCss(response.url())) {
        readFamilies(bytes.toString(), response.url(), families)
      }
    }))
  })
  return async (phase, navigate) => {
    active = { phase, responses: new Map(), cached: new Set(), bodies: [], decoded: new Map() }
    await navigate()
    await page.evaluate(async () => document.fonts.ready)
    await Promise.all(active.bodies)
    return [
      ...active.responses,
    ].map(([id, response]) => ({
      ...response,
      cached: response.cached || active.cached.has(id),
      httpDecodedBodyBytes: active.decoded.get(response.url),
      faces: families.get(response.url) ?? [],
    }))
  }
}
