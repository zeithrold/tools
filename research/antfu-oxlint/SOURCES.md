# 官方来源、锁定版本与可审计证据

核查时间：2026-10-01。优先依据官方仓库和文档；网页可变，下面的源码/文档 commit 固定实验语境。源码 clones 仅供读取，实验依赖来自锁文件中的 npm tarball，未将整个上游仓库复制进 tools。源码 tag 不是自动证明所有相似规则等价的凭证。

## antfu

版本 **9.5.1**，tag `v9.5.1`，commit **df4d896ed9b493ca0562fdf2c8c0fcd92fd16f6e**。

- [factory.ts](https://github.com/antfu/eslint-config/blob/df4d896ed9b493ca0562fdf2c8c0fcd92fd16f6e/src/factory.ts)：依赖/环境推断、gitignore、子模块、局部 overrides、composer、renamePlugins、editor disableRulesFix。
- [TypeScript 配置](https://github.com/antfu/eslint-config/blob/df4d896ed9b493ca0562fdf2c8c0fcd92fd16f6e/src/configs/typescript.ts)：tsconfigPath、projectService、typed/untyped 文件块和规则 options。
- [React 配置](https://github.com/antfu/eslint-config/blob/df4d896ed9b493ca0562fdf2c8c0fcd92fd16f6e/src/configs/react.ts)：实际 @eslint-react、React refresh 等插件来源。
- [测试配置](https://github.com/antfu/eslint-config/blob/df4d896ed9b493ca0562fdf2c8c0fcd92fd16f6e/src/configs/test.ts)：合成 test plugin，不能等同一个 npm Vitest plugin。
- [Markdown 配置](https://github.com/antfu/eslint-config/blob/df4d896ed9b493ca0562fdf2c8c0fcd92fd16f6e/src/configs/markdown.ts)：language/processor、虚拟代码文件和禁用分组。
- [Stylistic 配置](https://github.com/antfu/eslint-config/blob/df4d896ed9b493ca0562fdf2c8c0fcd92fd16f6e/src/configs/stylistic.ts)、[Import 配置](https://github.com/antfu/eslint-config/blob/df4d896ed9b493ca0562fdf2c8c0fcd92fd16f6e/src/configs/imports.ts)：实际插件/选项。

实际 npm 导出的 composer 与 ESLint calculateConfigForFile 结果见 [inventory.json](results/inventory.json)。报告的 profile 表由这些执行结果计算，不从 README 规则列表推算。

## Oxlint

版本 **1.86.0**，tag `oxlint_v1.86.0`，commit **2ae2939bb2fd98796393658b21556b2a2467e047**。

- [SourceCode API](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/apps/oxlint/src-js/plugins/source_code.ts)：AST、tokens、scope、parserServices 空对象。
- [Context API](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/apps/oxlint/src-js/plugins/context.ts)：上下文复用、语言选项、parser.parse 未实现；context 不应跨文件异步保存并期待旧值。
- [plugin loading](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/apps/oxlint/src-js/plugins/load.ts)、[options](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/apps/oxlint/src-js/plugins/options.ts)：plugin object、规则加载和 schema 验证。
- [原生 plugin 名称](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/crates/oxc_linter/src/config/plugins.rs)、[规则 ID 解析](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/crates/oxc_linter/src/config/rules.rs)：保留 namespace、别名及 scoped ID。
- [partial loader](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/crates/oxc_linter/src/loader/partial_loader/mod.rs)：Vue/Svelte/Astro JS 区域提取，非通用 ESLint parser/processor。
- [no-console](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/crates/oxc_linter/src/rules/eslint/no_console.rs)：动态属性跳过、suggestion 与 dangerous suggestion。
- [consistent-type-specifier-style](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/crates/oxc_linter/src/rules/import/consistent_type_specifier_style.rs)：原生 option 枚举，与 import-lite 原选项不同。
- [JS diagnostics/fixes 转换](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/crates/oxc_linter/src/lib.rs#L932)：常规 JS fix 的 FixKind::Fix 与 suggestions 分类。
- [conformance README](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/apps/oxlint/conformance/README.md)、[锁定 repo/version 清单](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/apps/oxlint/conformance/repos.json)、[Stylistic snapshot](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/apps/oxlint/conformance/snapshots/stylistic.md)：已知兼容插件的具体版本及失败，不能套用到任意新版本。

本机 CLI 的原生目录与 type_aware/fix 标记保存在 [native-rules.json](results/native-rules.json)，与官网动态页面规则计数分别处理。

## 官方迁移工具

`@oxlint/migrate` **1.86.0**，tag `v1.86.0`，commit **56e2fc76cf0c673631c30f6b4dcca0cba65f165e**，官方 [oxc-project/oxlint-migrate](https://github.com/oxc-project/oxlint-migrate/tree/56e2fc76cf0c673631c30f6b4dcca0cba65f165e)。

- [js_plugin_fixes.ts](https://github.com/oxc-project/oxlint-migrate/blob/56e2fc76cf0c673631c30f6b4dcca0cba65f165e/src/js_plugin_fixes.ts)：live antfu composer 的 renamePlugins 特判。
- [plugins_rules.ts](https://github.com/oxc-project/oxlint-migrate/blob/56e2fc76cf0c673631c30f6b4dcca0cba65f165e/src/plugins_rules.ts)：规则筛选、import sorting 排除、native/JS 映射。
- [jsPlugins.ts](https://github.com/oxc-project/oxlint-migrate/blob/56e2fc76cf0c673631c30f6b4dcca0cba65f165e/src/jsPlugins.ts)：npm specifier/name 启发式，不能替代有效 plugin object。
- [files.ts](https://github.com/oxc-project/oxlint-migrate/blob/56e2fc76cf0c673631c30f6b4dcca0cba65f165e/src/files.ts)、[ignorePatterns.ts](https://github.com/oxc-project/oxlint-migrate/blob/56e2fc76cf0c673631c30f6b4dcca0cba65f165e/src/ignorePatterns.ts)、[settings.ts](https://github.com/oxc-project/oxlint-migrate/blob/56e2fc76cf0c673631c30f6b4dcca0cba65f165e/src/settings.ts)、[index.ts](https://github.com/oxc-project/oxlint-migrate/blob/56e2fc76cf0c673631c30f6b4dcca0cba65f165e/src/index.ts)：AND、本地 ignores、根 settings、basePath 不完整迁移的依据。

本机 11 份原始 migrator 输出在 results/migrated-*.json；加载失败和修复后执行/排序反例在 experiments.json 的 migration 字段。没有仅根据源码猜测“工具肯定不能运行”。

## 官方文档及 ESLint/TS 插件依据

浏览并读取 [oxc-project/oxc-project.github.io](https://github.com/oxc-project/oxc-project.github.io/tree/e66e5351626f8d6e0619667ba2e15f1e5bbc92c0) 的文档 snapshot commit **e66e5351626f8d6e0619667ba2e15f1e5bbc92c0**，对应目录 `src/docs/guide/usage/linter/`：

- [JS plugins](https://oxc.rs/docs/guide/usage/linter/js-plugins.html)：alpha、ESLint API、尚不支持的 custom formats/parsers 与 JS type-awareness。
- [Type-aware linting](https://oxc.rs/docs/guide/usage/linter/type-aware.html)：tsgolint/typescript-go、安装及开关、TS7 编译器兼容边界。
- [Migration](https://oxc.rs/docs/guide/usage/linter/migrate-from-eslint.html)：官方 migrate 和分步迁移建议。
- [Config reference](https://oxc.rs/docs/guide/usage/linter/config-file-reference.html)：plugins/jsPlugins、severity、overrides/globals/settings。
- [Ignore comments](https://oxc.rs/docs/guide/usage/linter/ignore-comments.html)：disable 支持；本研究另验证 ESLint 指令语法和 rule options 差异。
- [Automatic fixes](https://oxc.rs/docs/guide/usage/linter/automatic-fixes.html)：safe/suggestion/dangerous 的用户选择。
- [ESLint configuration files](https://eslint.org/docs/latest/use/configure/configuration-files)：flat config files/ignores/basePath/合并依据；实际行为以本机 ESLint 10.11.0 集成实验为准。
- [typescript-eslint consistent-type-imports](https://typescript-eslint.io/rules/consistent-type-imports/) 与 [typed linting](https://typescript-eslint.io/getting-started/typed-linting/)：规则与类型服务区分。

狭窄 adapter 的审计依据是锁定 npm 安装文件 `@typescript-eslint/eslint-plugin/dist/rules/consistent-type-imports.js`（8.71.0）及 `@typescript-eslint/utils/dist/eslint-utils/getParserServices.js`：该版本 rule 以 allowWithoutFullTypeInformation:true 获取 services，只读取装饰器标志；utils 仍要求两份 node map 存在。prototype 用会抛错的 map 占位，仅保存这两个 false 标志。绝不声称为其他 typed 规则提供真实 Program。两例 byte-identical fix、版本/装饰器拒绝及缺少服务的反例均可复现。

## 证据文件的角色

| 文件 | 证据 | 不证明的范围 |
| --- | --- | --- |
| package-lock.json | 安装版本、来源、integrity | 所有 API/规则兼容 |
| inventory.json / rule-matrix.csv | 有效 rule ID、severity/options 与路径分组、候选分类 | 全项目所有文件/动态选项、行为等价百分比 |
| native-rules.json | 锁定 Oxlint CLI 的实际目录 | antfu 或 JS 插件覆盖率 |
| migrated-*.json | 官方工具实际输出 | 输出可直接执行或语义等价 |
| experiments.json | 全部原始诊断/配置异常/fix/typed/格式/匹配/混合/副作用证据 | 全插件 suite、全部 parser/选项组合 |
| benchmark.json | 等规则工作量微基准、所有时间样本与诊断核对 | 真实项目收益、类型感知性能、常驻 API/物理冷机 |
| test/bridge.test.mjs | 真实工具验证原型的关键匹配/对象/上下文拒绝/fix行为 | 可直接替换生产 lint 栈 |
