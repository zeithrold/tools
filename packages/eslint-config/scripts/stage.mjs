import { execFileSync } from 'node:child_process'
import { writeFileSync } from 'node:fs'
import process from 'node:process'

const tarball = process.argv[2]
if (tarball === undefined) {
  throw new Error('Usage: node scripts/stage.mjs <verified-tarball>')
}
const mode = process.env.STAGE_AUTH ?? 'token'
const environment = { ...process.env }
if (mode === 'token') {
  const token = process.env.NPM_TOKEN
  if (token === undefined || token.length === 0) {
    throw new Error('The repository NPM_TOKEN secret is unavailable; configure the authorized stage-only token.')
  }
  environment.pnpm_config__auth = JSON.stringify({
    'https://registry.npmjs.org': { '@ztd-me': { authToken: token } },
  })
}
else if (mode !== 'oidc') {
  throw new Error('STAGE_AUTH must be token or oidc')
}
const branch = process.env.GITHUB_REF_NAME
  ?? execFileSync('git', ['branch', '--show-current'], { encoding: 'utf8' }).trim()
const result = execFileSync('pnpm', [
  'stage',
  'publish',
  tarball,
  '--access',
  'public',
  '--json',
  '--publish-branch',
  branch,
], { env: environment, encoding: 'utf8' })
writeFileSync('stage-result.json', result)
console.log(result)
