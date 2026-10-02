export interface CssOptions {
  files: string[]
  cwd?: string
  /** External CSS declaration sources. Imports are not resolved implicitly. */
  tokenFiles?: string[]
  /** Exact runtime-provided names, reviewed in the project design contract. */
  externalCustomProperties?: string[]
}
export interface CssWarning {
  file: string
  line: number
  column: number
  rule: string
  text: string
}
export interface CssReport {
  schemaVersion: 1
  status: 'passed' | 'failed'
  files: string[]
  warnings: CssWarning[]
}
export function checkCss(options: CssOptions): Promise<CssReport>
