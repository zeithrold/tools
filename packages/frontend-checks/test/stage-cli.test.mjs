import assert from 'node:assert/strict'
import { execFileSync, spawnSync } from 'node:child_process'
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import process from 'node:process'
import test from 'node:test'

test('stage CLI rejects mismatched package identity and source before authentication', async () => {
  const packageInfo = JSON.parse(await readFile('package.json', 'utf8'))
  const directory = await mkdtemp(path.join(os.tmpdir(), 'ztd-stage-identity-'))
  const packed = path.join(directory, 'package')
  await mkdir(packed)
  const archive = path.join(directory, 'package.tgz')
  for (const [change, expectedError] of [
    [
      { name: '@ztd-me/other' },
      '@ztd-me/other',
    ],
    [
      { version: '99.0.0' },
      '99.0.0',
    ],
    [
      { private: true },
      'packedInfo.private',
    ],
    [
      {},
      'Stage only the immutable automatic workflow source',
    ],
  ]) {
    await writeFile(path.join(packed, 'package.json'), JSON.stringify({ ...packageInfo, ...change }))
    execFileSync('tar', [
      '-czf',
      archive,
      '-C',
      directory,
      'package',
    ])
    const result = spawnSync('node', ['scripts/stage.mjs', archive], {
      encoding: 'utf8',
      env: { ...process.env, GITHUB_SHA: 'not-the-source-commit', NPM_TOKEN: '' },
    })
    assert.equal(result.status, 1)
    assert.ok(result.stderr.includes(expectedError), result.stderr)
    assert.ok(!result.stderr.includes('NPM_TOKEN'), 'Artifact/source rejection precedes token lookup')
  }
})
