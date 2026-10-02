import assert from 'node:assert/strict'
import path from 'node:path'
import { test } from 'node:test'
import ts from 'typescript'

function diagnostics(unpatched) {
  const filename = path.resolve('src/radix-probe.mts')
  const options = {
    strict: true,
    exactOptionalPropertyTypes: true,
    skipLibCheck: false,
    noEmit: true,
    target: ts.ScriptTarget.ES2023,
    module: ts.ModuleKind.NodeNext,
    moduleResolution: ts.ModuleResolutionKind.NodeNext,
    types: [],
  }
  const host = ts.createCompilerHost(options)
  const original = host.getSourceFile.bind(host)
  host.getSourceFile = (file, version, ...rest) => {
    if (file === filename) {
      const probe = [
        'import type { SelectContentProps } from "@radix-ui/react-select";',
        'export type Props = SelectContentProps',
      ].join('\n')
      return ts.createSourceFile(file, probe, version, true)
    }
    const source = original(file, version, ...rest)
    if (unpatched && source && file.includes('/react-select/dist/index.d.')) {
      const text = source.text.replace(
        'extends Omit<PopperContentProps, keyof SelectPopperPrivateProps>, SelectPopperPrivateProps',
        'extends PopperContentProps, SelectPopperPrivateProps',
      )
      return ts.createSourceFile(file, text, version, true)
    }
    return source
  }
  return ts.getPreEmitDiagnostics(ts.createProgram([filename], options, host))
}

test('pinned Radix declaration-only patch fixes the reproduced TS2320 with strict library checking', () => {
  assert.ok(diagnostics(true).some(value => value.code === 2320))
  assert.deepEqual(diagnostics(false).map(value => ts.flattenDiagnosticMessageText(value.messageText, ' ')), [])
})
