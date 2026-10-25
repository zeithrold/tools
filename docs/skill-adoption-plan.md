# Skill 沉淀计划

状态：新工作流 Skill 的沉淀仍是待实施计划；现有七个 Skill 的全局分发重组已另行实现并安装。这里不将全局安装称为已完成全部候选沉淀。

## 1. 目标、边界与交付

将 Agent 工程调研中经过证据筛选的好方法沉淀为 tools 的可复用 Skill。这里的“全部”指有价值的机制都有明确去向，不是把所有上游文件复制一份、把相似方法计为独立能力，或在每个任务中启用全部 Skill。

交付包括：来源与许可证清单、候选处置矩阵、增强后的现有 Skill、必要的新 Skill、按风险和阶段选择的路由说明、行为回归样例，以及完整目录全局分发测试。通用 Agent/Skills 只安装到用户全局 `~/.agents/skills`，项目级只保留项目知识。项目业务规则、认证、部署目标、品牌、schema 和生产预算仍由消费者持有。

本轮完整调研保存在忽略目录 `.artifacts/research/agent-engineering/`，仅为本地快照，不进入发布包。长期来源和归属记录应随实际采纳的材料维护；不能让分发的 Skill 依赖本地调研报告才能使用。自主进化命令见 [evolve 计划](agent-evolution-plan.md)。

非目标：

- 不迁移 Ledger、不替换 ledger-tooling。用户现已明确授权 website/blog/showcase/memory 的重复通用 Skills 去重；只处理此范围，不删除记忆文件或产品源。
- 不增加项目名称特判、遗留偏好映射、统一生产部署流程。
- 不改变现有发布、同步所有权或字体/图标等既定政策；发现相邻冲突单列后续事项。
- 不将模型建议、Markdown lint 或一次静态扫描称为行为验证。
- 不自动安装、执行、提交或发布第三方 Skills。

## 2. 当前事实与复用点

| 当前来源 | 已有能力 | 实施含义 |
| --- | --- | --- |
| [`skills/`](../skills/) | go-testing、js-ts-testing、ui-foundation、ui-web、ui-flutter、frontend-engineering、frontend-verification | 先扩展已有职责，避免再造前端、无障碍和测试“总纲” |
| [`assets.go`](../assets.go) | `go:embed skills` 嵌入完整目录 | references、scripts、许可证都会影响分发体积，不能直接塞入调研 corpus |
| [`internal/skills/skills.go`](../internal/skills/skills.go) | 显式选择、目录摘要、碰撞检查、保留本地编辑、同步回滚 | 分发目标改为用户全局，复用安全机制，不增加旁路安装器 |
| [`internal/project/config.go`](../internal/project/config.go) | 严格配置和明确选择 | 项目保留模块/原生命令；全局 `~/.agents/zt.json` 或显式 CLI 列表负责通用 Skill 选择，不默认全选 |
| [`CONTRIBUTING.md`](../CONTRIBUTING.md) | 长期文档与运行证据分离 | 对照实验和截图保存在 ignored artifacts/CI，不放入 Skill 正文 |
| [`THIRD_PARTY_NOTICES.md`](../THIRD_PARTY_NOTICES.md) | 保留第三方原始许可 | “改写过”不自动消除衍生材料义务 |

## 3. 先规定最佳适用时机

分别判断任务紧急程度、后果风险、耦合度和当前阶段。紧急不等于低风险，文件少不等于改动简单；不能用固定文件数替代判断。下表是选择原则，不是要新增一个复杂调度引擎。

| 任务 | 最小选择 | 何时加码 | 何时不加码 |
| --- | --- | --- | --- |
| 简单问答、纯说明、机械文案修正 | 适用的仓库规则；定向内容/链接检查 | 发现行为、合同或安全含义时重新分类 | 不默认规划、派团队、跑全量测试或独立专家 |
| 局部、可回退的行为修复 | 对应领域 Skill + 原生定向回归 + 完成证据 | 原因不清时 systematic-debugging；非平凡行为加独立 review | 依赖紧密时不硬拆并行任务 |
| 跨模块功能、接口或状态变化 | task-planning + 领域 Skill + verification | 有独立可交付工作时 agent-team；影响面要求 code-review | 不先为每个微步骤建文档/Agent |
| 授权、持久化、迁移、并发、公开契约 | 明确范围/恢复方案 + 领域审查 + 互补独立评审 | 按真实风险补集成、负例、重启/恢复和兼容验证 | 不因赶时间删门禁，也不把所有理论风险做成新架构 |
| 紧急故障 | 先最小定位、可逆缓解和真实入口复测 | 不可逆或高风险动作先确认权限；保留并行诊断机会 | 暂缓可选美化与重型整理，不免除权限、证据和必要评审 |
| 性能问题 | performance-engineering + 对应领域 | 有真实瓶颈后才做 profiler、批处理或缓存实验 | 没有基线时不预先 memoize/cache |
| Skill/规则改动 | skill-authoring + verification + 独立评审 | 影响范围扩大或涉及权限时增加压力、负例与回归 | 文字变更不可一律当成纯文档免测 |
| 经验提升/进化候选 | evolution-proposals | 有可复用证据且独立验证可执行时进入 evolve | 单次偏好、秘密、业务数据、未解失败不提升为通用规则 |

每个 Skill 的 description 和正文都要回答：何时用、何时不用、需读取什么、允许做什么、产出是什么、何时停止或升级。实现阶段不重新反复加载未变化的同一份正文；阶段切换、内容变化或出现新风险时再加载相关知识。

选择顺序：任务边界和授权 → 必要的问题定位/规划 → 领域实现 → 风险触发的独立评审 → 对应验证 → 仅有可复用证据时提炼经验。独立工作可以并行；依赖结果的下一步不以委派伪装并行。

Skill 是指导，不提供授权。显式用户要求和适用项目规则优先；所有选择都不得弱化原有必需检查。非语义任务可以短路径完成，但不能伪装成已跑行为测试。

## 4. 候选处置矩阵

以下名称为拟议目录名，不代表当前已可用。合并边界先按职责确定，再决定是否拆 references；不要为了上游名称一一对应而重复建 Skill。

| 优先级 | 能力/拟议归属 | 主要借鉴来源 | 采纳内容与最佳时机 |
| --- | --- | --- | --- |
| P0 | `task-planning`（新增） | OpenAI Agents Python、Superpowers、Spec Kit、OpenSpec、DeerFlow | 非平凡变更的目标/非目标、支持范围、语义消费者、接口、依赖、可执行验收；小任务不强制文档包 |
| P0 | `verification`（新增） | Superpowers、OpenHands、Qwen、Kimi、Mastra、Hermes | 跨栈证据语义、真实入口、old-fail/new-pass、隔离与未运行状态；栈命令继续由原生测试 Skill 拥有 |
| P0 | `code-review`（新增） | OpenAI Agents Python、Claude plugins、BMAD、ADK、Qwen | 固定内容、风险分级、独立上下文、发现与复核分离、有限返工；普通改动不启用全体专家 |
| P0 | `skill-authoring`（新增） | Superpowers、Vercel、Hermes、CrewAI、wshobson | 触发/非目标、按需引用、完整目录、许可、无技能对照和压力测试；写/改 Skill 时用 |
| P0 | go-testing、js-ts-testing（增强） | Mastra、ADK、Pi、LangGraph | 行为契约、边界 mock、定向反馈与最终检查、平台原生验证；保留现有 fuzz/mutation 语义 |
| P1 | `agent-team`（新增） | GSD、wshobson、OpenAI Agents Python、Codex/OpenCode | 独立问题域、文件/接口所有权、上下文与工作区区分、通知、收集整合、预算；可独立交付才派发 |
| P1 | `systematic-debugging`（新增） | Qwen、Superpowers、Reflexion 的局部反思 | 假设、定点证据、证伪、根因、回归；原因已确定的小修不强行完整仪式 |
| P1 | `backend-engineering`（新增） | OpenCode、ADK、Mastra、DeerFlow、Cline | 信任边界、契约所有者、授权、作用域、幂等/恢复、生成漂移；具体架构按项目选择 |
| P1 | ui-foundation、ui-web、ui-flutter（增强） | Anthropic、BMAD UX、OpenHands | 小型设计决策包、视觉/体验职责、交互状态、焦点/窄屏/reduced-motion；已有设计合同优先 |
| P1 | frontend-engineering、frontend-verification（增强） | Mastra、Vercel、Next.js、OpenHands、Kilo、web-quality | 设计系统责任、真实 SDK + 边界 mock、语义 token、请求依赖、数据投影、有效 UI 与缓存新鲜度；框架专属引用按需加载 |
| P2 | `performance-engineering`（新增） | OpenClaw、Gemini、AutoGen、web-quality | 先定义负载/基线/指标，删除浪费再批处理和缓存；区分源码假设、lab、field |
| P2 | `evolution-proposals`（新增） | OpenClaw、Hermes、LangMem、GEPA、DSPy、SICA | 证据、候选、独立验证、版本与人工激活；不把权重训练或宿主代码自改默认化 |
| P2 | 各 Skill 的参考段落，不另立流程 | ECC、nanobot、Goose、Roo、Continue、Agent Zero | 证据复用、Ready/范围前置、角色能力、检查耗时、局部规则路由；去掉产品与平台强假设 |
| 不采纳为通用强制规则 | 仅留调研反证 | Ruflo synthetic workers、Agno 未证实自动覆盖、危险权限示例 | 不采纳占位能力、自报效果、统一覆盖率、强制全程多 Agent、自动部署和审批绕过 |

每条研究机制必须最终标记为：采纳到已有 Skill、新建、合并同源、项目专属不采纳、待证据或许可证阻塞。不能以“全都很好”为由跳过处置；不能因暂缓就丢失候选。

## 5. 来源、许可与移植契约

第一批产物先建立维护用处置清单：源仓、完整 commit、文件/行、材料身份（自用/分发/实现）、机制、目标 Skill、采用/改写/拒绝理由、许可证路径及状态、验证方法。该清单维护长期采纳关系，不堆本轮测量或运行回执。

- 许可证必须核实到具体文件和固定版本；未知或不兼容则阻塞复制，不能假定 GitHub 上的 Markdown 都是 MIT。
- 直接复制或衍生时保留适用许可证、作者和声明，更新第三方清单，并验证同步后完整保留。
- 仅吸收通用方法时仍记录来源；不得把厂商分发模板写成其内部生产实践。
- 去除硬编码模型、命令、目录、Sentry 等服务、生产部署、危险沙箱参数和未经授权的公开发帖默认值。
- 同源改编只算一个机制证据；历史 Codex 规则、已迁移 GSD/Next Skills 与当前规则分别记录。
- consumer 专属值必须是显式配置/接口；不能用项目名、域名或名称白名单恢复被拒绝的业务例外。

## 6. 分批实施与依赖

### S0 / P0：盘点与选择契约

范围：只建立采纳矩阵、来源/许可记录、路由说明和行为案例设计，尚不批量复制 Skill。

验收：

- 七领域和自主进化各机制均有去向，包括拒绝/暂缓项。
- 明确现有七个 Skill 的唯一职责和引用边界。
- 路由案例包含：纯文档、小修、紧急低风险、紧急高风险、跨模块、Skill 改动、性能、无证据进化请求。
- 每个案例说明最小加载集、跳过项及其理由，防止“所有任务都规划/派人/全量测试”。

### S1 / P0：可信工作流基础

实现 task-planning、verification、code-review、skill-authoring，并定向增强 Go/JS-TS 测试规则。以一份小变更为一批，避免一次将全部强规则注入消费者。

验收：明确触发与非目标；完整目录无悬空引用；完成/未跑语义一致；无测试或缺独立 reviewer 时报告缺口，不伪造通过；评审、修复和发布权限分开。

### S2 / P1：协作与领域标准

实现 agent-team、systematic-debugging、backend-engineering；增强五个现有 UI/前端 Skill。依赖 S1 的证据与评审合同，可按独立文件所有权并行，不并发修改同一权威规则。

验收：共享文件/接口冲突有处理人；后台完成走通知；子任务不重复控制者启动流程；前后端不采用一套固定项目架构；不改变现有项目设计/部署政策。

### S3 / P2：性能与进化提案

实现 performance-engineering 和 evolution-proposals。后者可先生成可审查候选文件，不等 evolve 网络提交功能完成；最终接入 evolve 的本地候选格式，避免两套账本/状态机。

验收：性能结论带条件且不引用未复现倍数；经验来源和适用域清楚；生成者无权改验收、全局预算或根规则；无独立收益证据时只称提案。

### S4：分发与渐进使用

更新目录说明、README 和必要的 synthetic examples，复用嵌入和完整目录同步，但以全局配置/锁作为唯一分发状态。`zt sync` 默认用户全局范围，旧项目 `--root` 写入拒绝并提示迁移；不默认安装所有新 Skill，不新增隐式依赖解析。按照用户已指定的四仓清单去重，不将项目名字硬编码到 runtime。详见 [全局分发合同](global-skills.md)。

验收：临时 HOME 能安装/更新；不同项目 cwd 使用同一全局状态；plan 不创建 HOME 内容；未选择的目录、项目知识、AGENTS、设计合同和本地编辑都不变；冲突、取消与失败恢复正确；许可证随目录交付。四仓删除前逐文件证明重复，并先验证全局副本，保留 memory 的产品 Skill 源与所有记忆数据。

## 7. 验证方法与完成门

结构验证和行为验证分别报告，任何一个都不能替代另一个。

1. 结构：frontmatter、目录名、描述触发/排除、引用文件存在、可携带资源、许可证、无秘密、无业务硬编码。
2. 行为：无新指导 baseline 与候选对照；每次独立上下文，先固定案例和评分规则，再执行；记录原始结果、错误激活、漏激活和越权。
3. Skill 路由：检查不同紧急度/风险/阶段是否选择了最小充分集合，而非仅检查“说出了 Skill 名称”。
4. 压力：赶时间、已有修改、工具缺失、未跑测试、Agent 自报成功、用户只要求规划、消费者品牌冲突等场景。
5. 权限负例：未知工具不能臆造；不能绕过审批、安装工具、跨目录编辑、发布或修改验收来追求成功。
6. 成本：记录回合、时间、工具/模型使用和实际可获得费用；未测则 unknown。采用前定义相对 baseline 的预算和退化条件，不宣称通用节省倍数。
7. 分发：复用 internal/skills 单元测试和临时 HOME 集成；验证所有新增目录在二进制/全局同步后资源完整、项目无写入、不影响个人全局 Skill；四仓去重另做删除前哈希复核与项目配置检查。

固定少量重复只作为筛查，不构成统计充分性。模型结果不稳定时扩大样本或保持待评审，不从一次成功宣布能力提升。Skill 规则变更的行为报告进入 ignored artifacts/CI evidence，而非长期正文。

修改 Go/CLI/同步代码时运行 `go test ./...`、`go vet ./...`；若涉及 detectors/planning，还必须对 pilot 做本地只读 inspection 并说明实际命令和结果。涉及原生包则运行其声明的 pnpm checks；不在只读 inspect/plan 中偷偷执行。

## 8. 长期来源锚点与未决项

- [OpenAI 风险分级评审](https://github.com/openai/openai-agents-python/blob/0dc4cf152bff1f50bdc0e41f28808563406545b2/.agents/skills/implementation-final-review/SKILL.md#L10-L46)
- [Superpowers 任务计划契约](https://github.com/obra/superpowers/blob/2ce8aa15d05bd57213e6f746f953aafb7ded7a8b/skills/writing-plans/SKILL.md#L95-L161)
- [OpenHands 验收独立性](https://github.com/OpenHands/OpenHands/blob/b7fdc6c312039a0b840fdcd34a5c23ca7e65c218/AGENTS.md#L69-L73)
- [Mastra 前端所有权](https://github.com/mastra-ai/mastra/blob/d49143a299757859bdbca93aa4b688c1383ace1e/.claude/skills/mastra-frontend/SKILL.md#L8-L35)
- [OpenClaw 性能工作顺序](https://github.com/openclaw/openclaw/blob/db38cc1066957e278d529d531160bb2773163682/.agents/skills/openclaw-performance/SKILL.md#L10-L58)

实施前确认：每份拟复制材料的实际许可、宿主支持的 Skill 激活/权限字段、可用于行为测试的模型与预算。目录名可在 S0 小范围评审后调整；若出现真实职责重叠，应合并而非维护多个“总纲”。

本计划不批准实现后的提交、推送、合并或发布；当前四仓重复 Skills 迁移已有明确授权，其他消费者迁移仍按具体任务授权执行。GitHub App 从第一版必需，密钥仅进系统凭据库，见 [配置指南](github-app-setup.md)。
