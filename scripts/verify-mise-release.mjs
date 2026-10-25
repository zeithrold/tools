// Verify that mise's github backend accepts the published asset contract.
//
// The script serves a local GitHub Enterprise-style API fixture over loopback,
// installs into isolated mise data directories, and never creates a release,
// tags a commit or touches the user's real mise installation.
import assert from 'node:assert/strict'
import { execFileSync, spawn } from 'node:child_process'
import { createHash } from 'node:crypto'
import { createServer } from 'node:http'
import { existsSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { parseArgs } from 'node:util'
import { archiveFiles, assetName, binaryName, hostTarget, repository, targets, thirdPartyNotices } from './release-contract.mjs'

const { values } = parseArgs({
  options: {
    dist: { type: 'string', default: '.artifacts/mise-fixture/dist' },
    tag: { type: 'string', default: 'v0.1.0' },
    build: { type: 'boolean', default: false },
    receipt: { type: 'string', default: '.artifacts/mise-fixture/receipt.json' },
  },
})

const tag = values.tag
const version = tag.replace(/^v/, '')
const dist = path.resolve(values.dist)
const host = hostTarget()
if (!host) {
  console.error('verify-mise-release: unsupported host platform')
  process.exit(2)
}

function build() {
  mkdirSync(dist, { recursive: true })
  for (const target of targets) {
    const directory = path.join(dist, '..', 'build', `${target.goos}-${target.goarch}`)
    mkdirSync(directory, { recursive: true })
    const binary = path.join(directory, binaryName(target.goos))
    if (target === host) {
      execFileSync('go', [
        'build',
        '-trimpath',
        `-ldflags=-s -w -X github.com/zeithrold/tools.releaseVersion=${tag}`,
        '-o', binary,
        './cmd/zt',
      ], {
        cwd: repository,
        env: { ...process.env, GOOS: target.goos, GOARCH: target.goarch, CGO_ENABLED: target.cgo },
      })
    }
    else {
      // Only the host archive needs to be real: this fixture verifies mise's
      // host selection, checksum and execution path, not cross-platform builds.
      writeFileSync(binary, `synthetic ${target.goos}/${target.goarch} placeholder; not a native build\n`)
    }
    execFileSync('bash', [path.join(repository, 'scripts/package-cli.sh'), directory, path.join(dist, assetName(tag, target))])
  }
  const lines = targets.map(target => assetName(tag, target)).sort().map((name) => {
    const hash = createHash('sha256').update(readFileSync(path.join(dist, name))).digest('hex')
    return `${hash}  ${name}`
  })
  writeFileSync(path.join(dist, 'SHA256SUMS'), `${lines.join('\n')}\n`)
}

if (values.build) {
  build()
}
for (const target of targets) {
  assert.ok(existsSync(path.join(dist, assetName(tag, target))), `missing ${assetName(tag, target)}; run with --build`)
}
assert.ok(existsSync(path.join(dist, 'SHA256SUMS')), 'missing SHA256SUMS')

function runMise(args, cwd) {
  return new Promise((resolve) => {
    const env = { ...process.env }
    for (const name of ['GITHUB_TOKEN', 'GH_TOKEN', 'MISE_GITHUB_TOKEN']) {
      delete env[name]
    }
    const child = spawn('mise', args, {
      cwd,
      env: {
        ...env,
        MISE_DATA_DIR: path.join(cwd, 'data'),
        MISE_CONFIG_DIR: path.join(cwd, 'config'),
        MISE_STATE_DIR: path.join(cwd, 'state'),
        MISE_CACHE_DIR: path.join(cwd, 'cache'),
        MISE_YES: '1',
        MISE_TRUSTED_CONFIG_PATHS: cwd,
        NO_COLOR: '1',
      },
      stdio: ['ignore', 'pipe', 'pipe'],
    })
    let stdout = ''
    let stderr = ''
    child.stdout.on('data', (chunk) => { stdout += chunk })
    child.stderr.on('data', (chunk) => { stderr += chunk })
    child.on('close', status => resolve({ status, stdout, stderr }))
  })
}

const requests = []
let tampered = false
const hostAsset = assetName(tag, host)

let base = ''
const server = createServer((request, response) => {
  const url = new URL(request.url, 'https://127.0.0.1')
  requests.push(url.pathname)
  if (url.pathname === '/repos/zeithrold/tools') {
    response.writeHead(200, { 'content-type': 'application/json' })
    response.end(JSON.stringify({ full_name: 'zeithrold/tools', name: 'tools', default_branch: 'main' }))
    return
  }
  if (url.pathname === '/repos/zeithrold/tools/releases' || url.pathname === '/repos/zeithrold/tools/releases/latest' || url.pathname === `/repos/zeithrold/tools/releases/tags/${tag}`) {
    const asset = name => ({
      id: targets.findIndex(target => assetName(tag, target) === name) + 1,
      name,
      size: statSync(path.join(dist, name)).size,
      content_type: name.endsWith('.zip') ? 'application/zip' : 'application/gzip',
      browser_download_url: `${base}/downloads/${name}`,
      url: `${base}/repos/zeithrold/tools/releases/assets/${name}`,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      download_count: 0,
      label: null,
    })
    const release = {
      id: 1,
      html_url: `${base}/zeithrold/tools/releases/tag/${tag}`,
      tag_name: tag,
      name: `zt ${tag}`,
      body: '',
      draft: false,
      prerelease: false,
      created_at: '2026-01-01T00:00:00Z',
      published_at: '2026-01-01T00:00:00Z',
      tarball_url: `${base}/zeithrold/tools/tarball/${tag}`,
      zipball_url: `${base}/zeithrold/tools/zipball/${tag}`,
      assets: [...targets.map(target => assetName(tag, target)), 'SHA256SUMS'].map(asset),
    }
    response.writeHead(200, { 'content-type': 'application/json' })
    response.end(JSON.stringify(url.pathname.endsWith('/releases') ? [release] : release))
    return
  }
  if (url.pathname.startsWith('/repos/zeithrold/tools/releases/assets/')) {
    const name = decodeURIComponent(url.pathname.slice('/repos/zeithrold/tools/releases/assets/'.length))
    const file = path.join(dist, name)
    if (!existsSync(file)) {
      response.writeHead(404, { 'content-type': 'application/json' })
      response.end('{}')
      return
    }
    let bytes = readFileSync(file)
    if (tampered && name === hostAsset) {
      bytes = Buffer.concat([bytes, Buffer.from('tampered')])
    }
    response.writeHead(200, { 'content-type': 'application/octet-stream', 'content-length': bytes.length })
    response.end(bytes)
    return
  }
  if (url.pathname.startsWith('/downloads/')) {
    const name = decodeURIComponent(url.pathname.slice('/downloads/'.length))
    let bytes = readFileSync(path.join(dist, name))
    if (tampered && name === hostAsset) {
      bytes = Buffer.concat([bytes, Buffer.from('tampered')])
    }
    response.writeHead(200, { 'content-type': 'application/octet-stream', 'content-length': bytes.length })
    response.end(bytes)
    return
  }
  response.writeHead(404, { 'content-type': 'application/json' })
  response.end('{}')
})
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
base = `http://127.0.0.1:${server.address().port}`

function config(root) {
  mkdirSync(root, { recursive: true })
  writeFileSync(path.join(root, 'mise.toml'), `[tools]\n"github:zeithrold/tools" = { version = "${version}", matching = "zt_", api_url = "${base}" }\n`)
}

const root = mkdtempSync(path.join(os.tmpdir(), 'zt-mise-fixture-'))
try {
  config(root)
  const first = await runMise(['install'], root)
  assert.equal(first.status, 0, `${first.stdout}\n${first.stderr}`)

  const run = await runMise(['exec', '--', 'zt', 'version'], root)
  assert.equal(run.status, 0, `${run.stdout}\n${run.stderr}`)
  assert.equal(run.stdout.trim(), tag, 'mise executes the installed zt from PATH')

  const installed = []
  const walk = (directory) => {
    for (const entry of readdirSync(directory, { withFileTypes: true })) {
      const file = path.join(directory, entry.name)
      if (entry.isDirectory()) {
        walk(file)
      }
      else if (entry.name === 'zt' || entry.name === 'zt.exe') {
        installed.push(file)
      }
    }
  }
  walk(path.join(root, 'data', 'installs'))
  assert.equal(installed.length, 1, 'mise installed exactly one zt executable')
  const installedDirectory = path.dirname(installed[0])
  for (const file of [...archiveFiles, ...thirdPartyNotices()]) {
    assert.ok(existsSync(path.join(installedDirectory, file)), `${file} ships beside the installed binary`)
  }

  const downloads = requests
    .filter(request => request.startsWith('/downloads/') || request.startsWith('/repos/zeithrold/tools/releases/assets/'))
    .map(request => decodeURIComponent(request.split('/').at(-1)))
  const downloadedArchives = downloads.filter(name => name.startsWith('zt_'))
  assert.deepEqual(downloadedArchives, [hostAsset], 'mise downloads only the host archive')

  tampered = true
  const rejectedRoot = mkdtempSync(path.join(os.tmpdir(), 'zt-mise-fixture-tampered-'))
  try {
    config(rejectedRoot)
    const rejected = await runMise(['install'], rejectedRoot)
    assert.notEqual(rejected.status, 0, 'mise rejects an archive that does not match SHA256SUMS')
    assert.match(`${rejected.stdout}\n${rejected.stderr}`.toLowerCase(), /sha256|checksum/)
  }
  finally {
    rmSync(rejectedRoot, { recursive: true, force: true })
  }

  const receipt = {
    mise: execFileSync('mise', ['--version'], { encoding: 'utf8' }).trim(),
    tag,
    host: `${host.goos}/${host.goarch}`,
    selectedAsset: hostAsset,
    downloadedAssets: downloadedArchives,
    installedBinary: path.relative(root, installed[0]),
    checksumRejected: true,
    ...(values.build
      ? {
          builtTargets: [`${host.goos}/${host.goarch}`],
          placeholderAssets: targets.filter(target => target !== host).map(target => assetName(tag, target)),
        }
      : {}),
  }
  mkdirSync(path.dirname(path.resolve(values.receipt)), { recursive: true })
  writeFileSync(path.resolve(values.receipt), `${JSON.stringify(receipt, null, 2)}\n`)
  console.log(`verify-mise-release: ${host.goos}/${host.goarch} selected ${hostAsset}; SHA256SUMS mismatch rejected`)
}
finally {
  await new Promise(resolve => server.close(resolve))
  rmSync(root, { recursive: true, force: true })
}
