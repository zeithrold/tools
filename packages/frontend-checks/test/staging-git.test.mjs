import assert from 'node:assert/strict'
import { execFileSync, spawnSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { mkdtemp, readFile, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import process from 'node:process'
import test from 'node:test'

const attachment = path.resolve('../../scripts/attach-publish-main.sh')
const packageName = '@ztd-me/staging-regression-fixture'

function run(directory, command, args, env = process.env) {
  return execFileSync(command, args, {
    cwd: directory,
    env,
    encoding: 'utf8',
    stdio: [
      'ignore',
      'pipe',
      'pipe',
    ],
    timeout: 30_000,
  }).trim()
}

async function fixture() {
  const root = await mkdtemp(path.join(os.tmpdir(), 'ztd-publish-git-'))
  const remote = path.join(root, 'remote.git')
  const seed = path.join(root, 'seed')
  const checkout = path.join(root, 'checkout')
  run(root, 'git', [
    'init',
    '--bare',
    remote,
  ])
  run(root, 'git', [
    'init',
    '--initial-branch=main',
    seed,
  ])
  await writeFile(path.join(seed, 'package.json'), JSON.stringify({
    name: packageName,
    version: '0.0.0',
    packageManager: 'pnpm@11.22.0',
    publishConfig: { access: 'public', registry: 'http://127.0.0.1:9/' },
  }))
  run(seed, 'git', ['add', 'package.json'])
  commit(seed, 'fixture')
  run(seed, 'git', [
    'remote',
    'add',
    'origin',
    remote,
  ])
  run(seed, 'git', [
    'push',
    'origin',
    'main',
  ])
  const sha = run(seed, 'git', ['rev-parse', 'HEAD'])
  run(root, 'git', ['init', checkout])
  run(checkout, 'git', [
    'remote',
    'add',
    'origin',
    remote,
  ])
  run(checkout, 'git', [
    'fetch',
    '--depth=1',
    'origin',
    sha,
  ])
  run(checkout, 'git', [
    'checkout',
    '--detach',
    sha,
  ])
  return { root, seed, checkout, sha }
}

function environment(sha) {
  const env = {
    ...process.env,
    GITHUB_SHA: sha,
    GITHUB_REF: 'refs/heads/main',
    GITHUB_ACTIONS: 'false',
  }
  delete env.NPM_ID_TOKEN
  delete env.ACTIONS_ID_TOKEN_REQUEST_URL
  delete env.ACTIONS_ID_TOKEN_REQUEST_TOKEN
  return env
}

function commit(directory, message, allowEmpty = false) {
  const args = [
    '-c',
    'user.name=Fixture',
    '-c',
    'user.email=fixture@example.invalid',
    'commit',
    '-m',
    message,
  ]
  if (allowEmpty) {
    args.push('--allow-empty')
  }
  run(directory, 'git', args)
}

test('pnpm staging rejects detached HEAD and preserves bytes on immutable main', async () => {
  const { root, checkout, sha } = await fixture()
  const env = environment(sha)
  const archive = path.join(root, 'package.tgz')
  run(checkout, 'pnpm', [
    'pack',
    '--out',
    archive,
  ], env)
  const bytes = await readFile(archive)
  const args = [
    'stage',
    'publish',
    archive,
    '--access',
    'public',
    '--json',
    '--publish-branch',
    'main',
    '--dry-run',
  ]
  const detached = spawnSync('pnpm', args, { cwd: checkout, env, encoding: 'utf8', timeout: 30_000 })
  assert.equal(detached.status, 1)
  assert.equal(JSON.parse(detached.stdout).error.code, 'ERR_PNPM_GIT_UNKNOWN_BRANCH')
  run(checkout, 'bash', [attachment], env)
  assert.equal(run(checkout, 'git', ['rev-parse', 'HEAD']), sha)
  assert.equal(run(checkout, 'git', ['branch', '--show-current']), 'main')
  assert.equal(run(checkout, 'git', ['rev-parse', '@{u}']), sha)
  const receipt = JSON.parse(run(checkout, 'pnpm', args, env))[packageName]
  assert.equal(receipt.integrity, `sha512-${createHash('sha512').update(bytes).digest('base64')}`)
  assert.equal(receipt.stageId, undefined, 'Dry-run does not create a registry stage')
  assert.deepEqual(await readFile(archive), bytes)
  await writeFile(path.join(checkout, 'uncommitted'), 'dirty fixture')
  const dirty = spawnSync('pnpm', args, { cwd: checkout, env, encoding: 'utf8', timeout: 30_000 })
  assert.equal(JSON.parse(dirty.stdout).error.code, 'ERR_PNPM_GIT_UNCLEAN')
})

test('branch attachment rejects non-main events, source mismatch and an advanced remote', async () => {
  const { seed, checkout, sha } = await fixture()
  const env = environment(sha)
  assert.throws(() => run(checkout, 'bash', [
    attachment,
  ], { ...env, GITHUB_REF: 'refs/heads/other' }), /Staging requires a push/u)
  assert.throws(() => run(checkout, 'bash', [attachment], { ...env, GITHUB_SHA: 'wrong-sha' }))
  commit(seed, 'remote advanced', true)
  run(seed, 'git', [
    'push',
    'origin',
    'main',
  ])
  assert.throws(() => run(checkout, 'bash', [attachment], env))
  assert.equal(run(checkout, 'git', ['rev-parse', 'HEAD']), sha)
  assert.equal(run(checkout, 'git', ['branch', '--show-current']), '')
})
