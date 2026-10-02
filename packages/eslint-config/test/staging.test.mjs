import assert from 'node:assert/strict'
import test from 'node:test'
import { stagePackage } from '../scripts/staging.mjs'

const packageInfo = { name: '@ztd-me/eslint', version: '0.2.0' }
const environment = { NPM_TOKEN: 'test-only-placeholder' }

test('public versions skip without credentials or a stage-list request', async () => {
  const receipt = await stagePackage('verified.tgz', packageInfo, {
    environment: {},
    run(command, args) {
      assert.equal(command, 'pnpm')
      assert.deepEqual(args, [
        'view',
        packageInfo.name,
        'versions',
        '--json',
      ])
      return '["0.2.0"]'
    },
  })
  assert.deepEqual(receipt, { ...packageInfo, status: 'already-public' })
})

test('pending versions preserve their stage ID and never upload a duplicate', async () => {
  const calls = []
  const receipt = await stagePackage('verified.tgz', packageInfo, {
    environment,
    run(command, args) {
      calls.push({ command, args })
      if (args[0] === 'view') {
        return '["0.1.0"]'
      }
      return JSON.stringify([
        {
          packageName: packageInfo.name,
          version: packageInfo.version,
          id: 'existing-stage',
        },
      ])
    },
  })
  assert.equal(calls.length, 2)
  assert.deepEqual(calls[1], {
    command: 'pnpm',
    args: [
      'stage',
      'list',
      packageInfo.name,
      '--json',
    ],
  })
  assert.deepEqual(receipt, { ...packageInfo, status: 'already-staged', stageId: 'existing-stage' })
})

test('new versions stage the verified archive on main with scoped in-memory auth', async () => {
  const calls = []
  const receipt = await stagePackage('verified.tgz', packageInfo, {
    environment,
    run(command, args, options) {
      calls.push({ command, args, options })
      if (args[0] === 'view') {
        return '["0.1.0"]'
      }
      return args[1] === 'list' ? '[]' : JSON.stringify({ stageId: 'new-stage' })
    },
  })
  assert.equal(calls.length, 3)
  assert.deepEqual(calls[2].args, [
    'stage',
    'publish',
    'verified.tgz',
    '--access',
    'public',
    '--json',
    '--publish-branch',
    'main',
  ])
  assert.equal(calls[2].command, 'pnpm')
  assert.deepEqual(JSON.parse(calls[2].options.env.pnpm_config__auth), {
    'https://registry.npmjs.org': { '@ztd-me': { authToken: environment.NPM_TOKEN } },
  })
  assert.deepEqual(receipt, { stageId: 'new-stage' })
})

test('registry failures, invalid metadata and missing credentials fail before staging', async () => {
  for (const response of [
    '{}',
    '[1]',
    'invalid JSON',
    new Error('registry failure'),
    '["0.1.0"]',
  ]) {
    await assert.rejects(stagePackage('verified.tgz', packageInfo, {
      environment: {},
      run(command, args) {
        assert.equal(command, 'pnpm')
        assert.equal(args[0], 'view')
        if (response instanceof Error) {
          throw response
        }
        return response
      },
    }))
  }
})

test('stage-list auth errors and malformed responses fail without upload', async () => {
  for (const response of [
    '{}',
    '[{"version":"0.2.0"}]',
    'invalid JSON',
    new Error('Unauthorized stage-list request'),
  ]) {
    const calls = []
    await assert.rejects(stagePackage('verified.tgz', packageInfo, {
      environment,
      run(command, args) {
        calls.push(args)
        assert.equal(command, 'pnpm')
        if (args[0] === 'view') {
          return '["0.1.0"]'
        }
        if (response instanceof Error) {
          throw response
        }
        return response
      },
    }))
    assert.equal(calls.length, 2)
    assert.equal(calls[1][1], 'list')
  }
})
