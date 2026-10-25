import assert from 'node:assert/strict'
import { Buffer } from 'node:buffer'
import { mkdtemp, readdir, readFile, rm, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { test } from 'node:test'
import { captureState } from '../src/playwright.mjs'

const png = Buffer.from([
  0x89,
  0x50,
  0x4E,
  0x47,
])

async function fixture(t) {
  const outputDir = await mkdtemp(path.join(os.tmpdir(), 'ztd-attach-'))
  t.after(() => rm(outputDir, { recursive: true, force: true }))
  return outputDir
}

/** Mirrors the documented attach contract: the source is copied before attach resolves. */
function fakeTestInfo(outputDir, attach = async () => {}) {
  return {
    outputDir,
    attachments: [],
    outputPath(name) {
      return path.join(outputDir, name)
    },
    async attach(name, options) {
      await attach(name, options)
      this.attachments.push({
        name,
        contentType: options.contentType,
        source: options.path,
        body: options.body,
        content: options.path === undefined ? options.body : await readFile(options.path),
      })
    },
  }
}

function fakePage(calls) {
  return {
    async screenshot(options) {
      calls.push(options)
      await writeFile(options.path, png)
      return png
    },
  }
}

test('captureState screenshots once, attaches by path and removes only its own source', async (t) => {
  const outputDir = await fixture(t)
  await writeFile(path.join(outputDir, 'keep.txt'), 'caller evidence')
  const info = fakeTestInfo(outputDir)
  const calls = []
  await captureState(fakePage(calls), info, 'account-keyboard')

  assert.equal(calls.length, 1)
  assert.equal(calls[0].fullPage, true)
  const [attachment] = info.attachments
  assert.equal(attachment.name, 'account-keyboard')
  assert.equal(attachment.contentType, 'image/png')
  assert.deepEqual(attachment.content, png)
  assert.equal(path.dirname(attachment.source), outputDir)
  assert.deepEqual((await readdir(outputDir)).sort(), ['keep.txt'])
})

test('repeated labels in one test keep separate sources instead of overwriting', async (t) => {
  const outputDir = await fixture(t)
  const info = fakeTestInfo(outputDir)
  const calls = []
  const page = fakePage(calls)
  await captureState(page, info, 'account-keyboard')
  await captureState(page, info, 'account-keyboard')

  assert.equal(info.attachments.length, 2)
  assert.notEqual(info.attachments[0].source, info.attachments[1].source)
  assert.deepEqual(await readdir(outputDir), [])
})

test('traversal-looking and non-ASCII labels stay inside the test output directory', async (t) => {
  const outputDir = await fixture(t)
  const info = fakeTestInfo(outputDir)
  await captureState(fakePage([]), info, '../確認/..')

  const [attachment] = info.attachments
  assert.equal(path.dirname(attachment.source), outputDir)
  assert.equal(attachment.source.startsWith(`${outputDir}${path.sep}`), true)
  assert.equal(attachment.source.includes(`..${path.sep}`), false)
  assert.deepEqual(await readdir(outputDir), [])
})

test('an existing caller file with the same name is never overwritten', async (t) => {
  const outputDir = await fixture(t)
  const callerFile = path.join(outputDir, 'account-keyboard.png')
  await writeFile(callerFile, 'caller evidence')
  const info = fakeTestInfo(outputDir)
  const calls = []
  await captureState(fakePage(calls), info, 'account-keyboard')

  const [attachment] = info.attachments
  assert.equal(attachment.name, 'account-keyboard')
  assert.equal(attachment.source, path.join(outputDir, 'account-keyboard-2.png'))
  assert.deepEqual(attachment.content, png)
  assert.deepEqual(await readFile(callerFile), Buffer.from('caller evidence'))
  assert.deepEqual((await readdir(outputDir)).sort(), ['account-keyboard.png'])
})

test('portable source names do not use Windows reserved device names', async (t) => {
  const outputDir = await fixture(t)
  const info = fakeTestInfo(outputDir)
  const labels = [
    'CON',
    'aux.txt',
    'LPT9',
    'COM¹',
  ]
  for (const label of labels) {
    await captureState(fakePage([]), info, label)
  }
  for (const attachment of info.attachments) {
    assert.ok(path.basename(attachment.source).startsWith('state-'))
  }
  assert.deepEqual(info.attachments.map(item => item.name), labels)
})

test('the inline fallback starts above the path-attachment byte limit', async (t) => {
  const outputDir = await fixture(t)
  const info = fakeTestInfo(outputDir)
  const page = fakePage([])
  await captureState(page, info, 'a'.repeat(180))
  await captureState(page, info, 'b'.repeat(181))

  const [pathLabel] = info.attachments
  assert.ok(pathLabel.source)
  assert.equal(pathLabel.body, undefined)
  const [inlineLabel] = info.attachments.slice(1)
  assert.equal(inlineLabel.source, undefined)
  assert.deepEqual(inlineLabel.body, png)
  assert.deepEqual(await readdir(outputDir), [])
})

test('very long labels attach inline with their display name and clean their source', async (t) => {
  const outputDir = await fixture(t)
  const info = fakeTestInfo(outputDir)
  const page = fakePage([])
  const longAscii = 'a'.repeat(220)
  const longMultibyte = '確'.repeat(120)
  await captureState(page, info, longAscii)
  await captureState(page, info, longMultibyte)

  for (const label of [longAscii, longMultibyte]) {
    const attachment = info.attachments.find(item => item.name === label)
    assert.ok(attachment, 'display name kept intact')
    assert.equal(attachment.source, undefined)
    assert.deepEqual(attachment.body, png)
  }
  assert.deepEqual(await readdir(outputDir), [])
})

test('a failed inline attach keeps the source for diagnosis', async (t) => {
  const outputDir = await fixture(t)
  const info = fakeTestInfo(outputDir, async () => {
    throw new Error('attach failed')
  })

  await assert.rejects(captureState(fakePage([]), info, 'a'.repeat(220)), /attach failed/)
  const sources = await readdir(outputDir)
  assert.equal(sources.length, 1)
  assert.deepEqual(await readFile(path.join(outputDir, sources[0])), png)
})

test('a failed attach keeps the source for diagnosis and propagates the failure', async (t) => {
  const outputDir = await fixture(t)
  const info = fakeTestInfo(outputDir, async () => {
    throw new Error('attach failed')
  })

  await assert.rejects(captureState(fakePage([]), info, 'account'), /attach failed/)
  assert.deepEqual(await readdir(outputDir), ['account.png'])
  assert.deepEqual(await readFile(path.join(outputDir, 'account.png')), png)
})
