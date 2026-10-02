import type { Linter, Rule } from 'eslint'
import type { LocalConfig, ReactOptions } from './options.js'
import refresh from 'eslint-plugin-react-refresh'
import { isRecord } from './array-simple.js'

function isRuleModule(value: unknown): value is Rule.RuleModule {
  return isRecord(value) && typeof value.create === 'function' && isRecord(value.meta)
}

function isRuleContext(value: unknown): value is Rule.RuleContext {
  return isRecord(value) && typeof value.report === 'function' && typeof value.filename === 'string'
    && isRecord(value.sourceCode) && Array.isArray(value.options)
}

function resolveFramework(value: unknown): 'next' | 'vinext' | undefined {
  if (value === undefined || value === 'next' || value === 'vinext') {
    return value
  }
  throw new Error('@ztd-me/eslint: unsupported react.framework')
}

function loadRefreshRule(): Rule.RuleModule {
  const candidate: unknown = refresh.rules['only-export-components']
  if (!isRuleModule(candidate)) {
    throw new Error('@ztd-me/eslint: incompatible react-refresh rule implementation')
  }
  return candidate
}

const refreshRule = loadRefreshRule()

const serverExports = [
  'metadata',
  'generateMetadata',
  'viewport',
  'generateViewport',
  'generateStaticParams',
  'dynamic',
  'dynamicParams',
  'revalidate',
  'fetchCache',
  'runtime',
  'preferredRegion',
  'maxDuration',
]

export const appRouterExports: Rule.RuleModule = {
  meta: {
    ...refreshRule.meta,
    schema: [
      {
        type: 'object',
        properties: { framework: { enum: ['next', 'vinext'] } },
        required: ['framework'],
        additionalProperties: false,
      },
    ],
  },
  create(context) {
    const isClient = context.sourceCode.ast.body.some(statement => (
      statement.type === 'ExpressionStatement' && 'directive' in statement && statement.directive === 'use client'
    ))
    const option: unknown = context.options[0]
    const framework = resolveFramework(isRecord(option) ? option.framework : undefined)
    const allowExportNames = isClient
      ? []
      : [
          ...serverExports,
          ...(framework === 'next'
            ? ['instant', 'prefetch']
            : []),
        ]
    const adapted: unknown = Object.create(context, {
      options: { value: [
        { allowExportNames, allowConstantExport: false },
      ] },
      filename: { value: context.filename.replace(/\.[jt]s$/u, extension => `${extension}x`) },
    })
    if (!isRuleContext(adapted)) {
      throw new Error('@ztd-me/eslint: incompatible ESLint rule context')
    }
    return refreshRule.create(adapted)
  },
}

function appDirectories(options: ReactOptions): string[] {
  if (options.appDir === undefined) {
    return ['app', 'src/app']
  }
  if (typeof options.appDir !== 'string') {
    throw new TypeError('@ztd-me/eslint: react.appDir must be a literal relative directory')
  }
  const directory = options.appDir.replaceAll('\\', '/').replace(/\/$/u, '')
  const hasTraversal = directory.split('/').some(part => part === '' || part === '.' || part === '..')
  if (/[:*?{}[\]()!]/u.test(directory) || hasTraversal) {
    throw new Error('@ztd-me/eslint: react.appDir must be a literal relative directory without globs or traversal')
  }
  return [directory]
}

export function appRouterConfigs(options: ReactOptions): LocalConfig[] {
  const framework = resolveFramework(options.framework)
  if (framework === undefined) {
    if (options.appDir !== undefined) {
      throw new Error('@ztd-me/eslint: react.appDir requires an explicit react.framework')
    }
    return []
  }
  const directories = appDirectories(options)
  const files = directories.map(directory => `${directory}/**/{page,layout}.{js,jsx,ts,tsx}`)
  const rules: Linter.RulesRecord = {
    'react-refresh/only-export-components': 'off',
    'ztd/app-router-exports': [
      'error',
      { framework },
    ],
  }
  return [
    {
      name: 'ztd/react-app-router',
      files: options.files === undefined ? files : options.files.flatMap(scope => files.map(file => [scope, file])),
      ignores: directories.map(directory => `${directory}/**/_*/**`),
      rules,
    },
  ]
}
