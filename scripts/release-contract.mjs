import path from 'node:path'
import { readdirSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

export const repository = fileURLToPath(new URL('../', import.meta.url))

export const releaseTag = /^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/

// Each target is built and executed on the runner of its own operating system,
// so darwin can link the native SDK and no target is reported from cross-compilation.
export const targets = [
  { goos: 'darwin', goarch: 'amd64', archive: 'tar.gz', runner: 'macos-26-intel', cgo: '1' },
  { goos: 'darwin', goarch: 'arm64', archive: 'tar.gz', runner: 'macos-26', cgo: '1' },
  { goos: 'linux', goarch: 'amd64', archive: 'tar.gz', runner: 'ubuntu-24.04', cgo: '0' },
  { goos: 'linux', goarch: 'arm64', archive: 'tar.gz', runner: 'ubuntu-24.04-arm', cgo: '0' },
  { goos: 'windows', goarch: 'amd64', archive: 'zip', runner: 'windows-2025', cgo: '0' },
  { goos: 'windows', goarch: 'arm64', archive: 'zip', runner: 'windows-11-arm', cgo: '0' },
]

// The archive root deliberately keeps the license next to the executable.
export const archiveFiles = ['LICENSE', 'THIRD_PARTY_NOTICES.md']

// Preserve the manually reviewed notices; inventory copying does not establish
// that a new dependency has been reviewed. See docs/licensing.md.
export function thirdPartyNotices(root = repository) {
  const base = path.join(root, 'third-party')
  const walk = relative => readdirSync(path.join(base, relative), { withFileTypes: true })
    .flatMap(entry => entry.isDirectory()
      ? walk(path.posix.join(relative, entry.name))
      : [path.posix.join(relative, entry.name)])
  return walk('').map(file => `third-party/${file}`).sort()
}

export function binaryName(goos) {
  return goos === 'windows' ? 'zt.exe' : 'zt'
}

export function assetName(tag, target) {
  return `zt_${tag}_${target.goos}_${target.goarch}.${target.archive}`
}

export function targetFor(goos, goarch) {
  return targets.find(target => target.goos === goos && target.goarch === goarch)
}

export function hostTarget(platform = process.platform, arch = process.arch) {
  const goos = { darwin: 'darwin', linux: 'linux', win32: 'windows' }[platform]
  const goarch = { x64: 'amd64', arm64: 'arm64' }[arch]
  return goos && goarch ? targetFor(goos, goarch) : undefined
}

export function archivePath(dist, tag, target) {
  return path.join(dist, assetName(tag, target))
}
