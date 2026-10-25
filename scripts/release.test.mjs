import assert from 'node:assert/strict'
import { execFileSync, spawnSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { chmodSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { test } from 'node:test'
import { assetName, binaryName, hostTarget, releaseTag, repository, targets } from './release-contract.mjs'

const scripts = path.join(repository, 'scripts')
const identity = path.join(scripts, 'release-identity.sh')
const verify = path.join(scripts, 'verify-release.mjs')
const tag = 'v9.9.9'

function writeBinary(directory, target, version = tag, mode = 0o755) {
  mkdirSync(directory, { recursive: true })
  const binary = path.join(directory, binaryName(target.goos))
  if (target.goos === 'windows') {
    writeFileSync(binary, 'synthetic windows executable')
    return binary
  }
  writeFileSync(binary, `#!/bin/sh
case "$1" in
  version) echo ${version} ;;
  license) echo 'MIT License' ;;
esac
`)
  chmodSync(binary, mode)
  return binary
}

function pack(directory, dist, target) {
  execFileSync('bash', [path.join(scripts, 'package-cli.sh'), directory, path.join(dist, assetName(tag, target))])
}

function writeChecksums(dist) {
  const lines = targets.map(target => assetName(tag, target)).sort().map((name) => {
    const hash = createHash('sha256').update(readFileSync(path.join(dist, name))).digest('hex')
    return `${hash}  ${name}`
  })
  writeFileSync(path.join(dist, 'SHA256SUMS'), `${lines.join('\n')}\n`)
}

function buildDist(root, { license = true, mode = 0o755 } = {}) {
  const dist = path.join(root, 'dist')
  mkdirSync(dist, { recursive: true })
  for (const target of targets) {
    const directory = path.join(root, 'build', `${target.goos}-${target.goarch}`)
    writeBinary(directory, target, tag, mode)
    pack(directory, dist, target)
    if (!license) {
      const archive = path.join(dist, assetName(tag, target))
      rmSync(archive)
      if (target.goos === 'windows') {
        execFileSync('zip', ['-q', '-j', archive, path.join(directory, 'zt.exe')])
      }
      else {
        execFileSync('tar', ['-czf', archive, '-C', directory, 'zt'])
      }
    }
  }
  writeChecksums(dist)
  return dist
}

function runVerify(dist, extra = []) {
  return spawnSync(process.execPath, [verify, '--dist', dist, '--tag', tag, ...extra], { encoding: 'utf8' })
}

function withDist(options, callback) {
  const root = mkdtempSync(path.join(os.tmpdir(), 'zt-release-test-'))
  try {
    return callback(buildDist(root, options), root)
  }
  finally {
    rmSync(root, { recursive: true, force: true })
  }
}

test('release contract keeps six native archives', async () => {
  assert.match('v0.1.0', releaseTag)
  assert.doesNotMatch('0.1.0', releaseTag)
  assert.deepEqual(targets.map(target => assetName('v0.1.0', target)), [
    'zt_v0.1.0_darwin_amd64.tar.gz',
    'zt_v0.1.0_darwin_arm64.tar.gz',
    'zt_v0.1.0_linux_amd64.tar.gz',
    'zt_v0.1.0_linux_arm64.tar.gz',
    'zt_v0.1.0_windows_amd64.zip',
    'zt_v0.1.0_windows_arm64.zip',
  ])
  for (const target of targets) {
    assert.equal(binaryName(target.goos), target.goos === 'windows' ? 'zt.exe' : 'zt')
  }
})

test('release workflow builds each target on its own operating system', async () => {
  const workflow = readFileSync(path.join(repository, '.github/workflows/release.yml'), 'utf8')
  for (const target of targets) {
    assert.ok(workflow.includes(`runner: ${target.runner}`), `${target.runner} builds ${target.goos}/${target.goarch}`)
    assert.ok(workflow.includes('scripts/release-identity.sh'), 'release embeds the resolved client identity')
  }
  assert.equal((workflow.match(/secrets\./g) ?? []).length, 1, 'only the client identity secret reaches the workflow')
  assert.ok(workflow.includes('internal/keystore'), 'native credential store gate is part of the release')
  assert.ok(workflow.includes('TestOSStoreIntegration'), 'the gate runs the real integration test')
  assert.ok(workflow.includes("ZT_KEYSTORE_INTEGRATION: '1'"), 'the gate enables the integration test')
  assert.ok(workflow.includes('needs: [package, keystore]'), 'publish waits for the credential gate')
  assert.ok(workflow.includes('scripts/check-cli-license.sh'), 'license output is captured before matching')
  assert.doesNotMatch(workflow, /license"?\s*\|\s*grep/, 'never pipe zt license into a short-circuiting reader')
  assert.ok(!workflow.includes('pull_request_target'), 'release never runs from an untrusted event')
})

test('release identity accepts only GitHub App client IDs and never prints a rejected value', async () => {
  const resolve = variables => spawnSync('bash', [identity], { encoding: 'utf8', env: { PATH: process.env.PATH, ...variables } })

  assert.deepEqual(resolve({ ZEITHROLD_GITHUB_CLIENT_ID: 'Iv23AbCdEfGh1234' }).stdout, 'Iv23AbCdEfGh1234\n')
  assert.deepEqual(resolve({ ZEITHROLD_GITHUB_CLIENT_ID: 'Iv1.0123456789abcdef' }).stdout, 'Iv1.0123456789abcdef\n')
  assert.deepEqual(resolve({ ZEITHROLD_GITHUB_CLIENT_ID: 'Iv23AbCdEfGh1234\n' }).stdout, 'Iv23AbCdEfGh1234\n')

  const numeric = resolve({ ZEITHROLD_GITHUB_CLIENT_ID: '12345678' })
  assert.notEqual(numeric.status, 0)
  assert.match(numeric.stderr, /ZEITHROLD_GITHUB_CLIENT_ID/)
  assert.ok(!numeric.stderr.includes('12345678'))
  assert.equal(numeric.stdout, '')

  const oauth = resolve({ ZEITHROLD_GITHUB_CLIENT_ID: 'Ov23liAbCdEf123456' })
  assert.notEqual(oauth.status, 0)
  assert.ok(!oauth.stderr.includes('Ov23liAbCdEf123456'))

  const legacy = resolve({ ZEITHROLD_GITHUB_APP_ID: 'Iv23AbCdEfGh1234' })
  assert.notEqual(legacy.status, 0)
  assert.match(legacy.stderr, /ZEITHROLD_GITHUB_CLIENT_ID/)
  assert.equal(legacy.stdout, '')

  const missing = resolve({})
  assert.notEqual(missing.status, 0)
  assert.match(missing.stderr, /ZEITHROLD_GITHUB_CLIENT_ID/)
})

test('verify-release accepts a complete release and runs the native archive', () => withDist({}, (dist) => {
  const result = runVerify(dist)
  assert.equal(result.status, 0, result.stderr)
  assert.match(result.stdout, /verified 6 assets/)
  assert.match(result.stdout, /native run: /)
  assert.doesNotMatch(result.stdout, /native run: skipped/)
}))

test('verify-release rejects a tampered archive', () => withDist({}, (dist) => {
  const asset = path.join(dist, assetName(tag, targets[2]))
  writeFileSync(asset, Buffer.concat([readFileSync(asset), Buffer.from('tampered')]))
  const result = runVerify(dist)
  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /sha256/)
  return result
}))

test('verify-release rejects an unexpected extra asset', () => withDist({}, (dist) => {
  writeFileSync(path.join(dist, 'zt_v9.9.9_linux_amd64.tar.gz.sig'), 'extra')
  const result = runVerify(dist)
  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /exactly/)
  return result
}))

test('verify-release rejects an archive without the license files', () => withDist({ license: false }, (dist) => {
  const result = runVerify(dist)
  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /LICENSE/)
  return result
}))

test('verify-release rejects a host binary that reports a different version', () => {
  const host = hostTarget()
  return withDist({}, (dist, root) => {
    if (!host) {
      return
    }
    const directory = path.join(root, 'build', `${host.goos}-${host.goarch}`)
    writeBinary(directory, host, 'v0.0.0')
    pack(directory, dist, host)
    writeChecksums(dist)
    const result = runVerify(dist)
    assert.notEqual(result.status, 0)
    assert.match(result.stderr, /expected v9\.9\.9/)
    return result
  })
})

test('package-cli restores the executable bit lost by artifact transfer', () => withDist({ mode: 0o644 }, (dist) => {
  const result = runVerify(dist)
  assert.equal(result.status, 0, result.stderr)
  return result
}))

test('license check captures the full output instead of piping it', () => {
  const root = mkdtempSync(path.join(os.tmpdir(), 'zt-license-check-'))
  try {
    const binary = path.join(root, 'zt')
    writeFileSync(binary, `#!/bin/sh
case "$1" in
  version) echo v9.9.9 ;;
  license)
    printf 'MIT License\\n'
    i=0
    while [ "$i" -lt 4000 ]; do printf 'notice %s\\n' "$i"; i=$((i+1)); done
    ;;
esac
`)
    chmodSync(binary, 0o755)
    execFileSync('bash', [path.join(scripts, 'check-cli-license.sh'), binary])

    const broken = path.join(root, 'zt-broken')
    writeFileSync(broken, '#!/bin/sh\necho no-license\n')
    chmodSync(broken, 0o755)
    assert.throws(() => execFileSync('bash', [path.join(scripts, 'check-cli-license.sh'), broken]))
  }
  finally {
    rmSync(root, { recursive: true, force: true })
  }
})

const miseInstalled = spawnSync('mise', ['--version']).status === 0

test('mise fixture builds only the host archive and labels the rest', {
  skip: miseInstalled ? false : 'mise is not installed',
  timeout: 240000,
}, () => {
  const root = mkdtempSync(path.join(os.tmpdir(), 'zt-mise-build-'))
  try {
    execFileSync(process.execPath, [
      path.join(scripts, 'verify-mise-release.mjs'),
      '--build',
      '--dist', path.join(root, 'dist'),
      '--receipt', path.join(root, 'receipt.json'),
    ], { encoding: 'utf8', timeout: 240000 })
    const receipt = JSON.parse(readFileSync(path.join(root, 'receipt.json'), 'utf8'))
    assert.deepEqual(receipt.builtTargets, [receipt.host])
    assert.equal(receipt.placeholderAssets.length, 5)
    assert.ok(receipt.placeholderAssets.every(name => name.startsWith('zt_')))
  }
  finally {
    rmSync(root, { recursive: true, force: true })
  }
})
