import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { mkdir, mkdtemp, readdir, readFile, rm, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { assertSourceLicense } from './license-coverage.mjs'
import { thirdPartyNotices } from './release-contract.mjs'

const repository = fileURLToPath(new URL('../', import.meta.url))
const read = file => readFile(path.join(repository, file), 'utf8')

function assertReviewedModules(modules, declaration, notices) {
  const rows = new Map()
  for (const line of declaration.split('\n')) {
    const match = /^\| `([^`@]+@[^`@]+)` \|/u.exec(line)
    if (match) {
      rows.set(match[1], line)
    }
  }
  for (const module of modules) {
    const row = rows.get(module)
    assert.ok(row, `Linked module needs manual license review in docs/licensing.md: ${module}`)
    const files = [
      ...row.matchAll(/`(third-party\/[^`]+)`/gu),
    ].map(match => match[1])
    assert.ok(files.length > 0, `No retained original mapped for ${module}`)
    for (const file of files) {
      assert.ok(notices.has(file), `Missing retained original for ${module}: ${file}`)
    }
  }
}

test('linked host CLI modules have manually reviewed declarations and retained originals', async () => {
  const output = execFileSync('go', [
    'list',
    '-deps',
    '-f',
    '{{with .Module}}{{if .Version}}{{.Path}}@{{.Version}}{{end}}{{end}}',
    './cmd/zt',
  ], { cwd: repository, encoding: 'utf8' })
  const modules = new Set(output.split('\n').filter(Boolean))
  assert.ok(modules.size > 0, 'Native go list must discover linked dependencies')
  const declaration = await read('docs/licensing.md')
  const notices = new Set(thirdPartyNotices())
  assertReviewedModules(modules, declaration, notices)
  const unreviewed = new Set(modules).add('example.invalid/unreviewed@v1.0.0')
  assert.throws(
    () => assertReviewedModules(unreviewed, declaration, notices),
    /needs manual license review.*example\.invalid\/unreviewed@v1\.0\.0/u,
  )
  assert.throws(
    () => assertReviewedModules(modules, declaration, new Set()),
    /Missing retained original/u,
  )
})

test('font evidence retains manual originals outside the public source payload', async () => {
  for (const notice of [
    'NOTO-SANS-OFL.txt',
    'NOTO-SANS-SC-OFL.txt',
    'NOTO-SERIF-SC-OFL.txt',
    'NOTO-COLOR-EMOJI-OFL.txt',
  ]) {
    const original = await read(`packages/ui/test/font-notices/${notice}`)
    assert.match(original, /SIL OPEN FONT LICENSE Version 1\.1/u, notice)
  }
  const payload = JSON.parse(await read('registry/ui.json'))
  assert.ok(
    payload.files.every(file => !file.path.includes('/font-notices/')),
    'private originals stay out of UI source delivery',
  )
})

test('first-party packages and independently copied Skills retain complete MIT grants', async () => {
  const license = await read('LICENSE')
  assert.match(license, /^MIT License\n\nCopyright \(c\) 2026 Zeithrold\n/)
  for (const name of await readdir(path.join(repository, 'packages'))) {
    const base = `packages/${name}`
    const manifest = JSON.parse(await read(`${base}/package.json`))
    assert.equal(manifest.license, 'MIT', base)
    assert.ok(manifest.files.includes('LICENSE'), `${base}: include LICENSE explicitly`)
    assert.equal(await read(`${base}/LICENSE`), license, base)
  }
  for (const name of await readdir(path.join(repository, 'skills'))) {
    assert.equal(await read(`skills/${name}/LICENSE`), license, name)
    const skill = await read(`skills/${name}/SKILL.md`)
    assert.match(skill.split('\n---\n')[0], /\nlicense: MIT\n?$/, name)
  }
  assert.equal(await read('packages/ui/test/consumer/LICENSE'), license, 'UI fixture templates')
})

test('registry payload retains canonical source and all original upstream notices', async () => {
  const inventory = JSON.parse(await read('registry.json'))
  assert.deepEqual(JSON.parse(await read('registry/registry.json')), inventory)
  for (const item of inventory.items) {
    const payload = JSON.parse(await read(`registry/${item.name}.json`))
    assert.equal(payload.files.length, item.files.length)
    await assertSourceLicense(payload, repository)
    for (const file of item.files) {
      const delivered = payload.files.find(entry => entry.path === file.path)
      assert.ok(delivered, `Missing registry file: ${file.path}`)
      assert.equal(delivered.target, file.target)
      assert.equal(delivered.content, await read(file.path), file.path)
    }
    for (const notice of await readdir(path.join(repository, 'packages/ui/third-party'))) {
      assert.ok(payload.files.some(file => file.target === `@ui/ztd-me/third-party/${notice}`), notice)
    }
  }
})

test('all six CLI archive layouts preserve every manually retained notice', async () => {
  const root = await mkdtemp(path.join(os.tmpdir(), 'zt-license-archives-'))
  try {
    for (const platform of [
      'linux',
      'darwin',
      'windows',
    ]) {
      for (const arch of ['amd64', 'arm64']) {
        const directory = path.join(root, `${platform}-${arch}`)
        await mkdir(directory)
        const binary = platform === 'windows' ? 'zt.exe' : 'zt'
        await writeFile(path.join(directory, binary), 'synthetic archive fixture')
        const archive = `${directory}${platform === 'windows' ? '.zip' : '.tar.gz'}`
        execFileSync('bash', [
          path.join(repository, 'scripts/package-cli.sh'),
          directory,
          archive,
        ])
        const unpack = file => platform === 'windows'
          ? execFileSync('unzip', [
              '-p',
              archive,
              file,
            ], { encoding: 'utf8' })
          : execFileSync('tar', [
              '-xOf',
              archive,
              file,
            ], { encoding: 'utf8' })
        assert.equal(unpack(binary), 'synthetic archive fixture')
        for (const file of [
          'LICENSE',
          'THIRD_PARTY_NOTICES.md',
          ...thirdPartyNotices(),
        ]) {
          assert.equal(unpack(file), await read(file), `${platform}/${arch}: ${file}`)
        }
      }
    }
  }
  finally {
    await rm(root, { recursive: true, force: true })
  }
})
