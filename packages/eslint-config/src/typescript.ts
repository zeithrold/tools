import type { TypeScriptOptions } from './options.js'
import { dirname, resolve } from 'node:path'
import process from 'node:process'
import ts from 'typescript'

export function resolveTypeScript(options: boolean | TypeScriptOptions): {
  tsconfigPath: string
  parserOptions: { tsconfigRootDir: string, projectService: false, project: string }
} {
  const settings = typeof options === 'object' ? options : {}
  const root = settings.tsconfigRootDir ?? process.cwd()
  const tsconfigPath = resolve(root, settings.tsconfigPath ?? 'tsconfig.json')
  const read = ts.readConfigFile(tsconfigPath, path => ts.sys.readFile(path))
  if (read.error !== undefined) {
    throw new Error(`@ztd-me/eslint: cannot read ${tsconfigPath}. `
      + 'Supply typescript.tsconfigPath or use typescript:false for JavaScript.')
  }
  const parsed = ts.parseJsonConfigFileContent(read.config, ts.sys, dirname(tsconfigPath))
  if (parsed.options.strict !== true || parsed.options.noUncheckedIndexedAccess !== true
    || parsed.options.strictNullChecks === false) {
    throw new Error('@ztd-me/eslint: tsconfig must enable strict (including strictNullChecks) '
      + 'and noUncheckedIndexedAccess')
  }
  return { tsconfigPath, parserOptions: { tsconfigRootDir: root, projectService: false, project: tsconfigPath } }
}
