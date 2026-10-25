# 全局通用 Skills 与项目知识

## 所有权

通用 Agent 工作流、设计/测试/验证 Skill 由 tools 的 `skills/` 提供，安装到当前用户的 `~/.agents/skills/<id>/`。项目级 `.agents/skills/` 只保留项目知识：业务语义、架构边界、专用工具和仓库特有流程。项目 AGENTS 和设计合同仍是当地权威，不被全局 Skill 覆盖。

“全局安装”只代表可被支持该路径的 Agent 发现，不代表每次加载全部正文，也不保证每个宿主自动识别该目录。使用前核验宿主发现能力；不要为不支持的宿主随意再复制一套内容造成漂移。

## zt sync 重组合同

当前全局分发接口：

```text
zt sync --global --skills go-testing,js-ts-testing,ui-foundation,ui-web,ui-flutter,frontend-engineering,frontend-verification --plan
zt sync --global --skills go-testing,js-ts-testing,ui-foundation,ui-web,ui-flutter,frontend-engineering,frontend-verification
```

`sync` 默认就是全局范围，`--global` 用于让命令更清晰。显式 `--skills` 是本次选择，不会偷偷修改用户配置。若需持久选择，非秘密配置放 `~/.agents/zt.json`：

```json
{
  "schemaVersion": 1,
  "skills": [
    "go-testing",
    "js-ts-testing",
    "ui-foundation",
    "ui-web",
    "ui-flutter",
    "frontend-engineering",
    "frontend-verification"
  ]
}
```

以后执行 `zt sync --plan` / `zt sync` 使用该全局选择。没有配置或显式选择时应提示配置，不从当前项目偷偷选择，不默认安装目录里的全部新 Skill。

- 管理锁：`~/.agents/zt.lock.json`；只描述全局被 zt 管理的目录。
- 项目 `zt.json` 保留模块和原生命令配置，不再承载 shared Skills 分发选择。
- 旧 `zt sync --root <project>` 被拒绝并提示迁移，不能将旧自动化静默解释成另一种写入范围。
- plan 不创建全局目录、锁、配置或缓存，不安装依赖/执行测试/访问网络。
- 实际 sync 只写显式选择的全局目录和自己的锁，不覆盖非管理目录或本地修改，不自动删除未选择 Skill。
- 整个目录（LICENSE、references、scripts 等）一致分发；不只复制 SKILL.md。
- 全局 Skill 不授予网络、公开写、Keystore 或部署权限；GitHub App 见 [单独指南](github-app-setup.md)。

此重组是分发边界变更；验证工具、项目原生命令和消费者业务行为不随之更改。

## 适用时机

全局索引只提供名称和触发描述，正文按任务加载。沿用 [Skill 路由矩阵](skill-adoption-plan.md#3-先规定最佳适用时机)：

- 问答/机械文档：只用必要上下文，不默认规划、多 Agent 或全量测试。
- 局部修复：领域知识 + 定向回归；不清楚原因时才加调试。
- 跨模块任务：规划与接口先行，有真正独立工作才并行。
- 高风险授权/迁移/并发：增加独立评审与真实入口证据；紧急不免除这些门。
- 项目知识仅在对应项目/领域读取，不将品牌、业务 schema、生产环境或秘密提升为全局规则。

## 既有项目迁移

website、blog、showcase、memory 是本次用户选择的迁移对象，不是 zt runtime 的命名特例。只迁移重复 Skill，**不迁移或删除任何 memory 数据、记忆文件、产品代码、测试或设计合同**。

顺序必须是：

1. 清点每仓适用规则、Skill 完整目录、分发锁、当前本地修改以及对相对路径的引用。
2. 对照 tools 当前目录逐文件比较，记录匹配、差异和项目特有内容；归属不能只看名称。
3. 先安装/核验完整全局副本，确认不覆盖用户已有全局 Skills。
4. 将通用 Skill 的本地独有改动审查后合并到对应通用来源；业务特有内容转为项目知识，不能丢弃。
5. 只删除已确认重复且已由全局覆盖的项目副本；移除项目 shared `skills` 选择，不修改模块/profiles。若项目的原生证据元数据仍依赖旧锁，则保留明确为空的退役锁作为该合同输入，不能直接删文件破坏验证；zt 不再写该项目锁，后续移除依赖单独审查。
6. 更新受影响文档路径，保留项目知识 Skill、项目 AGENTS 和其他工具的目录。
7. 用只读 inspection、内容比对和原生静态检查确认结构；项目 build/browser/部署验证没有运行时如实标注。

未解决差异或已有未处理的本地修改时停止该目录的删除，其他独立迁移可继续。自动删除不是新的 `sync` 职责；它是本次显式授权、经清单确认的迁移。

运行清单和哈希回执保存在 `.artifacts/`，不是长期运行规范。删除前重新检查内容身份，防止盘点后被用户修改。
