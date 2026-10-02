import assert from 'node:assert/strict'
import test from 'node:test'
import { stagePackage } from '../../../scripts/npm-staging.mjs'

const packageInfo = { name: '@ztd-me/frontend-checks', version: '0.1.0' }
const environment = { NPM_TOKEN: 'test-only-placeholder' }

function registryError(code, status = 1) {
  return Object.assign(new Error('Registry request failed'), {
    status,
    stdout: JSON.stringify({ error: { code } }),
  })
}

test('a first-package structured 404 reaches authenticated lookup and stages once', async () => {
  const calls = []
  const receipt = await stagePackage('verified.tgz', packageInfo, {
    environment,
    run(command, args, options) {
      calls.push(args)
      assert.equal(command, 'pnpm')
      if (args[0] === 'view') {
        throw registryError('ERR_PNPM_FETCH_404')
      }
      const auth = JSON.parse(options.env.pnpm_config__auth)
      assert.equal(auth['https://registry.npmjs.org']['@ztd-me'].authToken, environment.NPM_TOKEN)
      if (args[1] === 'list') {
        return '[]'
      }
      assert.deepEqual(args, [
        'stage',
        'publish',
        'verified.tgz',
        '--access',
        'public',
        '--json',
        '--publish-branch',
        'main',
      ])
      return '{"stageId":"new-stage"}'
    },
  })
  assert.equal(calls.length, 3)
  assert.equal(calls[1][1], 'list')
  assert.deepEqual(receipt, { stageId: 'new-stage' })
})

test('a missing public package never bypasses token or stage-list authorization', async () => {
  let calls = 0
  await assert.rejects(stagePackage('verified.tgz', packageInfo, {
    environment: {},
    run() {
      calls++
      throw registryError('ERR_PNPM_FETCH_404')
    },
  }), /NPM_TOKEN/u)
  assert.equal(calls, 1)
  await assert.rejects(stagePackage('verified.tgz', packageInfo, {
    environment,
    run(_command, args) {
      if (args[0] === 'view') {
        throw registryError('ERR_PNPM_FETCH_404')
      }
      assert.equal(args[1], 'list')
      throw new Error('Stage-list permission denied')
    },
  }), /permission denied/u)
})

test('auth, network, malformed and ambiguous registry failures never stage', async () => {
  for (const error of [
    registryError('ERR_PNPM_FETCH_401'),
    registryError('ERR_PNPM_FETCH_403'),
    registryError('ERR_PNPM_FETCH_500'),
    registryError('ERR_PNPM_FETCH_404', 2),
    Object.assign(new Error('404 in text is insufficient'), { status: 1, stdout: 'Not Found - 404' }),
    new Error('network unavailable'),
  ]) {
    let calls = 0
    await assert.rejects(stagePackage('verified.tgz', packageInfo, {
      environment,
      run(_command, args) {
        calls++
        assert.equal(args[0], 'view')
        throw error
      },
    }))
    assert.equal(calls, 1)
  }
})

test('a new-package pending stage is preserved after the public placeholder exists', async () => {
  const receipt = await stagePackage('verified.tgz', packageInfo, {
    environment,
    run(_command, args) {
      if (args[0] === 'view') {
        return '["0.0.0-stage"]'
      }
      assert.equal(args[1], 'list')
      return JSON.stringify([
        { ...packageInfo, packageName: packageInfo.name, id: 'pending-stage' },
      ])
    },
  })
  assert.equal(receipt.stageId, 'pending-stage')
  assert.equal(receipt.status, 'already-staged')
})
