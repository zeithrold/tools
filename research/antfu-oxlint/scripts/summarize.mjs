import { writeFile } from 'node:fs/promises'
import path from 'node:path'
import { loadJson, results, root } from './common.mjs'

const inventory = await loadJson(path.join(results, 'inventory.json'))
const experiments = await loadJson(path.join(results, 'experiments.json'))
const base = inventory.profiles.base
const baseFallback = new Set(base.rules.filter(r => r.category === 'eslint-fallback').map(r => r.id))
const lines = [
  '# 规则清单（由 `npm run summarize` 生成）',
  '',
  '分母是每个 profile 在 21 个记录路径上的有效配置中，severity > 0 的唯一 rule ID。候选分类不代表语义等价或生产覆盖率。同一 ID 在 JS 上有候选实现、在非 JS 文件上仍可能需要 ESLint。完整 options / severity / 文件分组见 [inventory.json](results/inventory.json)；全部候选 ID 见 [rule-matrix.csv](results/rule-matrix.csv)。',
  '',
  '| Profile | 分母 | 原生候选 | JS 插件候选 | 文件/类型模型兜底 |',
  '| --- | ---: | ---: | ---: | ---: |',
]
for (const [name, data] of Object.entries(inventory.profiles))
  lines.push(`| ${name} | ${data.denominator} | ${data.counts['native-candidate']} | ${data.counts['js-plugin-candidate']} | ${data.counts['eslint-fallback']} |`)
lines.push('', '## 基础配置全部 91 个兜底 ID', '', '这些 ID 只在所采样的非 JS 文件配置出现。其 parser、language、processor 或 AST 无法由当前 Oxlint JS 插件 API 直接提供；并不是说原规则自身没有 JS 实现。')
const groups = Map.groupBy([...baseFallback], id => id.split('/')[0])
for (const [namespace, ids] of groups)
  lines.push('', `### ${namespace}（${ids.length}）`, '', ...ids.map(id => `- \`${id}\``))
lines.push('', '## 各 profile 在基础清单之外增加的兜底 ID', '')
for (const [name, data] of Object.entries(inventory.profiles)) {
  const extra = data.rules.filter(r => r.category === 'eslint-fallback' && !baseFallback.has(r.id))
  if (extra.length)
    lines.push(`### ${name}（${extra.length}）`, '', ...extra.map(r => `- \`${r.id}\``), '')
}
lines.push('## 定向 specimen 的实际结果', '', '计数是记录的诊断数；JS 运行异常单列，不能当作规则诊断。修复相同只表示该片段最终文本相同。inline options 的反例虽然计数相同，报错行不同，见 README。类型感知、文件格式、匹配、混合与副作用实验在 experiments.json 的独立字段。', '', '| Specimen | 原 ID | ESLint / Oxlint 诊断 | 运行异常 | severity 计数一致 | 修复文本一致 |', '| --- | --- | --- | --- | --- | --- |')
for (const s of experiments.specimens) {
  const counts = s.eslint ? `${s.eslint.length} / ${s.oxlint.output?.diagnostics.length ?? '配置失败'}` : 'ESLint 配置拒绝'
  lines.push(`| ${s.name} | ${s.id ?? 'no-console'} | ${counts} | ${s.runtimeError ?? '—'} | ${s.matchingSeverityCounts ?? '—'} | ${s.edits?.equal ?? '未执行'} |`)
}
lines.push('', '## 官方 migrator 的 skipped 清单（基础 profile）', '', '这是工具自己的分类，不代表原型无法补齐；例如 no-restricted-syntax 和合成的 test/no-only-tests 已通过原插件实验。perfectionist/sort-imports 被单独警告并省略，未出现在 skipped 中。', '')
for (const [category, ids] of Object.entries(base.skipped))
  lines.push(`### ${category}`, '', ...ids.map(id => `- \`${id}\``), '')
await writeFile(path.join(root, 'RULES.md'), `${lines.join('\n').trimEnd()}\n`)
