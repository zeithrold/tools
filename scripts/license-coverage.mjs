import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { readdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'

export async function assertPackedLicense(archive, packageRoot, notices = {}) {
  const unpack = file => execFileSync('tar', [
    '-xOf',
    archive,
    `package/${file}`,
  ], { encoding: 'utf8' })
  const manifest = JSON.parse(unpack('package.json'))
  assert.equal(manifest.license, 'MIT', 'Packed package must declare MIT')
  const license = await readFile(path.join(packageRoot, 'LICENSE'), 'utf8')
  assert.equal(unpack('LICENSE'), license, 'Packed package must retain the complete original MIT notice')
  for (const [destination, source] of Object.entries(notices)) {
    const original = await readFile(path.join(packageRoot, source), 'utf8')
    assert.equal(unpack(destination), original, `Packed notice differs from its original: ${destination}`)
  }
}

export async function assertSourceLicense(item, sourceRoot) {
  assert.equal(item.meta?.license, 'MIT', 'Registry item must declare its first-party MIT license')
  const license = item.files.find(file => path.basename(file.path) === 'LICENSE')
  assert.ok(license, 'Source delivery must include LICENSE')
  assert.equal(license.content, await readFile(path.join(sourceRoot, 'LICENSE'), 'utf8'))
  const notices = item.files.filter(entry => /(?:THIRD_PARTY_NOTICES\.md|third-party\/.*\.txt)$/.test(entry.path))
  for (const file of notices) {
    const original = await readFile(path.join(sourceRoot, file.path), 'utf8')
    assert.equal(file.content, original, `Source delivery changed an original notice: ${file.path}`)
  }
}

export async function retainBundledLicenses(consumer, supplementalNotices = {}) {
  const output = path.join(consumer, 'dist/client/THIRD_PARTY_LICENSES.json')
  const entries = JSON.parse(await readFile(output, 'utf8'))
  assert.ok(entries.length > 0, 'Client bundles must retain original dependency notices')
  const store = path.join(consumer, 'node_modules/.pnpm')
  const installed = await readdir(store)
  for (const entry of entries) {
    const prefix = `${entry.name.replaceAll('/', '+')}@${entry.version}`
    const folder = installed.find(name => name === prefix || name.startsWith(`${prefix}_`))
    const root = folder
      ? path.join(store, folder, 'node_modules', entry.name)
      : path.join(consumer, 'node_modules', entry.name)
    const manifest = JSON.parse(await readFile(path.join(root, 'package.json'), 'utf8'))
    assert.equal(entry.name, manifest.name)
    assert.equal(entry.version, manifest.version, entry.name)
    assert.equal(entry.identifier, manifest.license, entry.name)
    const files = (await readdir(root)).filter(name => /^(?:licen[cs]e|copying)/i.test(name))
    const texts = await Promise.all(files.map(async file => (await readFile(path.join(root, file), 'utf8')).trim()))
    if (!entry.text && supplementalNotices[`${entry.name}@${entry.version}`]) {
      entry.noticeFile = supplementalNotices[`${entry.name}@${entry.version}`]
      entry.text = (await readFile(path.join(consumer, entry.noticeFile), 'utf8')).trim()
      texts.push(entry.text)
    }
    assert.ok(entry.text, `Bundled dependency has no original notice: ${entry.name}`)
    assert.ok(texts.includes(entry.text), `Bundled dependency notice changed: ${entry.name}`)
  }
  await writeFile(output, `${JSON.stringify(entries, null, 2)}\n`)
}
