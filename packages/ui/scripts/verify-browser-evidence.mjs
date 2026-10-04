import process from 'node:process'
import { verifyBrowserEvidence } from './browser-artifacts.mjs'

const verified = await verifyBrowserEvidence({ localFontPreview: Boolean(process.env.ZTD_LOCAL_FONT_PREVIEW) })
console.log(`Verified retained browser evidence: ${verified.join(', ')}`)
