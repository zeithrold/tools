import assert from 'node:assert/strict'
import { readdir, readFile, realpath, writeFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import path from 'node:path'
import process from 'node:process'
import { pathToFileURL } from 'node:url'

const [consumer, sourceRoot] = process.argv.slice(2)
if (!consumer) {
  throw new Error('Provide an installed consumer directory')
}
const rootRequire = createRequire(path.join(consumer, 'package.json'))
const frontendRoot = sourceRoot === undefined
  ? path.dirname(rootRequire.resolve('@ztd-me/frontend/styles.css'))
  : path.resolve(consumer, sourceRoot)
const modules = [
  'appearance.js',
  'ui/menu-choice.js',
  'locale-select.js',
  'provider.js',
  'context.js',
]
const origins = Object.fromEntries(modules.map(file => [
  file,
  createRequire(path.join(frontendRoot, file)),
]))
const resolve = async (origin, name) => realpath(origins[origin].resolve(name))
const dropdown = await resolve('appearance.js', '@radix-ui/react-dropdown-menu')
const choiceDropdown = await resolve('ui/menu-choice.js', '@radix-ui/react-dropdown-menu')
const select = await resolve('locale-select.js', '@radix-ui/react-select')
const react = await realpath(rootRequire.resolve('react'))
assert.equal(dropdown, choiceDropdown)
const menuRequire = createRequire(createRequire(dropdown).resolve('@radix-ui/react-menu'))
const selectRequire = createRequire(select)
const sharedRadix = await Promise.all([
  'react-context',
  'react-dismissable-layer',
  'react-focus-scope',
  'react-portal',
  'react-presence',
  'react-popper',
].map(async (name) => {
  const dependency = `@radix-ui/${name}`
  const menuPath = await realpath(menuRequire.resolve(dependency))
  const selectPath = await realpath(selectRequire.resolve(dependency))
  assert.equal(menuPath, selectPath)
  return { dependency, resolved: menuPath, shared: true }
}))
for (const origin of modules) {
  assert.equal(await resolve(origin, 'react'), react)
}
const contexts = await Promise.all([
  'provider.js',
  'appearance.js',
  'locale-select.js',
].map(origin =>
  realpath(origins[origin].resolve('./context.js')),
))
assert.equal(new Set(contexts).size, 1)
const providerContext = await import(pathToFileURL(contexts[0]).href)
const hookContext = await import(pathToFileURL(contexts[1]).href)
assert.equal(providerContext.PreferenceContext, hookContext.PreferenceContext)
const retainedImports = await Promise.all(['appearance.js', 'locale-select.js'].map(async file => ({
  file,
  externalImport: /import \* as \w+ from '@radix-ui\//u.test(await readFile(path.join(frontendRoot, file), 'utf8')),
})))
const installed = await readdir(path.join(consumer, 'node_modules/.pnpm'))
const radix = installed.filter(name => name.startsWith('@radix-ui+')).sort()
const evidence = {
  consumer,
  frontendRoot,
  dropdown,
  choiceDropdown,
  select,
  react,
  contexts,
  sharedDropdownIdentity: true,
  sharedReactIdentity: true,
  sharedPreferenceContextIdentity: true,
  sharedRadix,
  retainedImports,
  radix,
  limitation: 'This isolated consumer does not establish dependency identity in uninspected production consumers.',
}
const receipt = sourceRoot === undefined ? 'dependency-identity.json' : 'dependency-identity-source.json'
await writeFile(path.resolve('.artifacts', receipt), JSON.stringify(evidence, null, 2))
console.info(JSON.stringify(evidence, null, 2))
