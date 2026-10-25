# GitHub App 创建、安装与系统凭据库

GitHub App 是 `zt evolve` 第一版远端能力的必需前置。客户端采用 GitHub App 的 OAuth Device Flow，不使用 PAT、普通 OAuth App 或 `gh auth` 作为隐式替代。App 安装和用户登录不授予 Agent 自动合并、发布或上传私人材料的权限。

**当前源码提供 Device Flow/Keystore 和 Cobra CLI，`zt evolve` 仍为计划。本文不是用户登录、正式发布或跨平台运行成功的回执。** 先创建并安装 App，用户登录后将 token 保存在系统凭据库；本机不需要 App 私钥或 Client secret。运行证据单独保留在 ignored artifacts 或 CI。

## 1. 创建 App

1. 打开 GitHub → 用户 Settings → Developer settings → GitHub Apps → New GitHub App。
2. 填唯一名称，例如 `zt-evolve-zeithrold`（已被占用时换名）。
3. Homepage URL 可以填写 `https://github.com/zeithrold/tools`，不是认证回调地址。
4. 开启 **Enable Device Flow**，保留 expiring user access tokens。此 CLI 不需要 Callback URL、Setup URL 或 Client secret；安装时的 Web OAuth 自动跳转不是必要步骤。
5. 暂不接收事件，将 Webhook 的 Active 关闭；本地 CLI 通过显式 API 调用提交/查询，无需常驻公网服务。以后增加 webhook 必须单独部署、校验签名并保存其 secret。
6. Where can this GitHub App be installed：只自己使用时选择 Only on this account。要分发给其他账号/组织安装时使用 Any account，并保持各 installation 的仓库范围最小。
7. 配置下表最小 Repository permissions 后创建 App。

| Repository permission | 权限 | 用途 |
| --- | --- | --- |
| Metadata | Read-only（GitHub 必需） | 仓库和安装身份 |
| Contents | Read & write | 创建候选 tree/commit/专属分支 |
| Pull requests | Read & write | 创建/查询 Draft PR |
| Checks | Read-only | 查询独立检查结论，不由提交者伪造 Checks 成功 |
| Actions | Read-only | 查询 workflow 状态与证据；不授予 dispatch/write |
| Issues / Discussions | No access | MVP 不自动发问题或讨论，未来按需要单独申请 |
| Workflows / Administration / Secrets / Members | No access | 不修改 CI、保护规则、密钥或组织设置 |

不要添加 App 到 ruleset/branch protection bypass list。GitHub 的 Contents 权限不是“仅允许写某个 Skill 路径”；路径、目标和候选权限还必须由受信 zt 策略限制。GitHub 也没有供本方案单独授予“能建 PR 但绝不能 merge”的权限开关：仓库保护规则必须要求人类审查，zt adapter 不实现 merge/enable-auto-merge 调用，不能声称上表权限在任意未保护分支上天然禁止合并。

## 2. 安装到选定仓库

1. App 设置 → Install App → 选择自己的账号 → Only select repositories。
2. 先只选 `tools`。原生 consumer Skills 去重不需要给 website/blog/showcase/memory 授予远端写权限。
3. 用于真实提交验收的 sandbox 仓需明确加入安装；创建 sandbox、扩大安装范围都需单独授权。
4. 记录非秘密元数据：GitHub host、**Client ID**、选定仓库；App ID 可留作管理标识。

App ID、Client ID 和 Installation ID 不是同一个值。Device Flow 的 `client_id` 必须是 Client ID，不能填写数字 App ID。登录不等于已安装：CLI 还要查询该用户可访问的 installation/repository，组织安装或 SSO 可能需要管理员批准。

## 3. 用户登录，不分发 App 私钥

CLI 向 GitHub 申请 device code，显示 user code 和官方验证页面；用户在浏览器中登录并同意授权。CLI 按返回的 interval 轮询，处理 pending、slow_down、拒绝、过期和取消。设备代码只在当前登录进程中使用，不落盘。

- 公开 Client ID 可随发布二进制携带；没有 App 私钥、Client secret 或用户 token 被编入包。
- access token 和 refresh token 保存在系统凭据库，不放 `.env`、配置、artifact、argv 或日志。
- 根据响应有效期刷新并成对替换 token；Device Flow 生成的用户 token 刷新不需要 Client secret。
- 撤销、refresh 过期或 Keystore 不可用时明确失败，需要时重新登录，不切换 PAT/其他账号。
- 用户授权令牌代表用户，而不是纯 App bot。权限是用户和 App 共同具备且可访问资源的交集，不能为无写权限用户制造上游写权限。

如未来要以 installation bot 进行服务端无人值守操作，应另建受控 signer；不能为了它重新要求所有 CLI 用户下载私钥，也不能把 signer 凭据嵌入公开发行包。

## 4. System Keystore 合同

这里的 System Keystore 指操作系统的受保护凭据服务，不是一个加密 JSON 文件，也不要求写 macOS 的系统级 `/Library/Keychains/System.keychain`。

| 平台 | 凭据后端 | 失败处理 |
| --- | --- | --- |
| macOS | 当前用户 Keychain，遵守其访问控制和交互批准 | 锁定/拒绝访问则 blocked，不绕过提示或扩大整个 Terminal 的访问权 |
| Windows | 当前用户 Credential Manager 或经评审的原生凭据接口 | 不可用/容量不足则报错，不写明文替代文件 |
| Linux | Secret Service（如 GNOME Keyring/KWallet 的兼容服务） | 无 session bus、无服务或锁定时 blocked；headless 不自动退回文件/env |

持久保存用户 access/refresh token 及其关联身份。磁盘非秘密配置至多存 host、Client ID、目标和 credential reference，不能从项目配置或候选覆盖身份以诱导跨账号访问。

凭据应绑定 host、Client ID 和用户；credential reference 不构成授权。运行时仍核验实际用户和 installation/repository 访问范围，不能只凭“已登录”推断目标可写。

token 每次使用时只临时读入内存，不导出给候选进程。系统凭据库保护的是静态存储，不表示 token 在应用内存中永不出现。并发刷新要串行化并原子保存新凭据，不能用旧 token 覆盖新状态。

原生后端容量、锁定和交互限制必须真实测试；不可用时不降级明文。Keystore adapter 的依赖和许可单独评审；macOS 若需 CGO，使用对应 runner 构建，不为静态跨编译牺牲保密性。

当前适配固定 `99designs/keyring v1.2.2`：macOS 只允许 Keychain，Windows 只允许 WinCred，Linux 只允许 Secret Service。Linux 使用已解锁的 `login` collection；仅提供其他命名 collection 的桌面环境不能直接当成已支持，需要配置相同集合或另行适配。可用性预检不保证之后写入必定成功；用户拒绝、锁定或存储失败仍须明确报错，不能输出 token 或落文件补救。

上游 keyring 在 Linux 的包初始化会连接 session bus，缺少 bus 时可能执行 `dbus-launch`。因此不能声称二进制启动完全没有本地 IPC/进程；本地 inventory 不调用凭据方法、不做 GitHub 认证、不执行项目检查或改写项目文件。原生测试使用隔离 bus/临时 Keychain/测试 item，不接触用户真实凭据。

## 5. 本地配置流程

使用包含认证功能的构建；开发构建需要显式 Client ID，Release 构建可使用已嵌入的公开标识：

```text
zt auth login --client-id CLIENT_ID --hostname github.com
zt auth status
zt auth status --probe
zt auth logout
```

- `login` 是显式用户认证及写入系统凭据库的操作，不在 inspect/plan 中自动启动。
- 不提供 `--token TOKEN`、PAT 或把秘密写入 `zt.json` 的入口。
- `status` 默认只检查本地状态，不联网、不输出 token，也不宣称远端安装健康。
- `--probe` 才显式核验远端身份/权限，必要时刷新 token；认证网络操作不是纯本地 inventory。
- `logout` 删除选定本地凭据，不把本地删除描述为撤销 GitHub 授权。远端撤销通过用户 GitHub 设置进行。
- 登录不等于候选批准；`submit` 仍绑定候选 digest、repo/base、公开 payload 和允许操作。

全局共享 Skills 的选择配置位于 `~/.agents/zt.json`；它不存 App 私钥。受信认证/发布策略也不从 consumer 项目 `zt.json` 自动读取。

## 6. 运行时认证链

1. 从公开发布配置或显式 flag 解析 Client ID，拒绝将数字 App ID 当作 Device Flow Client ID。
2. 登录时 POST `/login/device/code`，浏览器确认后 POST `/login/oauth/access_token` 交换用户 token；只打开可信 GitHub 验证地址。
3. 用户 token 受 App/用户/资源权限交集限制；repository 缩限参数不能代替对真实可访问目标的核验。
4. 启用过期 token（默认 access 约八小时、refresh 约六个月），依响应而非固定格式管理有效期和轮换。
5. 原生 HTTP adapter 访问 GitHub，禁止把凭据转发到其他 host 或不可信重定向。
6. 401/403、授权撤销、安装暂停、目标不可访问或 Keystore 不可用均 fail closed，不转用 PAT、文件或另一身份。

发布 CI 只读取 `ZEITHROLD_GITHUB_CLIENT_ID`；旧 App ID 变量已移除，不保留兼容入口。错误只报告类型问题，不输出配置值。Client ID 是公开客户端标识；App 私钥、Client secret、PAT 或用户 token 绝不可被当成构建配置。

## 7. 验收与故障定位

需要分开验证：

- token 已写入系统凭据库且读写/删除可用；
- 用户身份和 Client ID 正确，刷新及撤销按预期处理；
- 用户可访问的 installation 已启用且包含目标；
- token 的实际权限允许所需操作，没有越过受信发布策略；
- Draft PR 可创建；
- PR 触发了真实 required checks，结果属于对应 head；
- App 未配置保护规则 bypass，仓库必需审查生效，zt 不调用合并、自动合并或部署入口；不要把 App 的 PR 写权限误称为 GitHub 原生的“不可合并”权限。

App 事件通常可触发后续 Actions，不能把它与 workflow 默认 `GITHUB_TOKEN` 的事件抑制混同；但 fork 来源、仓库规则和 CI 条件仍可能影响执行，必须实际验收，不能仅凭创建 PR 的 HTTP 成功码宣称 CI 已运行。

本文不是用户登录或 PR 成功回执；真实账号授权必须由用户在浏览器完成，真实仓库写入仍需要目标和候选授权。Linux 容器内的假 token Keystore 测试也不能冒充 macOS/Windows 或真实 GitHub 登录验收。

## 官方参考

- [Registering a GitHub App](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/registering-a-github-app)
- [Installing your own GitHub App](https://docs.github.com/en/apps/using-github-apps/installing-your-own-github-app)
- [GitHub App user tokens / Device Flow](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app)
- [Refreshing Device Flow user tokens](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/refreshing-user-access-tokens)
