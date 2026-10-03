import type { OptionsReact } from '@antfu/eslint-config'
import type { ConfigOptions, LocalConfig } from './options.js'
import antfu, { GLOB_SRC, GLOB_TESTS, GLOB_TS, GLOB_TSX } from '@antfu/eslint-config'
import hooks from 'eslint-plugin-react-hooks'
import sonar from 'eslint-plugin-sonarjs'
import { appRouterConfigs, appRouterExports } from './app-router.js'
import { arrayLayout } from './array-layout.js'
import { isRecord } from './array-simple.js'
import { reactRules } from './rules/react.js'
import { sharedRules } from './rules/shared.js'
import { testsRules } from './rules/tests.js'
import { tsSyntaxRules } from './rules/ts-syntax.js'
import { typedRules } from './rules/typed.js'
import { vueA11yRules } from './rules/vue-a11y.js'
import { vueRules } from './rules/vue.js'
import { resolveTypeScript } from './typescript.js'

export type { ConfigOptions, LocalConfig, ReactOptions, TypeScriptOptions, VueOptions } from './options.js'

const typeAwareIgnores = ['**/*.md/**', '**/*.astro/*.ts']

function shared(sfcFiles: string[]): LocalConfig {
  return {
    name: 'ztd/shared',
    files: [
      GLOB_SRC,
      ...sfcFiles,
    ],
    plugins: { sonarjs: sonar, ztd: { rules: {
      'array-layout': arrayLayout,
      'app-router-exports': appRouterExports,
    } } },
    linterOptions: { reportUnusedDisableDirectives: 'error' },
    rules: {
      ...sharedRules,
      'ztd/array-layout': 'error',
      'antfu/curly': 'off',
      'antfu/if-newline': 'off',
      'antfu/consistent-list-newline': [
        'error',
        { ArrayExpression: false },
      ],
      'style/array-bracket-newline': 'off',
      'style/array-element-newline': 'off',
      'no-unused-vars': 'off',
      'eslint-comments/no-unlimited-disable': 'error',
    },
  }
}

function typed(files: string[], parserOptions: ReturnType<typeof resolveTypeScript>['parserOptions']): LocalConfig[] {
  return [
    {
      name: 'ztd/typescript-syntax',
      files,
      rules: {
        ...tsSyntaxRules,
        'max-params': 'off',
        'no-shadow': 'off',
        'no-unused-expressions': 'off',
        'no-useless-constructor': 'off',
        'no-setter-return': 'off',
        'ts/no-unused-vars': 'off',
        'ts/explicit-function-return-type': 'off',
      },
    },
    {
      name: 'ztd/typescript-typed',
      files,
      ignores: typeAwareIgnores,
      // antfu applies its general parserOptions to both parsers; keep project settings in this typed scope.
      languageOptions: { parserOptions },
      rules: {
        ...typedRules,
        'require-await': 'off',
        'no-throw-literal': 'off',
        'prefer-promise-reject-errors': 'off',
        'no-constant-binary-expression': 'off',
      },
    },
  ]
}

function react(options: ConfigOptions['react']): LocalConfig[] {
  if (options === false || options === undefined) {
    return []
  }
  const settings = typeof options === 'object' ? options : {}
  const { 'react/no-leaked-conditional-rendering': _typedRule, ...rules } = reactRules
  return [
    {
      name: 'ztd/react',
      files: settings.files ?? [GLOB_SRC],
      plugins: { 'react-hooks': hooks },
      rules: {
        ...rules,
        'react/web-api-no-leaked-fetch': settings.experimental === true ? 'error' : 'off',
        ...Object.fromEntries([
          'config',
          'gating',
          'incompatible-library',
          'preserve-manual-memoization',
        ].map(rule => [
          `react-hooks/${rule}`,
          settings.compiler === true ? 'error' : 'off',
        ])),
      },
    },
    ...appRouterConfigs(settings),
  ]
}

function vue(options: ConfigOptions['vue']): LocalConfig[] {
  if (options === false || options === undefined) {
    return []
  }
  const files = typeof options === 'object' ? options.files ?? ['**/*.vue'] : ['**/*.vue']
  return [
    {
      name: 'ztd/vue',
      files,
      rules: {
        ...vueRules,
        ...vueA11yRules,
        'style/max-len': 'off',
        'vue/array-bracket-newline': 'off',
        'vue/array-element-newline': 'off',
      },
    },
  ]
}

function vitest(enabled: boolean, tsFiles: string[]): LocalConfig[] {
  if (!enabled) {
    return []
  }
  const { 'test/unbound-method': _unbound, ...syntaxTests } = testsRules
  const configs: LocalConfig[] = [
    {
      name: 'ztd/vitest',
      files: GLOB_TESTS,
      rules: { ...syntaxTests, 'test/no-only-tests': 'error' },
    },
  ]
  if (tsFiles.length > 0) {
    configs.push({
      name: 'ztd/vitest-typed',
      files: tsFiles.flatMap(file => GLOB_TESTS.map(testFile => [file, testFile])),
      ignores: typeAwareIgnores,
      rules: { 'ts/unbound-method': 'off', 'test/unbound-method': [
        'error',
        { ignoreStatic: false },
      ] },
    })
  }
  return configs
}

function vueFiles(options: ConfigOptions['vue']): string[] {
  if (options === false || options === undefined) {
    return []
  }
  return typeof options === 'object' ? options.files ?? ['**/*.vue'] : ['**/*.vue']
}

function antfuReact(options: ConfigOptions['react'], tsFiles: string[]) {
  if (options === false || options === undefined) {
    return false
  }
  const files = typeof options === 'object' ? options.files ?? [GLOB_SRC] : [GLOB_SRC]
  const overrides: OptionsReact['overrides'] = {
    'react-refresh/only-export-components': [
      'error',
      { allowConstantExport: false, allowExportNames: [] },
    ],
  }
  return {
    files,
    filesTypeAware: files.flatMap(file => tsFiles.map(tsFile => [file, tsFile])),
    overrides,
  }
}

function antfuVue(files: string[]) {
  return files.length === 0 ? false : { a11y: true, vueVersion: 3 as const, files }
}

function resolveDefaults(options: ConfigOptions) {
  return {
    ...options,
    typescript: options.typescript ?? true,
    react: options.react ?? false,
    vue: options.vue ?? false,
    test: options.test ?? false,
    rules: options.rules ?? {},
  }
}

function validateFormatting(settings: unknown): void {
  if (settings === false || (isRecord(settings) && settings.experimental === true)) {
    throw new Error('@ztd-me/eslint: stable stylistic formatting is required; '
      + 'disable competing experimental list rules')
  }
}

/** Strict flat configs. Project overrides are deliberately last and reviewable. */
export async function createConfig(
  options: ConfigOptions = {},
  ...localConfigs: LocalConfig[]
): Promise<Awaited<ReturnType<typeof antfu>>> {
  validateFormatting(options.stylistic)
  const {
    typescript,
    react: enableReact,
    vue: enableVue,
    test,
    rules,
    ...base
  } = resolveDefaults(options)
  const tsOptions = typescript === false ? false : resolveTypeScript(typescript)
  const sfcFiles = vueFiles(enableVue)
  const tsFiles = [
    GLOB_TS,
    GLOB_TSX,
    ...sfcFiles,
  ]
  const strict: LocalConfig[] = [
    shared(sfcFiles),
    ...react(enableReact),
    ...vue(enableVue),
    ...(tsOptions === false ? [] : typed(tsFiles, tsOptions.parserOptions)),
    ...vitest(test, typescript === false ? [] : tsFiles),
  ]
  return await antfu({
    ...base,
    typescript: tsOptions === false
      ? false
      : {
          tsconfigPath: tsOptions.tsconfigPath,
          filesTypeAware: tsFiles,
          ignoresTypeAware: typeAwareIgnores,
        },
    react: antfuReact(enableReact, tsFiles),
    vue: antfuVue(sfcFiles),
    test,
    jsx: { a11y: false },
    autoRenamePlugins: true,
    isInEditor: false,
    antislop: false,
  }, ...strict, { name: 'ztd/project-rules', rules }, ...localConfigs)
}

export default createConfig
