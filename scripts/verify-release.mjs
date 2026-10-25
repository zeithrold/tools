import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { existsSync, mkdtempSync, readFileSync, readdirSync, rmSync, writeFileSync } from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { parseArgs } from 'node:util'
import { archiveFiles, assetName, binaryName, hostTarget, releaseTag, repository, targets, thirdPartyNotices } from './release-contract.mjs'

const { values } = parseArgs({
  options: {
    dist: { type: 'string', default: 'dist' },
    tag: { type: 'string' },
    'skip-run': { type: 'boolean', default: false },
    receipt: { type: 'string' },
  },
})

const problems = []
const fail = message => problems.push(message)

const dist = path.resolve(values.dist)
const licenseFiles = [...archiveFiles, ...thirdPartyNotices()]
const tag = values.tag
if (!tag || !releaseTag.test(tag)) {
  console.error('verify-release: --tag must be a semantic version tag such as v0.1.0')
  process.exit(2)
}
if (!existsSync(dist)) {
  console.error(`verify-release: ${dist} does not exist`)
  process.exit(2)
}

function digest(file) {
  return createHash('sha256').update(readFileSync(file)).digest('hex')
}

function entries(archive) {
  return archive.endsWith('.zip')
    ? execFileSync('unzip', ['-Z1', archive], { encoding: 'utf8' }).split('\n').filter(Boolean)
    : execFileSync('tar', ['-tzf', archive], { encoding: 'utf8' }).split('\n').filter(Boolean)
}

function extract(archive, directory) {
  if (archive.endsWith('.zip')) {
    execFileSync('unzip', ['-q', archive, '-d', directory])
    return
  }
  execFileSync('tar', ['-xzf', archive, '-C', directory])
}

function executable(archive, name) {
  if (archive.endsWith('.zip')) {
    // ZIP has no reliable executable bit; mise restores it for extracted tools.
    return true
  }
  for (const line of execFileSync('tar', ['-tvzf', archive], { encoding: 'utf8' }).split('\n')) {
    if (!line.trim()) {
      continue
    }
    const fields = line.trim().split(/\s+/)
    const mode = fields[0]
    const entry = fields[fields.length - 1]
    if (entry === name) {
      return mode.length === 10 && mode[3] === 'x'
    }
  }
  return false
}

const expected = targets.map(target => assetName(tag, target)).sort()
const actual = readdirSync(dist).filter(name => name !== 'SHA256SUMS').sort()
try {
  assert.deepEqual(actual, expected)
}
catch {
  fail(`release assets must be exactly ${expected.join(', ')}; found ${actual.join(', ')}`)
}

const checksumFile = path.join(dist, 'SHA256SUMS')
const checksums = new Map()
try {
  for (const line of readFileSync(checksumFile, 'utf8').split('\n')) {
    if (!line) {
      continue
    }
    const match = /^([0-9a-f]{64}) {2}([^/\\]+)$/.exec(line)
    if (!match) {
      fail(`SHA256SUMS has an unexpected line: ${line}`)
      continue
    }
    if (checksums.has(match[2])) {
      fail(`SHA256SUMS lists ${match[2]} more than once`)
    }
    checksums.set(match[2], match[1])
  }
}
catch {
  fail('SHA256SUMS is missing')
}
const listed = [...checksums.keys()]
try {
  assert.deepEqual(listed, [...listed].sort())
}
catch {
  fail('SHA256SUMS must list assets in byte order (LC_ALL=C)')
}

for (const name of actual) {
  const file = path.join(dist, name)
  if (!checksums.has(name)) {
    fail(`SHA256SUMS does not list ${name}`)
    continue
  }
  const actualDigest = digest(file)
  if (actualDigest !== checksums.get(name)) {
    fail(`${name}: sha256 is ${actualDigest}, SHA256SUMS says ${checksums.get(name)}`)
  }
}
if (actual.length && checksums.size !== actual.length) {
  fail(`SHA256SUMS lists ${checksums.size} entries for ${actual.length} assets`)
}

const runTarget = values['skip-run'] ? undefined : hostTarget()
let nativeRun = 'skipped'

for (const target of targets) {
  const name = assetName(tag, target)
  const archive = path.join(dist, name)
  if (!actual.includes(name)) {
    continue
  }
  const wanted = [binaryName(target.goos), ...licenseFiles].sort()
  const found = entries(archive)
  try {
    assert.deepEqual([...found].sort(), wanted)
  }
  catch {
    fail(`${name}: archive must contain exactly ${wanted.join(', ')}; found ${found.join(', ')}`)
    continue
  }
  for (const entry of found) {
    if (path.isAbsolute(entry) || entry.split(/[/\\]/).includes('..')) {
      fail(`${name}: unsafe archive entry ${entry}`)
    }
  }
  if (target.goos !== 'windows' && !executable(archive, binaryName(target.goos))) {
    fail(`${name}: ${binaryName(target.goos)} is not executable inside the archive`)
  }

  const directory = mkdtempSync(path.join(os.tmpdir(), 'zt-release-verify-'))
  try {
    extract(archive, directory)
    for (const file of licenseFiles) {
      const shipped = readFileSync(path.join(directory, file))
      const canonical = readFileSync(path.join(repository, file))
      if (!shipped.equals(canonical)) {
        fail(`${name}: ${file} does not match the repository copy`)
      }
    }
    if (runTarget === target) {
      const binary = path.join(directory, binaryName(target.goos))
      const version = execFileSync(binary, ['version'], { encoding: 'utf8' }).trim()
      if (version !== tag) {
        fail(`${name}: zt version printed ${JSON.stringify(version)}, expected ${tag}`)
      }
      if (!execFileSync(binary, ['license'], { encoding: 'utf8' }).includes('MIT License')) {
        fail(`${name}: zt license does not include the MIT license text`)
      }
      nativeRun = `${target.goos}/${target.goarch}`
    }
  }
  catch (error) {
    fail(`${name}: ${error.message}`)
  }
  finally {
    rmSync(directory, { recursive: true, force: true })
  }
}

if (problems.length > 0) {
  for (const problem of problems) {
    console.error(`verify-release: ${problem}`)
  }
  process.exit(1)
}

const receipt = { tag, assets: expected, checksums: 'verified', nativeRun }
const summary = `verified ${expected.length} assets, SHA256SUMS, native run: ${nativeRun}`
if (values.receipt) {
  writeFileSync(values.receipt, `${JSON.stringify(receipt, null, 2)}\n`)
}
console.log(`verify-release: ${summary}`)
