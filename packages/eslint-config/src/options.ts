import type { OptionsConfig, StylisticConfig, TypedFlatConfigItem } from '@antfu/eslint-config'
import type { Linter } from 'eslint'

export interface TypeScriptOptions {
  /** Default: tsconfig.json. Its resolved compiler options must enable strict and noUncheckedIndexedAccess. */
  tsconfigPath?: string
  /** Absolute project root; defaults to process.cwd(). */
  tsconfigRootDir?: string
}

export interface ReactOptions {
  /** Scope both antfu and strict React rules; defaults to JS/TS source extensions. */
  files?: string[]
  /** Additional official React Compiler checks. */
  compiler?: boolean
  /** Opt in to the experimental fetch cleanup check. */
  experimental?: boolean
  /** Explicit App Router integration. Defaults to framework-neutral React. */
  framework?: 'next' | 'vinext'
  /** Literal app directory relative to the config root. Default: app and src/app. Requires framework. */
  appDir?: string
}

export interface VueOptions {
  /** Vue 3 SFCs. Default: **\/*.vue. */
  files?: string[]
}

type RemovedOptions
  = | 'typescript'
    | 'react'
    | 'vue'
    | 'jsx'
    | 'test'
    | 'autoRenamePlugins'
    | 'isInEditor'
    | 'antislop'
    | 'stylistic'

type BaseOptions = Omit<OptionsConfig, RemovedOptions>

export interface ConfigOptions extends BaseOptions {
  /** Stable formatting remains active because line limits and array policy require it. */
  stylistic?: true | Omit<StylisticConfig, 'experimental'>
  /** Typed checking is enabled by default. Set false only for a JavaScript-only scope. */
  typescript?: boolean | TypeScriptOptions
  /** Explicitly enable React. Default: false. */
  react?: boolean | ReactOptions
  /** Explicitly enable Vue 3, including accessibility. Default: false. */
  vue?: boolean | VueOptions
  /** Enable the researched Vitest checks only for Vitest projects. Default: false. */
  test?: boolean
  /** Final, explicit project rule overrides. */
  rules?: Linter.RulesRecord
}

export type LocalConfig = TypedFlatConfigItem | Linter.Config
