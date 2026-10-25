# zt evolve 设计与实施计划

状态：待评审；全部命令、数据结构和内部包名均为拟议接口，本次没有实现、提交或发布。

## 1. 目标与推荐方向

让 Agent 将真实任务中的可复用改进，通过可追踪候选、独立验证和 GitHub PR 送回工具仓库，形成跨项目反馈闭环。用户指定的第一个目标是 `zeithrold/tools`，但仓库坐标必须是消费者显式配置，而不是 runtime 中的默认业务特例或名称 allowlist。

推荐 MVP：**本地候选 → 纯读取计划 → 显式验证 → 幂等 Draft PR → 人类评审/合并**。Agent 可以自动生成与提交已授权范围内的提案，不代表它能自动决定验收、修改根规则、合并或发布。

GitHub PR 是代码/Skill 的审查与集成面，Checks 和 Actions artifacts 是执行证据面；Issue/Discussion 是尚无安全 patch 的想法入口。先打通一条端到端 Draft PR 链路，不首批建设 Projects 调度器、LLM 优化平台或通用训练系统。

与 [Skill 沉淀计划](skill-adoption-plan.md) 的关系：通用 Agent/Skills 安装到用户全局 `~/.agents/skills`，项目级只保留项目知识；先确立 skill-authoring、code-review、verification 的合同。evolution-proposals 使用同一种候选信封，evolve 负责落地机制，不再复制一套规则。完整自主进化研究快照在 `.artifacts/research/agent-engineering/`，不进入分发内容。

用户已确认：**从第一版起必须 GitHub App Device Flow，凭据只存 System Keystore**，不采用客户端 PEM/JWT、gh/PAT 或普通 OAuth App。创建和安装步骤及认证接口见 [GitHub App 配置指南](github-app-setup.md)；全局分发见 [global Skills](global-skills.md)。Device Flow 使用公开 Client ID，发布配置名为 `ZEITHROLD_GITHUB_CLIENT_ID`。

## 2. 现状与实现接缝

| 当前代码 | 已存在的能力 | 计划复用/限制 |
| --- | --- | --- |
| [`cmd/zt/main.go`](../cmd/zt/main.go) | 标准库 flag CLI、JSON 输出、check 的 context/cancellation | 增加独立 evolve 命令组；不改变 inspect/plan/sync --plan 的只读性 |
| [`internal/project/config.go`](../internal/project/config.go) | 严格 JSON schemaVersion 校验、模块/命令/Skill 显式配置 | 若扩展配置需兼容性测试；未知字段当前会失败，不能暗中塞新字段 |
| [`internal/check/check.go`](../internal/check/check.go) | 原生 argv 执行、显式 profile、超时、日志与 passed/failed/blocked/not_run | 复用原生检查语义，补候选内容身份；不是任意第三方代码安全沙箱 |
| [`internal/skills/skills.go`](../internal/skills/skills.go) | Skill 摘要、已管理目录、本地编辑保护 | 沿用全局分发摘要；项目知识不交给 evolve/sync 管理 |
| [CI workflow](../.github/workflows/ci.yml) | PR 验证、固定 action、失败证据保留 | 核验新增 PR 能真正触发 required checks；不从“PR 已创建”推导 CI 已通过 |
| [许可边界](../THIRD_PARTY_NOTICES.md) | 第一方 MIT 与上游原始许可分离 | 自动提案也必须通过具体来源/许可核查 |

## 3. 必须守住的边界

1. tools 只提供通用候选、来源、验证、预算和 GitHub 提交机制。任务业务数据、评分规则、隐藏测试、认证语义、部署目标仍属消费者。
2. 本地源码、会话和工具输出均可能不可信。模型提出 patch，不持有修改 evaluator、隐藏集、预算、审批策略或发布凭据的权力。
3. `zt inspect`、`zt plan`、`zt sync --plan` 不触发 evolve、测试、依赖安装或网络提交。
4. `zt sync` 只拥有显式选择的全局 Skill 目录和全局锁文件；evolve 不回写消费者 AGENTS、设计合同、项目知识 Skill 或本地编辑。
5. 当前用户另行授权 website/blog/showcase/memory 的重复通用 Skills 去重，不包含记忆数据或产品源。Ledger 仍不迁移；该一次性迁移不扩大 sync 的自动删除职责。不安装 hook、不创建 PR、不申请凭据；实现计划不是发布授权。
6. 自动化止于已授权的提案与送审；不默认合并、部署、发包、更新工具版本或开启权重训练。
7. MVP 仅接受 Skill 改进到显式选定的上游 `skills/<id>/`；根 AGENTS、CI、CLI runtime、包清单/lock 与认证文件不属于自动写路径。扩大 scope 需要单独设计与授权，不通过候选自改 allowPaths。

## 4. 命令契约（建议）

下列语法仅作设计；每个 subcommand 的位置参数和 flags 在实现测试中固定。默认不访问网络、不运行模型、不安装工具。

| 命令 | 输入与输出 | 副作用/权限 |
| --- | --- | --- |
| `zt evolve prepare --patch FILE --evidence FILE --upstream DIR --repo OWNER/REPO --base REF` | 将针对上游的显式 patch 和脱敏证据整理为不可变候选；输出 candidate ID/digest | 只写专属 `.zt/artifacts/evolve/`；不改原工作区、不联系 GitHub、不自动执行 patch 中脚本 |
| `zt evolve plan CANDIDATE --json` | 读取候选、可信策略和已缓存的上游基线，列出 paths、来源、缺口、原生命令、目标与发布内容摘要 | 严格只读，无网络、无安装、无测试，不创建 journal/cache；stdout 计划不叫验证通过 |
| `zt evolve check CANDIDATE --profile NAME` | 在显式准备的可信/隔离环境执行审核过的原生 profile，生成绑定内容的报告 | 显式执行；不自动扩权、不安装依赖；环境缺失 blocked，不退回凭据齐全的宿主执行 |
| `zt evolve submit CANDIDATE --repo OWNER/REPO --publish --reviewed-digest DIGEST` | 复查内容、权限和当前远端身份，创建专属分支、候选 commit 与 Draft PR，返回 URL/SHAs | 唯一写远端命令；必须明确授权，未就绪候选拒绝发送，不因 draft 而放宽上传隐私门 |
| `zt evolve status CANDIDATE --remote --json` | 读取对应 PR/head/check-run 状态并区分 local observed、remote pending、failed、passed | 默认只读；`--remote` 明确发出只读 GitHub 请求，不轮询到无期限 |

例如未来用户可明确调用 `zt evolve submit <id> --repo zeithrold/tools --publish --reviewed-digest <digest>`。示例中的目标是用户选择，不是 CLI 内置默认值。

prepare 首版接收已由 Agent/人生成的上游 patch，不从任意消费者目录自动映射业务文件到 tools。全局 `~/.agents/skills/<id>` 与全局锁可提供来源线索，但必须检查原锁和基线、去掉业务内容；项目知识不可按同名自动提升。不能将本地编辑自动等同于可再分发材料。

首版不内置模型调用。后续可加显式 provider adapter 生成候选，但不能进入只读命令，也不能改变既有授权和验收门。

## 5. 候选信封、身份与状态

建议 `schemaVersion: 1`，先满足单候选单目标，不先造通用事件平台。最少字段：

- candidate ID、创建版本、生成者声明、来源类型与适用任务类；
- upstream repo/host/base ref、完整 base commit/tree、目标 Skill ID、规范化路径与 patch/content 摘要；
- 来源 revision、消费者原始 managed Skill 摘要；若来源 dirty，明确描述被采纳内容，不能用 HEAD 冒充全部源状态；
- 问题、期望机制、非目标、适用与不适用条件、原始证据引用；
- 许可证状态、出处与允许再分发的判断；unknown 必须阻塞；
- 验证契约 ID/版本、profile、命令、环境/工具/模型/数据身份、结果与 artifacts 摘要；
- 发布内容清单、脱敏状态、风险、实际预算、授权记录；
- 外部 journal：远端分支、commit、PR ID/URL、最后已确认写步骤和未知结果。

内容身份与可变操作日志分离：签审绑定 canonical manifest、patch、目标、基线、验收合同、公开证据和权限范围。check 报告单独绑定这些身份；submit 记录不反过来改变候选 ID。任何受审输入变化使旧批准/相关验证失效，必须生成新候选或重新确认，不能静默重算继续发。

建议状态：

`prepared → checked → publish_ready → submitted → merged/closed`

验证结果另有 `passed / failed / blocked / not_run / unknown`；工作流状态不得抹去失败。Agent 提供的既有结果标记 `reported`，工具亲自观察的执行标记 `observed`；二者都不等于平台签发的不可伪造证明。

`publish_ready` 的门：路径/来源/许可/隐私/授权通过、要求的原生验证已执行、风险对应的独立评审已完成。缺环境可导出候选本地待处理，但不能用未验证结果走自动提交流程。经所有者明确批准的“仅诊断 Draft PR”作为以后单独 scope，不默认为绕过路径。

## 6. 策略、授权与 GitHub 目标配置

### 分离三类配置

- 消费者 `zt.json`：模块和原生验证 profile 等项目配置，不再选择通用分发 Skill；全局 `~/.agents/zt.json` 负责非秘密 Skill 选择。发布目标属于受信操作者配置，不能由项目目录默认接管。
- 操作者可信策略：允许的 host/repo/base/path、验证合同、权限上限、候选/成本/上传预算；不可由候选 patch 或消费者不可信输入覆盖。存储位置/权限需在 E0 明确并测试。
- 凭据：第一版使用 GitHub App Device Flow user token，access/refresh token 只存 System Keystore；普通文件只保存非秘密 host/Client ID、账号关联和 credential reference。客户端不持有 App 私钥或 Client secret，秘密不写入候选、环境、配置、命令日志或 PR。

首版提供 Keystore 与 Device Flow 的窄接口，用原生 HTTP adapter 取得和刷新用户 token；权限受 App、用户及可访问 installation/repository 的交集限制。不把 token 传给 gh、shell 或候选进程，不回退 PAT/普通 OAuth App/env/明文文件。macOS 使用用户 Keychain，Windows 使用原生受保护凭据服务，Linux 使用 Secret Service；不可用、锁定或拒绝访问时 blocked，不能为 headless 自动降级。用户允许多原生 runner 支持 CGO，不以纯静态构建为由弱化存储。

提供显式 `zt auth login/status/logout`：login 显示短码并引导用户浏览器授权，尊重 interval/slow_down/取消；status 默认本地，显式 probe 才联网核验/刷新；logout 删除本地凭据，不伪称已远端撤销。用户 token 默认有过期时间，刷新串行化并安全替换成对凭据。App 安装与用户授权是不同前置，纯本地 prepare/plan 不读取凭据或联网。App/Client ID/账号变化使相关发布批准失效。

配置提交路径与检查执行的信任边界不同：有权限发 PR 不意味着有权限执行候选脚本；能运行定向检查也不意味着能上传其原始日志。

### 手动与无人值守

- 手动：用户检查 plan/公开 payload 后，以 candidate digest、目标、base、scope 绑定的显式 submit 授权。
- 后续无人值守：操作者预先授予限时、限 repo/base/path/验证合同/并发/上传量的授权；每次写前核验仍有效。配置“自动运行”不是无限授权；撤销或预算耗尽停止新副作用。
- 执行者不能扩大授权；任务优先级高也不能跳过安全门。未知权限、配置损坏、审批加载失败均 fail closed。

## 7. 幂等 Draft PR 的实现策略

推荐 GitHub Git data/PR API，而非切换用户分支和批量 git add：

1. 从已审候选取得精确身份；GET 目标 repo/base、权限和已有候选分支/PR。先只读对账：候选、目标、分支内容与 PR 身份完全匹配时，即使 base 已前移，也返回既有 PR 和当前状态，不新增写入，不把旧验证说成当前基线验证。显式 GitHub host，不能把本地 `local` remote 当远端发布地址。
2. 只有仍需首次或继续写入时，才要求当前 base 与被验证基线一致；不符则停止并要求重新 prepare/review/check，报告已存在的部分远端副作用。不自动 rebase、更换 base、清理旧分支或把旧 head 证据当新验证。
3. 使用精确 base tree 创建仅含允许文件的 tree/commit。未修改文件沿用原树；拒绝删除/越界/二进制/符号链接等首版不支持操作。
4. 创建确定性专属分支，例如 `feat/skills-evolve-<id>`，commit/PR subject 遵守贡献命名。身份来自实际认证用户/App，不伪造作者。
5. 创建 Draft PR，正文只有已审公开摘要、固定来源、适用/非适用、验收结果/缺口、候选身份和证据链接。
6. 写 durable journal；如果网络超时发生在写之后，返回 unknown，并用 GET 按候选分支/marker 对账。不能盲重试建第二个 commit/PR。
7. 同 candidate、同目标的重试返回已有 PR；同分支存在意外内容/他人写入时冲突停止，不 force、不覆盖、不自动关闭他人 PR。

候选主键包含目标和基线；并发提交同一候选时依赖远端 ref 唯一性及对账，不先造全局锁。关闭/拒绝的提案不自动重新打开；修复后创建新候选并关联旧 PR。后续更新自己的 PR采用新增候选/明确续接合同，不能假借幂等重写已评审内容。

远端 commit 是未来 submit 的预期副作用，本次“先不提交”阶段不执行任何该流程。API 创建 commit 也不是绕过授权的“无提交模式”。

## 8. GitHub 能力分工与安全 CI

| 能力 | 采用方式 | 不采用的替代解释 |
| --- | --- | --- |
| Draft PR | 可执行改进的标准送审面，CODEOWNERS/人类评审/required checks 按目标仓规则 | draft 不是安全审计通过，也不授予自动合并 |
| Actions/Checks | 精确 head 的原生验证与负例，保留失败 artifacts | workflow 文件存在不等于执行；job 名相同不等于结果适用 |
| Actions artifacts | 有保留期的脱敏执行证据；过期保留状态和摘要，不伪造链接可用 | checksum 不等于行为正确，hash 不能替代脱敏 |
| Issue（后续可选） | 无法通用化、缺许可或缺 patch 时，由单独授权发送最小问题描述 | 不上传整个会话、源仓业务文件或失败日志 |
| Discussion/Projects | 仅在实际需要知识讨论/队列管理后评估 | MVP 不依赖它们，不自动创建看板、标签或评分机器人 |

需要明确测试 PR 来源对 CI 触发的影响：默认 `GITHUB_TOKEN` 产生的事件常不会触发后续 Actions；首版使用 App installation 身份，在沙箱仓验证能触发目标验证，不以 PAT 作为救场路径。不能把“API 成功”当作 CI 已排队。

任何外部候选执行都在无秘密隔离环境；禁止 `pull_request_target` checkout 并执行不可信 head，禁止把带写 token 的发布 job 与候选执行混在同一信任面。不得为拿绿灯关闭安全设置或降低 required checks。新增 workflow 固定 action/toolchain 版本并保留失败证据；现有 Verify/Deploy 和控制面部署检查不受本功能改写。

## 9. 隐私、路径与验收安全

自动化的公开边界必须比本地 evidence store 窄：

- 默认不发送完整会话、模型推理、环境、终端日志、源码快照、用户数据、绝对本机路径和私有仓库链接。
- 来源材料先标可信/不可信及许可；摘要、向量化、LLM 改写不能洗掉污染标签。
- 只使用明确列出的公开 payload；本地 secrets 扫描是辅助，不是“保证无秘密”。人类批准或可信操作策略必须覆盖实际将上传的内容。
- evidence URI 只允许策略选定来源；MVP 不任意 fetch URL，不跟随重定向读内网，不把凭据转发到候选指定 host。
- patch 和 evidence 限总字节、文件数和路径；拒绝 `..`、绝对路径、Windows 分隔符/大小写冲突、symlink/hardlink 外逃、mode 欺骗和控制字符；以真实文件系统验证而非仅字符串 clean。
- evidence 不能读取 candidate 根以外任意文件；路径检查后到使用前检测内容变化，失败关闭。
- hidden tests、evaluator 和保留集由受信宿主持有。候选开发者和自动优化器不能修改评分规则、查看封存答案或自签可信验收。
- 执行 profile 来自已审策略/基线，不从候选中的新脚本名猜测通过条件。原生工具链负责 checks；宿主执行权限不足则 blocked。
- 不执行任意候选在凭据齐全的宿主机；现有 internal/check 的超时和 argv 安全不等于 OS 沙箱。MVP 可先只接受可信操作者审查后运行的候选；无人值守执行依赖后续隔离 adapter。

## 10. 分阶段实施与验收

### E0 / P0：冻结最小合同

任务：确认候选 schema、受信策略来源、内容摘要与审批失效条件、MVP Skill 写范围、repo/base、所需验证和公开 payload；首先创建/安装启用 Device Flow 的最小权限 GitHub App，落实 Keystore、短码登录、刷新和真实账号/资源范围核验。盘点现有 check 报告如何绑定源码，不把真实 token 写入 fixture。

产物：小型接口与威胁模型、synthetic fixtures、幂等/失败时序测试表；不先写模型、向量库或任务调度器。

验收：单一候选状态所有者；手动授权语义清楚；source/payload/evaluator 边界独立；每个新增字段有当前消费者。Keystore fake 和各平台原生存取/拒绝分别验证；错误、JSON 和日志不含 token；真实登录等待用户浏览器授权，不用假 token 测试冒充真实认证通过。

### E1 / P0：离线 prepare/plan

任务：新增 CLI 路由和候选读写，规范化摘要、来源许可、路径与隐私门，JSON 输出和错误语义；使用临时 synthetic 上游，不修改 pilot。

验收：prepare 只写自有 artifact 目录；plan 对文件/进程/网络无副作用；脏工作区不被改；未知 schema/路径/许可 fail closed；同输入产生相同内容身份；目标变化使批准失效。

### E2 / P0：绑定身份的验证

任务：复用 native check/profile 语义，在明确可信或隔离环境运行，保存新旧候选证据；要求 fresh 独立评审的风险类型只读审查；上层测试保持独立。

验收：无命令/缺依赖/取消/超时为 blocked/failed/not_run 而非 pass；变更 patch、环境、验收合同后旧结果失效；日志完整保存在本地，公开摘要另审；不自动安装、不扩权。

### E3 / P1：幂等 Draft PR

任务：新增通用 GitHub HTTP adapter，使用 E0 的 App/Keystore 身份；用 API 创建限定范围的 tree/commit/ref/PR；实现 journal 和超时后对账。

验收：fake server 覆盖重试与冲突；随后在用户明确授权的 sandbox 仓完成一次真实 prepare → check → submit → status，确认 CI 触发、PR 内容、并发重试不会重复。未授权时只做本地 adapter 测试，不拿真实 tools 仓“试一下”。

### E4 / P1：目标仓验证接入

任务：维护者按实际风险配置 PR gate、CODEOWNERS、Checks/Artifacts；不引入新的自动 merge 或发布权限；确认凭据不会出现在不可信执行中。

验收：伪造通过报告、缺证据、未完成检查不能被算作 verified；新 head 不复用旧 head 绿灯；公开材料经过脱敏且可定位；pipeline 保留失败原因。

E1—E4 完成才构成 MVP：一份通用 Skill 改进能够在明确授权下从消费者提出并自动形成可审查 PR。只实现 schema/本地命令或 API mock 不能称“自主进化闭环已运行”。

### E5 / P2：受限自动化与 GitHub 扩展

在真实 MVP 证据后，再实现限时无人值守提交、去重/速率预算、Issue 入口、来源撤销和成本计量。自动提案不自动激活；保持所有者掌握合并和发行。

只有独立任务收益、退化检测和恢复机制完备后，才另行实验离线 prompt/Skill 搜索。程序自改、权重训练、自动消费未审 PR、自动重写根规则不在当前路线。

## 11. 测试清单

- CLI：help、未知 subcommand、flags/位置参数、JSON 稳定性、错误退出、stdout/stderr、取消。
- 内容与授权：dirty 源、基线变更、审批过期、目标/host 变化、候选篡改、伪造 evidence、许可 unknown、目录外读取、秘密输出。
- 验证：原生成功/非零/不存在/未运行/超时、完整日志、子进程清理、未授权安装/网络被阻止、先后门禁。
- 认证：Keystore 锁定/不存在/拒绝、超大 credential、身份引用篡改、短码过期/拒绝/slow_down/取消、token 刷新与并发轮换、账号/安装撤销；无 PAT/env 回退且无秘密输出。
- 远端：无认证、权限不足、token 目标/权限过宽、fork/目标错误、base 移动、branch 竞争、PR 已有/已关闭/已合并、限流。
- 故障：tree/commit/ref/PR 每一步写成功但响应丢失；通过只读对账继续，无重复、无 force、无他人覆盖。专测“PR 已创建但响应丢失 → base 前移 → 重试”仍只读返回既有 PR；仅有部分副作用且仍需写入时按新基线阻断。
- 只读回归：对 inspect/plan/sync --plan/evolve plan 做前后文件快照、fake subprocess 与网络观察，证明不运行模型/测试/安装/提交。
- 隔离：拒绝候选修改 evaluator/策略/工作流；发布 job 不能执行候选代码；token 和秘密不进入 artifacts/PR。
- 端到端：合成 consumer 和 upstream、Skill local-edit 冲突、真实授权 sandbox PR 与 CI；不迁移两 pilot。

新增 Go 代码至少运行 `go test ./...`、`go vet ./...`。改 detectors/planning 时追加本地 pilot 只读 inspection。修改原生包/前端 workflow 时跑对应原生检查，保留完整失败证据；本计划不通过 `inspect/plan` 代跑它们。

## 12. Skill 选择与待确认决定

E0/E1：使用 task-planning/skill-authoring 的合同思想，但新 Skill 未落地前不能声称已调用。局部 CLI 单测不需要整队专家；涉及凭据、上传、路径执行、授权的 E2—E4 必须有安全与正确性互补的独立评审。紧急修复先收窄可逆路径，不绕过许可、授权、验收或脱敏。

建议确认后再开实施分支：

1. 已确认：第一版必须 GitHub App Device Flow + System Keystore + Draft PR；不保留 PEM/JWT、gh/PAT 或普通 OAuth App 过渡。Client ID 已由用户配置 Actions secret，真实用户登录和 CI 触发验证仍是独立必需门。
2. 是否接受首版只提交通用 Skill 目录，不自动修改 root AGENTS/CLI/workflows？推荐接受，扩大范围另立评审。
3. 哪个 sandbox repo 可用于最终真实 PR 验收，谁批准公开 payload？本次不创建测试仓或 PR。
4. 原生验证在何种可信/隔离环境执行，模型行为 eval 用什么预算与保留集？缺失则保持 blocked，不用自报成功补齐。

本计划不要求在聊天或普通文件中提供任何密钥，也不批准之后的合并、发包或部署。用户已明确授权的四仓重复 Skills 迁移单列执行，不涉及记忆文件。
