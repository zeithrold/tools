# antfu → Oxlint：源码研究与保守桥接原型

研究日期：2026-10-01。结论：**可以桥接相当一部分 JS/TS 检查，而且已有规则无需重写；目前不能把完整 antfu flat config 转成一份 Oxlint 配置并宣称等价。最可行的是按实际文件解析配置、逐规则验证后迁出、剩余检查继续由 ESLint 执行。** 本目录给出可运行原型、原始配置盘点、反例、集成测试和等工作量性能数据。

基础 profile 在 21 个代表路径上共有 **426 个唯一启用 ID：151 个原生候选、184 个 JS 插件候选、91 个文件模型兜底 ID**。这是候选盘点，不是“迁移成功率”。36 个定向 specimen 覆盖 20 个唯一 ID，另有类型感知、框架/数据文件、配置匹配、混合运行和副作用实验。混合原型只主动迁出 3 条已验证规则，在示例的两个 JS 文件上运行；其他文件和规则保留 ESLint，逐文件/ID/severity 校验全部 **20 条诊断**。没有将 335 个候选自动当成已兼容。

## 仓库与隔离范围

实际仓库是 `zeithrold/tools`，起点 `9a4c16b`，进入时工作树干净。`packages/eslint-config/index.js` 是私有的 `@zeithrold/eslint-config` 草稿，包装 antfu，默认 TypeScript 开启，React/Svelte 关闭，配置 `no-console` 允许 `warn/error`，可启用 `no-explicit-any` 和 7 条严格类型规则；项目拥有自己的 ignores、框架选择和最终 overrides。仓库没有可直接测量的真实前端消费项目。本报告的基础 profile 直接研究 antfu，不能当作该包装器或任何消费项目的最终覆盖率。

本次只新增 `research/antfu-oxlint/`，未替换生产 lint、修改消费仓库或接触本地 Biome 探索。没有 npm 发布、付费服务或权限变更。遵循根 AGENTS.md、README、包装器文档与 [JS/TS 测试 Skill](../../skills/js-ts-testing/SKILL.md)。

## 版本与复现

2026-10-01 从 npm registry 核实 `@antfu/eslint-config` / `oxlint` / `@oxlint/migrate` 的 latest 分别为 **9.5.1 / 1.86.0 / 1.86.0**。锁定 ESLint **10.11.0**、`oxlint-tsgolint` **7.0.2003**、本地 TypeScript **6.0.3**、`@typescript-eslint` parser/plugin **8.71.0**、Stylistic **5.10.0**、Perfectionist **5.12.1**、RegExp **3.3.1**、Unused Imports **4.4.1**。其余框架依赖和完整 integrity 信息在 [package-lock.json](package-lock.json)。框架依赖只是实验所用版本，未声称全都是各包的 latest。

实测机器：Linux x64、Node **24.19.0**、npm **11.9.0**、AMD EPYC 9V74，容器可见 5 个逻辑 CPU。没有使用 `--legacy-peer-deps`。本地 TS 6 是 ESLint/parser 的 npm 依赖选择；tsgolint 内部运行的是另一套 `typescript-go` 编译器，不是调用这份 npm TypeScript。

```sh
cd research/antfu-oxlint
npm --cache /tmp/oxlint-npm-cache ci --no-audit --no-fund
npm run lint
npm run reproduce
```

`reproduce` 顺序运行 inventory、experiment、5 个 Node 集成测试、benchmark、summarize。性能测量期间请避免其他 CPU 密集任务；默认 200 文件、5 次重复，可用 `BENCH_FILES` 和 `BENCH_REPETITIONS` 改小作冒烟检查，但小规模数据不可替代这里记录的性能结论。`npm run experiment` 依赖均已安装，但不依赖先运行 inventory。`.generated/` 是可重建的配置、临时文件和 fixture，不进 Git；`results/` 保存完整数据。路径、耗时及诊断顺序会随工作目录/机器改变，不能要求结果文件逐字不变。

## 实际 flat config 能保存哪些语义

[antfu factory 源码](https://github.com/antfu/eslint-config/blob/df4d896ed9b493ca0562fdf2c8c0fcd92fd16f6e/src/factory.ts) 返回可 await 的 `FlatConfigComposer`，不是固定 JSON。它按依赖存在性推断 TS/Vue，读取编辑器环境及 gitignore，合成测试插件，重命名插件，调用模块选项/overrides，追加用户 flat blocks。`type: 'lib'`、typed TS、React/Vue/Svelte/Astro/Solid、formatters 会改变规则、解析器及文件组。`isInEditor` 不只改变 severity：它还移除部分规则的 fixer；本次 editor 与 base 的启用 ID 数相同，不能因此推断行为相同。

直接保存最终 rule ID/options 不足以保存插件函数、parser、processor、language、规则对象被修改的 fixer、配置匹配或环境条件。稳妥做法是**在目标项目自己的依赖与环境中执行原配置，并让 ESLint `calculateConfigForFile` 作为匹配/合并依据**，再为明确文件生成 Oxlint 配置；原插件对象通过模块引用复用。配置、依赖、gitignore、环境、文件内容或 manifest 变化都必须重新生成。构建时求值也不能永久保存未来的动态选项。

基础 profile 实际得到 41 个 flat blocks。相同规则有不同文件/选项版本，例如 JS 303、普通 TS 308、测试 TS 312、`.d.ts` 305、scripts TS 306、CLI JS 299 条启用规则；这些不是可无条件平铺的同一组。JSONC/YAML/TOML 的有效配置还可能含 JS 基础规则，能“在 JSON 文件配置里看见 rule ID”不表示其 AST 与 JS 相同。

盘点分母是 [common.mjs](scripts/common.mjs) 中 21 个路径，经 ESLint 解析后的 severity > 0 唯一 ID：JS/TS/TSX/test/d.ts/script/CLI/config/CJS、三种框架文件、package/tsconfig/JSON/JSONC/YAML/TOML/Markdown/CSS/HTML。**不是规则包的全部规则数、配置声明数、所有可能文件/选项的并集，也不包括真实项目的任意 overrides。** inventory 的 typed profile 只解析有效配置，不构建不存在的项目 TS Program；真正的 typed 执行另用实验脚本生成的 tsconfig。

| Profile | 唯一 ID 分母 | 原生候选 | JS 插件候选 | 文件/类型模型兜底 |
| --- | ---: | ---: | ---: | ---: |
| automatic / base / editor（各自） | 426 | 151 | 184 | 91 |
| library | 427 | 152 | 184 | 91 |
| typed | 445 | 170 | 184 | 91 |
| react（含 typed） | 512 | 171 | 249 | 92 |
| vue | 576 | 151 | 184 | 241 |
| svelte | 471 | 151 | 184 | 136 |
| astro | 434 | 151 | 184 | 99 |
| solid | 441 | 151 | 199 | 91 |
| formatters（CSS/HTML） | 427 | 151 | 184 | 92 |

一个 ID 在 JS 有候选路径，仍可在非 JS occurrences 中兜底。原生候选只按**实际插件来源和 Oxlint 目录**筛选，明确不把 antfu 的 `react`（`@eslint-react`）当成 `eslint-plugin-react`，不把 `import-lite` 直接等同原生 import，也不把合成 `test/no-only-tests` 当成 Vitest 实现。所有候选仍需 options、语义和修复审查。缺少原生但有真实 ESLint 实现的规则，包括核心 rule 和 `node/prefer-global/process` 这类含斜线规则名，可以尝试 JS 包装；不能只按最后一个 `/` 猜插件名字。

完整 91 个基础兜底 ID，以及 Vue 新增 150、Svelte 新增 45、Astro 新增 8、React 新增 1、formatters 新增 1，全部列于 [RULES.md](RULES.md)。完整各 profile 的候选 ID 在 [rule-matrix.csv](results/rule-matrix.csv)，每种 options/severity 及其文件组在 [inventory.json](results/inventory.json)。

## 四类落地判断

| 分类 | 已有证据 | 边界与动作 |
| --- | --- | --- |
| 原生可映射的已测子集 | `no-debugger`、`eqeqeq` smart、`prefer-const` 的 destructuring/读取选项、TS ban-ts-comment、consistent-type-definitions、consistent-type-imports；两条 typed 规则另测 | 限记录的输入/选项；诊断位置、文案、建议不承诺完全相同。151 个原生候选不等于 151 个通过项 |
| JS 插件可补的已测子集 | 原 core no-console/no-restricted-syntax；Stylistic semi/quotes/indent/comma-dangle；antfu curly/top-level-await；import-lite、Perfectionist、unused-imports、RegExp、e18e、合成测试插件、实际 @eslint-react | 复用实际插件对象；需要 Oxlint AST/SourceCode/上下文足够兼容。保留原命名空间，遇保留名称则改名且含原 disable 的文件兜底 |
| 需要 ESLint 兜底 | JSONC、YAML、TOML、Markdown language/processor、CSS/HTML formatter；Vue/Svelte/Astro 模板；真正依赖 parserServices/TS Program 的 JS 规则；尚未验证的原生/插件规则 | 原型全部保守保留；原生 tsgolint 对已实现 typed 规则可作单独审计路线，无法给任意 JS 插件补类型服务 |
| 当前纯 Oxlint 无法直接等价 | ESLint 自定义 parser/language/processor 的 AST 和虚拟文件处理、任意 inline rule options、某些 disable 语法差异；所有规则的通用安全修复保证 | 当前配置 + JS plugin API 缺少所需机制。外部预处理/ESLint 调度可以继续实现功能，但那已是额外运行层；不能宣称一份 Oxlint 配置等价替代完整 antfu |

“兜底”与“当前纯 Oxlint 无法等价”是不同层级：前者描述原型执行归属，后者说明为什么不能通过配置/规则改名解决。全部未审计候选在生产建议上也归入 ESLint 保留集合。

## 代表性行为差异与插件实测

原始输入、options/severity、ESLint messages、Oxlint JSON、fix 输出都在 [experiments.json](results/experiments.json)。specimen 的统计表见 [RULES.md](RULES.md)。`exit 1` 可能是正常 error 诊断，也可能配置加载/规则运行失败，报告明确分开；warning-only 的 Oxlint `exit 0` 也不表示没有警告。

| 实验 | 实测结果 | 判断 |
| --- | --- | --- |
| no-console：allow warn/error、局部 console 遮蔽、静态/动态属性 | ESLint 3，原生 Oxlint 2；漏掉 `console[x](...)`。原 ESLint 核心规则装进 JS 插件恢复 3/3 | 同名不是等价；[原生源码](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/crates/oxc_linter/src/rules/eslint/no_console.rs) 对不可静态求名的属性返回。包装器默认规则也受这个差异影响 |
| import-lite 的 consistent-type-specifier-style | `top-level` 原值被原生拒绝；转成 `prefer-top-level` 后诊断相同但修复空白文本不同；原 import-lite JS 插件修复逐字相同 | schema 及修复必须审计；原型没有偷偷把“同计数”叫等价 |
| TS consistent-type-imports 原 JS 规则 | 无类型项目仍会读取 getParserServices；直接运行报缺少 parserServices。原生简单例子通过 | “语法规则”也可能要求 parser services 的形状 |
| 狭窄的 TS syntax adapter | 原 mixed type/value imports、注释两例修复与 ESLint 相同；装饰器标志开启时直接拒绝 | 仅锁定 @typescript-eslint 8.71.0 的已审计读取；不伪造 TS Program |
| 原 Stylistic/antfu/Perfectionist/unused-imports/RegExp/e18e 等 | 记录的片段诊断 severity 计数一致，有 fixer 的片段最终修复文本一致 | 可以直接复用规则，未运行每个插件完整 suite |
| no-only-tests | 真实 antfu 合成 `test` 插件可以通过；仅改名到 Vitest 并查原生目录会丢规则 | 必须引用实际有效 plugin 对象 |
| React | 实际 @eslint-react no-array-index-key JS 插件报 1 条 warning，与 ESLint 相同 | 不能用原生 React 包的同名/近似规则替代整个 @eslint-react |
| SourceCode / context probe | 两侧 14 tokens、1 comment、2 code paths、scope 可用、settings 保留；Oxlint parserServices 是 `{}`；Program 诊断起始位置不同 | 现代 tokens/scope/CFG/selector API 已相当完整，但不能推断所有上下文/位置完全一致 |
| `eslint-disable-next-line style/semi` | 保留 `style` 名称时两边 1 条；改成 `style-js` 后 Oxlint 2 条 | 插件前缀重命名可使旧 disable 失效 |
| `// eslint-disable ts/ban-ts-comment` | ESLint 1，原生和原 JS Oxlint 都 0；块注释 `/* eslint-disable ... */` 两边 0 | 此差异来自行注释全文件 disable 语法，不是本例 TS alias 不支持 |
| inline no-console allow-log options | 两边各 1 条；ESLint 报 `console.warn`（第 3 行），Oxlint 报 `console.log`（第 2 行） | inline rule options 没有保存。只比总条数会得出错误结论 |
| `no-console` options 增加 invented:true | 两边都配置拒绝，独立记录错误 | severity/options 需要先 schema 验证，不能从名称迁移盲用 |

### Autofix 与安全性

对各片段分别运行 ESLint fix 和 Oxlint `--fix`，比较最终文本；另以 no-console 的 statement/sequence 两种代码实测原生和原 core JS 插件的 `--fix`、`--fix-suggestions`、`--fix-suggestions --fix-dangerously`，记录在 `experiments.json.fixModes`。ESLint `fix:true` 不会自动应用 suggestions，因此没有把它和 Oxlint suggestions 模式混为一谈。原生有安全/危险/suggestion 分级，[官方说明](https://oxc.rs/docs/guide/usage/linter/automatic-fixes.html) 提供 `--fix`、`--fix-suggestions`、`--fix-dangerously`。锁定版本将 JS `context.report` 的常规 fixer 以 `FixKind::Fix` 加入：[源码](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/crates/oxc_linter/src/lib.rs#L932)。JS rule meta 没有自动推导程序行为安全性的机制。12 次 fix mode 实测中，两边 `--fix` 均不改 no-console；statement 在 suggestions 模式都被删除；sequence 只有原生的 suggestions + dangerously 会将调用改成 `undefined`，原 ESLint 核心 JS 插件没有这项 suggestion、保持原文。因此即使诊断相同，不同 fixer/suggestion 集合仍可能改变自动修改结果。

`unused-imports-side-effect-risk` 中，一个“未使用”的 import 所引入模块设置全局值。两边 `--fix` 都删除整个 import，修复文本完全一致，实际运行导出值却从 **1 变为 0**。这是复用上游 fixer 继承的风险，不是本次发现的 Oxlint 独有缺陷。因此修复逐字一致不能证明安全。拆成 ESLint/Oxlint 两次 fix 还会改变规则交互、冲突选择和多轮运行顺序；原型主要证明诊断分流，**没有提供完整 antfu 多轮 fix 等价保证**。上生产前仍应分别审计诊断与修复，并让项目选择是否启用具体 fixer。

## 最新版 Oxlint 的真实边界

实测 `oxlint --rules --format json` 为 **871 条目录项，其中 60 个 `type_aware` 标记**，完整结果见 [native-rules.json](results/native-rules.json)。这是该二进制目录，不是 antfu 覆盖率，也不保证所有目录项在一个配置中都可执行。官方网页 snapshot 仍写 59/61 typed coverage，口径/更新可能不同；本报告不把它替换成二进制计数。

[官方 JS plugin 文档](https://oxc.rs/docs/guide/usage/linter/js-plugins.html) 定义兼容 ESLint v9+，但仍是 alpha。源码实现 SourceCode、tokens/comments、scope、code paths、selectors、options/schema、settings、report/fixes 等；支持 npm 和本地插件模块。`parserServices` 冻结为空对象，`context.parser.parse` 明确未实现；**不能注入 ESLint 自定义解析器/文件格式，不能让任意 JS 规则获得 TypeScript 类型服务**。上游对 Stylistic/RegExp/e18e 等有 conformance snapshots，其中 Stylistic 的 128/129 fully passing 针对 **5.7.1**，本次安装 **5.10.0**；那份统计不是本次版本全部规则的验证证书。[锁定 snapshot](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/apps/oxlint/conformance/snapshots/stylistic.md)

类型感知是独立的 `tsgolint` / `typescript-go` 通道。真实 tsconfig fixture 中 no-floating-promises（error，ignoreVoid:true）与 no-unsafe-assignment（warn）两侧均报 **2 条**；文案、span、suggestion payload 不完全相同。相同 Oxlint 配置缺少 `--type-aware`/`options.typeAware` 时，实测 **0 rules / 0 diagnostics / exit 0**，不能把它当作通过类型检查。原 JS TS plugin 则报 parserServices 运行异常。原型路由暂将 typed 规则留在 ESLint；typed native 实验单独执行。官方通道要求 TS 7 的编译器兼容语义，已移除的 tsconfig 特性需要另做迁移，不能保证与项目 npm TS 6 或旧版本完全相同。[官方 typed 文档](https://oxc.rs/docs/guide/usage/linter/type-aware.html)

框架原生支持主要是 `.vue/.svelte/.astro` 的 script/frontmatter 提取，不能据此声称模板完整受支持。46 条原生 Vue 目录项主要针对 JS/TS Vue API。本次 7 个文件中，Oxlint 原生 no-debugger 检查了 3 个框架脚本区域，JSONC/YAML/TOML/Markdown 没有被检查；ESLint 同时发现 Vue v-for key、Svelte each key、JSONC duplicate key、YAML/TOML parse error、Markdown heading 和 fenced JS debugger。直接给 JS 插件装原 Vue require-v-for-key，会报告缺少 vue-eslint-parser 服务。CSS/HTML 的 antfu format/prettier 各有诊断，Oxlint 这两个文件 `number_of_files: 0` 并退出 1。React/Solid 的普通 JSX 规则另走 JS/TS AST，不能与 SFC 模板混为一类。[partial loader 源码](https://github.com/oxc-project/oxc/blob/2ae2939bb2fd98796393658b21556b2a2467e047/crates/oxc_linter/src/loader/partial_loader/mod.rs)

## 已有迁移工具：可用基础，尚非等价桥接

实际安装并执行 `@oxlint/migrate@1.86.0`，没有先假定需要重造转换器。它已经识别 live antfu composer 的 `renamePlugins()`，且能导出原生和 JS 插件规则，所以本原型没有重写原 antfu 规则。需传递 live composer；先 await 成普通数组会失去它的 antfu 特殊处理路径。

基础配置打开 jsPlugins/nursery 后的实测：

1. 原始输出加载错误的 `@eslint/eslint-plugin-markdown`，配置失败。
2. 修正 markdown 与 eslint-comments 包名后，`import/consistent-type-specifier-style` 的 `top-level` 被拒绝，eslint-comments 名称未绑定。
3. 再为 markdown/eslint-comments 设置显式 plugin name，并翻译 import option 后，简单 TS 文件可运行，得到与 ESLint 一致的 quotes/semi 两条诊断。这只证明配置能加载及该例能检查，不能证明全量覆盖。
4. 另一个排序 fixture 中，原始 ESLint 的 `perfectionist/sort-imports` 诊断被省略；migrator 明确假设使用 Oxfmt import sorting，但 Oxfmt 未运行且默认不开排序。换 formatter 的 options、分组、自定义排序、disable 和 fixer 语义仍需独立验证，不能算已经桥接。

工具的 skipped ID、warnings、11 份原始输出均入库。它还会丢本地 overrides ignores、跳过 AND globs、忽略 basePath、只保存根 settings。四文件匹配反例的 ESLint 结果分别为 **warning 1 / error 1 / warning 1 / 0**；迁移后为 **0 / 0 / 0 / 0**，因为原来限定 nested basePath 的 off block 被扩散至全局。工具只提示 AND/local ignores，并未把 basePath 差异完整解释出来。故本原型用 ESLint 计算每个文件的有效配置，并保留其真实插件对象及 settings，补充的是语义保守的编排层，而不是通用“规则改名器”。

另查 `@antfu/oxlint-config` registry 返回 E404；这只排除查询当时的这个包名，不代表穷尽整个生态。

## 原型结构与限制

- [materialize.mjs](prototype/materialize.mjs)：输入原 config module、有限文件 manifest、**外部验证过的 routes allowlist**；通过 ESLint 求每个文件有效配置，生成逐文件 Oxlint 配置和真实 plugin module 引用。未知规则保留 ESLint；不同文件可有不同 settings 和同名但不同对象的 plugin。核心规则通过 builtinRules 包装。
- [plugin.mjs](prototype/plugin.mjs)：取得 antfu 实际插件对象（含合成 test）、core JS 包装及 SourceCode/context probe。
- [syntax-type-imports.mjs](prototype/syntax-type-imports.mjs)：仅适配 @typescript-eslint 8.71.0 consistent-type-imports 读取的两个 false 装饰器标志。没有 Program，TS node map `.get/.has` 会显式抛错；版本改变或装饰器标志开启时拒绝运行。这不是可供其他 typed 规则使用的“假类型服务”。
- [experiment.mjs](scripts/experiment.mjs)：完整生成/执行例子，含原型 API 调用；[bridge.test.mjs](test/bridge.test.mjs) 用真实 ESLint/Oxlint 验证匹配、severity-only option merge、scoped/含斜线规则名、逐文件 plugin/settings、类型服务拒绝和语法 adapter 修复。

路由器对非 JS、processor、非已审计 parser、不同 sourceType/ecmaVersion、noInlineConfig、不能无损 JSON 化的 settings、任何识别出的 ESLint/Oxlint directive **整文件保留 ESLint**。所有 typed routes 和不能无损 JSON 化的 rule options 暂保留 ESLint。JSON 配置不能序列化函数/RegExp/symbol/circular/getter/稀疏数组等设置。inline 注释兜底也保留 ESLint 的 unused-disable 统计，避免残余 lint 因规则迁出而错误报 unused。没有强行改写用户注释。

每文件生成配置与进程是验证用途，尚未分组/缓存。插件 wrapper 在加载时重算该文件配置以取真实对象，有初始化成本。**下面性能表测的是等价静态编译配置，未把这个逐文件原型包装器的初始化与转换成本计入**；不能把表里的加速直接当作原型产品吞吐。没有文件监听/缓存失效、生产 runner、完整 parserOptions/所有 AST 差异的验证、整套全量 fixer 顺序证明，也没有自动安全证明器。`accept(entry, resolved)` 需要审计 option 和 parser 环境；示例中的 style accept:true 只对记录的环境负责，不能推广到任意配置。

## 性能：保留工作量以后再比较

使用 200 个相同 TS 文件，每个 72 行；单线程 Oxlint、ESLint concurrency off，关闭 lint 结果缓存，包含新进程启动、依赖初始化、配置解析、lint 和 JSON 输出。首轮是在生成 corpus 后的第一次新进程；5 次重复交替执行先后顺序，以中位数汇总。重复期间其他实验停止运行。

**“首轮”不是 OS 页缓存清空后的物理冷机；“暖缓存”仍然每次启动新进程，表示文件缓存受热，不是复用已经初始化的 JS 引擎。** 没有特权驱逐缓存，未测持久 API 的进程内 warm lint。cold process/module 初始化都包含在计时中。原始样本及机器信息见 [benchmark.json](results/benchmark.json)。

| 相同规则工作量 | 诊断核对 | 首轮 ESLint / 目标 ms | 暖缓存新进程中位数 ESLint / 目标 ms | 比值 |
| --- | --- | ---: | ---: | ---: |
| 原生 6 条，ESLint 每次构建 antfu 后缩到相同规则 | 1200 / 1200 | 2585.6 / 59.8 | 2707.3 / 68.3 | 39.65× |
| **原生 6 条，精简 ESLint 加载所需 TS 依赖** | 1200 / 1200 | 1618.4 / 67.8 | 1571.9 / 64.4 | **24.40×** |
| **6 原生 + 3 JS（semi/quotes/适配 type imports），精简 ESLint** | 3200 / 3200 | 1724.7 / 1122.4 | 1682.1 / 1107.1 | **1.52×** |
| **完整 antfu 308 条 vs 残余 305 + Oxlint 3** | 3400 / (1400 + 2000) | 4286.5 / 4557.4 | 4327.8 / 4671.5 | **0.93×，混合慢约 8%** |

原生 6 条使用同一有效配置的 debugger/console/eqeqeq/prefer-const/ban-ts-comment/type-definitions；语料只含静态 console，已另测其动态属性差异，因此本行不证明 no-console 全域等价。9 条再加入两个原 Stylistic 规则和狭窄 TS adapter。6/9 条都核对每文件/ID/severity 的 hash；full-hybrid 核对总数和各 ID 数，不是把全 antfu 与少数 Oxlint 规则直接比较。精简 ESLint 行移除了无关 antfu 构建开销，防止把依赖加载差异当成 lint 算法加速。

完整工作量的示例迁出只 3 条，不代表所有可行迁出组合的最优性能；它证明**为了保留剩余功能而增加一轮解析和 JS 初始化，完全可能变慢**。语料重复、单机和小规模都会影响数值，不能外推真实项目端到端收益；未测 typed 性能、formatter 替代、所有 candidates、真实消费工程或 API 常驻模式。

## 推荐落地路线

1. 继续让现有 `@zeithrold/eslint-config` 和项目本地 overrides 决定真实规则；在一个实际消费仓库中拿明确 manifest/tsconfig 求有效配置，记录依赖、gitignore 和 editor 模式。生产栈保持可回退。
2. 从无 inline directives 的普通 JS/TS 文件和已验证原生小集合开始，逐规则加入包含反例、options、diagnostic span、suggestions、fix 的差分测试。`no-console` 默认保留原实现或使用原 core JS 包装；不要仅凭 eslint-plugin-oxlint 的同名名单把检查关掉。
3. 复用现有 Stylistic/antfu/import-lite/排序等 JS 插件，不重写规则。先修官方 migrate 的 specifier/name、AND/local ignores/basePath/settings/合成插件问题，或直接用本原型的 ESLint 有效配置路线；迁出清单仍显式审计。
4. 原型进阶时合并完全相同的 file config、共享 plugin 导入、缓存真实 plugin 对象，正确更新 manifest/配置/内容失效；继续保留包含 directives 与非 JS 文件的完整 ESLint pass。优化后用真实等工作量测试决定是否收益为正。
5. typed 规则独立评估 tsgolint 的 TS7 项目兼容和 options，而非向 JS 插件伪造 Program。对装饰器、类型服务、模板或 processor 请求继续 ESLint；必要时向 Oxlint/migrate 上游提交最小复现。本次没有发布 npm 包或向外部项目提交 issue。
6. fixer 作为第二阶段：保留原 fixer 是否开启、分清 suggestion 与 normal/dangerous fix，检查跨规则多轮冲突与模块副作用，再决定哪些可默认自动应用。未达成这些条件时，可以只迁出诊断。

## 实际检查状态

| 检查 | 状态 | 范围 |
| --- | --- | --- |
| `npm --cache /tmp/oxlint-npm-cache ci --no-audit --no-fund` | passed | 从锁文件重装，417 packages，无 peer 绕过 |
| `npm run inventory` | passed | 11 profiles × 21 路径，真实有效配置及官方 migrate 输出 |
| `npm run experiment` | passed | 36 specimen + 12 fix mode 执行及独立 typed/format/matching/hybrid/副作用实验，断言关键反例及逐文件混合诊断 |
| `npm test` | passed | 5 个真实工具集成测试 |
| `npm run lint` | passed | 原型/脚本/测试，antfu config；研究用 Node test 关闭 test 插件 |
| `npm run benchmark` | passed | 200 文件、5 次重复、相同配置工作量与诊断核对 |
| `npm run summarize` | passed | 生成完整兜底/差异规则清单 |
| `go test ./...`, `go vet ./...`, `go build -o /tmp/zt-oxlint-research ./cmd/zt` | passed | 使用已有 Go 1.26.8 工具链；没有改 Go 源码 |
| `zt inspect --root /workspace/tools --json`、`packages/eslint-config` 及研究目录 | executed / read-only | 核实声明脚本；包装器无 test/lint scripts，inspect 不代表执行测试 |
| 真正消费工程、完整插件 conformance、persistent warm API、OS-cache-cold、typed performance | not run | 本仓库无消费工程；本实验没有提供这些证明 |
| 浏览器/E2E、生产部署、pilot detector inspections | not applicable / not run | 没有 UI、部署或 detector/planning 改动 |

配置/JS 运行拒绝属于**预期的差分实验结果**，原始失败文本保留，不冒充 passed lint。首次安装 JSX a11y peer 与 ESLint 10 冲突后，移除不需要的直接可选依赖，再以严格 peer 检查安装成功。没有尚未解决的安装/工具访问阻塞；生产等价性未覆盖的范围按上述限制保留为 unknown。官方来源、锁定 commit 和具体实现依据见 [SOURCES.md](SOURCES.md)。
