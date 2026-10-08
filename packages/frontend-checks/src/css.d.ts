export type CssOptions = {
  files: string[]
  cwd?: string
  /** Explicit JS/TS/JSX/TSX files whose static Tailwind class literals are checked. */
  classFiles?: string[]
  /** External CSS declaration sources. Imports are not resolved implicitly. */
  tokenFiles?: string[]
  /** Exact runtime-provided names, reviewed in the project design contract. */
  externalCustomProperties?: string[]
}
export type CssWarning = {
  file: string
  line: number
  column: number
  rule: string
  text: string
}
export type CssReport = {
  schemaVersion: 1
  status: 'passed' | 'failed'
  files: string[]
  warnings: CssWarning[]
}
export function checkCss(options: CssOptions): Promise<CssReport>
