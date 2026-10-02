import { execFileSync } from 'node:child_process'
import process from 'node:process'

async function isPublished(name, version, run, environment) {
  const versions = JSON.parse(run('pnpm', [
    'view',
    name,
    'versions',
    '--json',
  ], { env: environment, encoding: 'utf8' }))
  if (!Array.isArray(versions) || versions.some(value => typeof value !== 'string')) {
    throw new Error('Unexpected public registry metadata; refusing to stage')
  }
  return versions.includes(version)
}

function pendingStage(items, name, version) {
  if (!Array.isArray(items) || items.some(item => (
    typeof item?.id !== 'string'
    || typeof item.packageName !== 'string'
    || typeof item.version !== 'string'
  ))) {
    throw new Error('Unexpected stage-list response; refusing to stage')
  }
  return items.find(item => item.packageName === name && item.version === version)
}

export async function stagePackage(tarball, { name, version }, {
  environment = process.env,
  run = execFileSync,
} = {}) {
  if (await isPublished(name, version, run, environment)) {
    return { name, version, status: 'already-public' }
  }
  const token = environment.NPM_TOKEN
  if (token === undefined || token.length === 0) {
    throw new Error('The repository NPM_TOKEN secret is unavailable; configure the authorized stage-only token.')
  }
  const env = {
    ...environment,
    pnpm_config__auth: JSON.stringify({
      'https://registry.npmjs.org': { '@ztd-me': { authToken: token } },
    }),
  }
  const options = { env, encoding: 'utf8' }
  const staged = pendingStage(JSON.parse(run('pnpm', [
    'stage',
    'list',
    name,
    '--json',
  ], options)), name, version)
  if (staged !== undefined) {
    return { name, version, status: 'already-staged', stageId: staged.id }
  }
  return JSON.parse(run('pnpm', [
    'stage',
    'publish',
    tarball,
    '--access',
    'public',
    '--json',
    '--publish-branch',
    'main',
  ], options))
}
