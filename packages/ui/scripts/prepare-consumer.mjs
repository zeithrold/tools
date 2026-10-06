import { execFileSync } from 'node:child_process'
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { copyConsumerFixture, retainConsumerLicenses } from './consumer-licenses.mjs'

export async function prepareConsumer(packageSpecifier) {
  await mkdir('.artifacts', { recursive: true })
  const consumer = await mkdtemp(path.resolve('.artifacts/consumer-'))
  await copyConsumerFixture(consumer)
  const packageInfo = JSON.parse(await readFile('package.json', 'utf8'))
  const devDependencies = Object.fromEntries(Object.entries(packageInfo.devDependencies).filter(([name]) => [
    '@types/react',
    '@types/react-dom',
    '@types/node',
    'typescript',
    'vite',
  ].includes(name)))
  const dependencies = {
    '@ztd-me/ui': packageSpecifier,
    'lucide-react': packageInfo.dependencies['lucide-react'],
    'react': packageInfo.devDependencies.react,
    'react-dom': packageInfo.devDependencies['react-dom'],
  }
  await writeFile(path.join(consumer, 'package.json'), JSON.stringify({
    private: true,
    license: 'MIT',
    type: 'module',
    packageManager: 'pnpm@11.22.0',
    dependencies,
    devDependencies,
  }))
  const workspace = await readFile('pnpm-workspace.yaml', 'utf8')
  const unpatchedPolicy = workspace.split('patchedDependencies:')[0]
  await writeFile(path.join(consumer, 'pnpm-workspace.yaml'), unpatchedPolicy)
  const config = JSON.parse(await readFile('tsconfig.build.json', 'utf8'))
  config.compilerOptions.noEmit = true
  config.compilerOptions.types.push('vite/client')
  delete config.compilerOptions.rootDir
  delete config.compilerOptions.outDir
  config.include = ['*.tsx']
  await writeFile(path.join(consumer, 'tsconfig.json'), JSON.stringify(config))
  const run = args => execFileSync('pnpm', args, { cwd: consumer, stdio: 'inherit' })
  run(['install'])
  run(['install', '--frozen-lockfile'])
  execFileSync('node', ['smoke.mjs'], { cwd: consumer, stdio: 'inherit' })
  run(['exec', 'tsc'])
  run([
    'exec',
    'vite',
    'build',
    '--outDir',
    'dist/client',
  ])
  run([
    'exec',
    'vite',
    'build',
    '--ssr',
    'server.tsx',
    '--outDir',
    'dist/server',
  ])
  await retainConsumerLicenses(consumer)
  return consumer
}
