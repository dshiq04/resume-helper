# ResumeTrail 项目 Bug 审查报告

- 审查日期：2026-11-24
- 代码目录：`F:\resume-helper\resume-helper`
- 审查范围：Go 后端（`main.go`、`cmd/`、`internal/`）与 React 前端（`frontend/src/`）
- 验证环境：Windows 10/11、本地时区 UTC+8、Go 1.26、Node.js + TypeScript

> 说明：本报告只记录问题，**未修改任何业务代码**。审查期间产生的临时探针测试、临时数据库和测试应用已全部删除（见文末“测试数据清理”）。

---

## 一、结论摘要

| 级别 | 数量 | 问题 |
| --- | --- | --- |
| 高（P1） | 1 | 日期范围筛选存在时区/时间文本比较错误，导致投递记录“查不到” |
| 中（P2） | 4 | 仅日期作为结束日期时整日被排除；繁中/日/韩翻译缺失；快速录入窗口分类不刷新；多窗口下闲置自动锁屏误触发 |
| 低（P3） | 7 | Headless 模式关闭窗口无效、切换数据库后旧会话仍有效、批量接口缺少上限、Excel 导入内存风险、简历删除残留 `.deleting` 文件、README 引用了已删除的恢复脚本、`Start` 注释与实现不一致 |

已确认的详细问题共 **12 项（均已修复，见第六节）**，另有 6 项需要环境确认或属于代码卫生的观察项（见第三节）。

---

## 二、详细问题

### 1. 【高】日期范围筛选时区处理错误，记录会“凭空消失”

**位置**

- 时间写入：`internal/database/database.go:83-89`（`NowFunc` 返回 `time.Now().Local()`）
- 投递时间：`internal/server/applications.go:173-175`（`AppliedAt = time.Now()`）
- 列表筛选：`internal/server/applications.go:90-130`
- 统计筛选：`internal/server/statistics.go:18-45`、`:62-76`、`:627-646`
- 前端发送参数：`frontend/src/hooks/useApplications.tsx:123-124`、`frontend/src/pages/StatisticsPage.tsx:74-80`

**现象**

在 UTC+8 环境下，当天 16:00 之后创建的投递，使用前端“今天”日期筛选时查询结果为 0；不筛选时能正常看到。

**根因**

1. SQLite 驱动 `glebarez/go-sqlite` 将 `time.Time` 按 **带本地时区偏移的文本** 保存，例如：
   `2026-11-24 17:14:25.1320618+08:00`
2. SQLite 的 `datetime` 列在比较时按 **文本字典序** 比较；数据库里的时间是 `+08:00`，而查询参数来自前端 `toISOString()`，是 UTC 文本，例如：
   `2026-11-24 15:59:59.999+00:00`
3. 字典序比较忽略了时区偏移，导致“今天 16:00 之后”的本地时间被判断为大于“今天 23:59 的 UTC 表示”，从而被排除。

实际复现（真实 API 请求，UTC+8）：

```text
GET /api/applications?page=1&pageSize=10
→ total = 1

GET /api/applications?page=1&pageSize=10
    &from=2026-11-23T16:00:00.000Z
    &to=2026-11-24T15:59:59.999Z      ← 前端“今天”的本地起止时间
→ total = 0
```

在浏览器中实际选择“今天 → 今天”后，表格行数同样变为 `0`。

**影响范围**

- 投递列表日期筛选（`GET /api/applications`）
- 统计 KPI（`GET /api/stats`）
- 趋势与日历热力图（`GET /api/stats/daily`）
- 转化漏斗（`GET /api/stats/funnel`）
- 跨时区迁移/导入后，`applied_at` 文本偏移不一致，还可能导致 `ORDER BY applied_at` 排序错乱

对于 UTC+8，每天 **16:00–24:00** 创建的记录都会被当天筛选漏掉；对于 UTC-5，每天 **00:00–05:00** 的记录会被漏掉；其他时区与夏令时同理。

**建议修复**

根本修复是统一存储为 UTC，而不是只修改查询参数：

1. `database.NowFunc` 改为 `time.Now().UTC()`；
2. `createApplication` 中 `AppliedAt = time.Now().UTC()`；
3. 所有接收 `from`/`to` 的接口在绑定参数时统一转换为 UTC；
4. 对已有数据做一次迁移，把历史时间文本规范化为 UTC（或在查询中统一使用 SQLite `datetime(...)` 归一化，但这会失去索引优势）；
5. 为 `TestApplicationListDateRangeFilter` 增加“本地自然日”用例（现有测试使用 UTC 日期边界，掩盖了该问题）。

---

### 2. 【中】仅日期作为结束日期时，结束当天整天被排除

**位置**

- 时间解析：`internal/server/application_history.go:269-286`
- 列表筛选：`internal/server/applications.go:99-110`
- 统计筛选：`internal/server/statistics.go:18-45`

**现象**

使用纯日期参数查询同一天：

```text
GET /api/applications?from=2026-11-24&to=2026-11-24
→ total = 0（即使当天确实有记录）

GET /api/stats?from=2026-11-24&to=2026-11-24
→ total = 0
```

**根因**

`parseFlexibleTime` 对 `2006-01-02` 格式按 **当天 00:00:00** 解析；作为 `to` 上界时使用 `<= to`，因此只能包含 00:00:00 这一刻，当天其余时间全部被排除。

前端目前发送的是 `endOf("day").toISOString()`，所以 UI 主要受第 1 项影响；但 README 公开了这些 API，手工调用或其他客户端会直接踩到该问题。

**建议修复**

解析日期时保留“是否仅日期”的信息；当它作为 `to` 使用时，改为当天 `23:59:59.999...` 或使用 `< 次日 00:00` 的排他上界。

---

### 3. 【中】繁中、日、韩翻译缺失，界面出现简体中文/英文混排

**位置**

- `frontend/src/i18n/index.tsx:931`（`zhTWMessages` 直接展开 `zhCNMessages`）
- `frontend/src/i18n/index.tsx:1235`（`jaMessages`）
- `frontend/src/i18n/index.tsx:1551`（`koMessages`）
- 回退逻辑：`frontend/src/i18n/index.tsx:1978-1987`

**现象**

- 繁中（zh-TW）界面出现简体中文，例如设置页同时出现“設定”和“设置”、“数据导入与导出”、“导出数据”。
- 日文（ja）界面出现英文，例如设置页的 “Auto-lock when idle”“Data import & export”“Export data”。
- 韩文（ko）同样缺少 138 个键，回退为英文。

经统计（相对完整的 `en` 字典，共 441 个键）：

| 语言 | 已翻译键数 | 缺失键数 | 回退结果 |
| --- | --- | --- | --- |
| zh-CN | 441 | 0 | — |
| zh-TW | 299 | 142 | 简体中文（因为 `...zhCNMessages`） |
| ja | 303 | 138 | 英文 |
| ko | 303 | 138 | 英文 |

缺失集中在较新的功能：`history.*`、`trash.*`、`close.*`、`settings.desktop/idleLock/dataTransfer`、`transfer.*`、`stats.*` 新增项。

**建议修复**

1. 补齐 zh-TW / ja / ko 的缺失键；
2. 增加构建期或测试期的“多语言键集合一致性”检查；
3. zh-TW 不应通过展开 zh-CN 来兜底，否则会静默产生简繁混排。

---

### 4. 【中】快速录入窗口已打开时，切换分类不生效

**位置**

- `main.go:391-397`（`openQuickEntryWindow`）
- `frontend/src/pages/QuickEntryWindowPage.tsx:22-35`（只在挂载时读取一次 `category`）

**现象**

快速录入窗口（置顶常驻）已经打开时，从“春招”页面再次点击快速录入，窗口被 `Show/Focus`，但分类仍然保持上一次的值。

**根因**

其他独立窗口都处理了 URL 更新：

```go
existing.SetURL(target)
```

但快速录入窗口只做了：

```go
if existing, ok := app.Window.Get("quick-entry"); ok {
    existing.Show()
    existing.Focus()
    return
}
```

页面组件也只在首次挂载时读取 `?category=`，之后不会再响应。

**建议修复**

与 `openApplicationEditWindow` 等保持一致，为已存在窗口调用 `SetURL("/quick-entry?category="+url.QueryEscape(category))`，或通过事件通知页面更新分类。

---

### 5. 【中】多窗口下闲置自动锁屏会被误触发

**位置**

- `frontend/src/App.tsx:151-198`

**现象**

开启“闲置 N 分钟自动锁屏”后，用户在主窗口无操作、但在简历预览/快速录入/编辑等独立窗口中持续操作超过 N 分钟，主窗口仍会执行 `logout`，导致所有窗口的会话失效（预览可能报 401）。

**根因**

- 每个窗口都是独立的前端上下文，`App.tsx` 的 `IdleLock` effect 在每个窗口都会运行；
- 计时器只监听 **当前窗口** 的鼠标/键盘事件；
- 主窗口不知道其他窗口有活动，也不会被其他窗口重置。

**建议修复**

- 通过 Wails 事件或后端心跳共享“最后活动时间”，任一窗口活动都刷新所有窗口计时器；或
- 只让主窗口负责空闲计时，其他窗口忽略；或
- 锁屏时只锁当前窗口而不是全局会话。

---

### 6. 【低】Headless / 浏览器模式下，编辑和详情窗口的关闭按钮无效

**位置**

- `frontend/src/utils/desktop.ts:129-139`（非桌面环境调用 `window.close()`）
- `frontend/src/pages/ApplicationEditWindowPage.tsx:106`
- `frontend/src/pages/ApplicationViewWindowPage.tsx:199-207`

**现象**

使用 `--headless` 或普通浏览器访问时，Dashboard 通过 `window.location.assign("/application-edit?id=...")` 在同一标签页打开编辑页。保存成功后调用 `closeWindow()`，但浏览器会忽略脚本发起的 `window.close()`，页面停留在编辑页；详情页的关闭按钮同样无反应。

**对比**

`ResumePreviewWindowPage` 已经实现了正确的降级逻辑：

```ts
if (isDesktopRuntime()) { ... }
if (window.opener) { window.close(); return; }
if (document.referrer.startsWith(location.origin)) { history.back(); return; }
location.assign("/resumes");
```

编辑/详情窗口没有复用这一逻辑。

**建议修复**

提取统一的 `closeWindowOrGoBack()`，在非桌面运行时执行 `history.back()` 或跳回列表页。

---

### 7. 【低】切换数据库后旧会话仍然有效

**位置**

- `internal/server/database_settings.go:94-99`
- `internal/server/auth.go:57-61`、`:238-252`

**现象**

在设置页热切换 SQLite 数据库后：

- `s.auth` 的会话表仍在内存中，`auth.setDB(newDB)` 只换数据库，没有清空会话；
- 如果新数据库启用了密码，`/api/auth/status` 会因为旧 token 仍有效而返回 `authenticated: true`，不会要求输入新数据库的密码。

虽然这是本机用户主动选择数据库，但属于会话状态与数据源不一致，建议按安全边界处理。

**建议修复**

切换数据库成功后调用 `s.auth.invalidateAllSessions()`，前端清除 token 并重新走锁定/解锁流程。

---

### 8. 【低】批量状态/分类接口没有数量上限，且逐条更新

**位置**

- `internal/server/applications.go:375-420`（批量状态）
- `internal/server/applications.go:422-447`（批量分类）
- 对照：`batchDeleteApplications` 在 `:348-351` 有 2000 条上限

**问题**

- 批量删除有 `len(ids) > 2000` 校验，批量状态/分类没有；
- 批量状态在事务中逐条 `Update`，超大数组会生成极长的 `IN` 子句并执行数千次 SQL。

本地应用风险有限，但容易造成界面卡顿或数据库长时间占用。

**建议修复**

复用相同的 `too_many_ids` 校验；批量状态可改为一次 `UPDATE ... WHERE id IN ?`，时间线记录再按需插入。

---

### 9. 【低】Excel 导入在检查行数上限之前就把整张表读入内存

**位置**

- `internal/server/excel.go:358-377`

**问题**

`file.GetRows(sheets[0])` 会一次性读取整张工作表；行数上限 `maxExcelRows = 5000` 是在读取之后才检查的。一个 10 MiB 的 xlsx 压缩包可以解压出极大的工作表或共享字符串，可能在触发上限前就耗尽内存。

**建议修复**

- 使用 `excelize.Rows()` 流式读取，读一行计一行，超过上限立即中止；
- 或在 `excelize.OpenReader` 时配置解压大小限制，并先做行列数量探测。

---

### 10. 【低】删除简历失败时可能残留 `.deleting` 文件

**位置**

- `internal/server/resumes.go:292-314`

**问题**

删除流程先把文件重命名为 `xxx.reb.deleting`，再删除数据库记录，最后 `os.Remove(trashPath)`。如果最后一步失败，数据库记录已经删除，残留的 `.deleting` 文件不会再被任何逻辑清理，长期积累占用磁盘。

**建议修复**

启动时清理 `resume/*.deleting`，或删除失败时记录日志并提供清理入口。

---

### 11. 【低】README 仍引用已删除的密码恢复脚本

**位置**

- `README.md:45`（说明根目录提供 `recover-password.cmd` / `recover-password.sh`）
- 当前工作区 `git status`：

```text
 D recover-password.cmd
 D recover-password.sh
```

`cmd/recover-password` 的 Go 程序仍然存在，但两个包装脚本在当前工作树中已被删除。若删除是有意的，应同步修改 README；若非有意，应恢复这两个文件。

---

### 12. 【低】`Server.Start` 注释与实际行为不一致

**位置**

- `internal/server/server.go:338-340` 注释：端口不可用时“选择临时端口”
- `internal/server/server.go:357-363` 实现：端口占用时直接返回错误
- README 的描述是正确的（拒绝启动第二个实例）

**建议**

更新注释，避免后续维护者误以为存在自动换端口逻辑。

---

## 三、需要环境确认 / 低置信度观察项

### A. 无边框可调整窗口缺少缩放手柄

- `main.go` 中编辑、详情、数据迁移窗口都设置了 `DisableResize: false`（`:351`、`:382`、`:676`）；
- 但只有主窗口（`App.tsx`）和简历预览（`ResumePreviewWindowPage.tsx`）渲染了 `ResizeHandles`；
- 如果当前 Wails 版本的无边框窗口依赖这些自定义热区才能调整大小，那么编辑/详情/数据迁移窗口将无法缩放。

建议在 Windows 桌面上手动拖拽窗口边缘确认；若确实无法缩放，为这三个页面补上 `<ResizeHandles />`。

### B. 本地 API Token 对“本机进程”的防护有限

`README.md:355` 声称 Token 用于阻止“其他网页或本机进程”访问本地 API。但 Token 就注入在 `GET /` 返回的 `index.html` 中，任何本机进程都可以请求首页并解析出 Token。它真正能防住的是**跨源网页**（CORS + 不可读响应），对同机进程只能算一层很弱的门槛。建议修正文档中的威胁模型描述，或结合操作系统级权限做进一步限制。

### C. `createApplicationHistory` 允许补录未来日期/与当前状态不一致的阶段

接口本身只校验状态枚举和日期格式，不校验日期是否晚于 `applied_at`、也不要求补录阶段与当前状态一致。正常 UI 不会触发，但 API 调用或手工备份可以写入异常时间线。建议增加业务校验（例如补录日期不得晚于当前状态变更时间）。

### D. 备份导入的回收站记录未做状态/分类校验

`internal/server/backup.go:801-805` 对 `manifest.Trash` 直接重置 ID 后插入，没有像 `manifest.Applications` 那样校验 `category`/`status`。手工编辑过的备份可能在回收站中留下非法分类/状态，恢复后形成脏数据。建议复用应用记录的校验逻辑。

### E. 数据库热切换后 1 秒关闭旧连接池

`internal/server/database_settings.go:108-117` 在切换 1 秒后关闭旧连接池。如果此时仍有长请求（例如大批量导入、统计查询）在使用旧 `*gorm.DB`，会收到 `sql: database is closed`。建议对正在执行的请求做引用计数，或延迟到所有请求结束后再关闭。

### F. 代码格式/换行符

`gofmt -l` 会列出：

```text
internal/security/password.go
internal/desktop/shortcut_windows.go
```

这两个文件使用 CRLF，其余 Go 文件为 LF，导致 `gofmt -l` 误报整文件差异。建议统一为 LF，或在 `.gitattributes` 中明确 `*.go text eol=lf`。

---

## 四、已执行的检查

| 检查 | 结果 |
| --- | --- |
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test ./...` | 全部通过（48+ 个测试） |
| `cd frontend && npm run typecheck` | 通过 |
| Headless 实际启动 + 浏览器访问 | Dashboard / Statistics / Settings / Trash / Resumes / Quick Entry / Application View 均无控制台报错 |
| 日期范围实测 | 确认第 1、2 项问题 |
| 多语言实测 | 确认第 3 项问题 |
| `go test -race ./...` | 未能执行：当前环境 `CGO_ENABLED=0`，`-race` 需要 cgo |

> 现有测试 `TestApplicationListDateRangeFilter` 使用 UTC 日期边界，因此没有覆盖本地自然日边界，建议补充回归测试。

---

## 五、测试数据清理

审查过程中产生的数据已全部清理：

- 临时探针测试文件 `internal/server/zz_tmp_probe*_test.go` 已删除；
- 临时 headless 服务已停止；
- 临时数据目录 `C:\Users\dshiq04\AppData\Local\Temp\rh-audit`（含 `resume-helper.db`、`config.yaml`、`logs/`、浏览器测试应用）已删除；
- 审查阶段未留下新增或修改的业务文件；当时 `git status` 仅保留审查前已存在的用户改动：
  - `frontend/src/i18n/index.tsx`（已有修改）
  - `recover-password.cmd` / `recover-password.sh`（已有删除）

---

## 六、修复记录（2026-11-24）

本轮已按本报告修复全部 12 项确认问题，并处理部分观察项。

| 编号 | 状态 | 修复内容 |
| --- | --- | --- |
| 1 | 已修复 | 写入时间统一为 UTC（`NowFunc`、`AppliedAt`、状态变更时间等）；所有日期范围查询和排序使用 SQLite `datetime(...)` 归一化，兼容旧的带时区文本 |
| 2 | 已修复 | 纯日期作为 `to` 时使用 `AddDate(0,0,1) - 1ns`，包含完整自然日，并兼容 DST |
| 3 | 已修复 | 补齐 zh-TW 142 个、ja 138 个、ko 138 个缺失翻译；移除 zh-TW 对 zh-CN 的兜底展开，避免再次混入简体 |
| 4 | 已修复 | 已存在的快速录入窗口现在会 `SetURL` 更新分类；新增 `quickEntryTarget` 单测 |
| 5 | 已修复 | 闲置锁屏通过 Wails 事件 + localStorage `storage` 事件跨窗口共享活动；锁定时通知其他窗口重新检查认证 |
| 6 | 已修复 | 浏览器/Headless 模式下 `closeWindow` 改为先 `history.back()`，否则回到首页 |
| 7 | 已修复 | 切换数据库文件时清空内存会话，旧 token 失效，需要重新解锁 |
| 8 | 已修复 | 批量状态/分类接口增加 2000 条上限，返回 `too_many_ids` |
| 9 | 已修复 | Excel 导入改为 `Rows()` 流式读取，超过 5000 行立即中止，不再先读完整表 |
| 10 | 已修复 | 启动时清理 `resume/*.deleting` 残留文件 |
| 11 | 已修复 | 恢复 `recover-password.cmd` / `recover-password.sh`，与 README 一致 |
| 12 | 已修复 | 更新 `Server.Start` 注释，明确端口占用时拒绝启动 |
| 观察 A | 已修复 | 编辑、详情、数据迁移窗口补充 `<ResizeHandles />` |
| 观察 B | 已修复 | README 中本地 Token 的威胁模型描述已修正 |
| 观察 D | 已修复 | 备份导入会校验回收站记录的分类、状态和公司名 |
| 观察 F | 已修复 | `.gitattributes` 增加 `*.go text eol=lf`，并格式化相关 Go 文件，`gofmt -l` 已无输出 |

新增回归测试：

- `internal/server/audit_fixes_test.go`
  - 本地自然日边界、混合时区偏移、日期单日筛选、统计与日历；
  - 批量接口 2000 条上限；
  - 数据库切换后旧会话失效；
  - `.deleting` 残留清理；
  - Excel 流式读取行数上限；
  - 备份导入拒绝非法回收站记录。
- `main_test.go`
  - 快速录入窗口分类 URL。

本轮验证结果：

| 检查 | 结果 |
| --- | --- |
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test ./... -count=1` | 通过（含新增回归测试） |
| `gofmt -l .` | 无输出 |
| `cd frontend && npm run typecheck` | 通过 |
| `cd frontend && npm run build` | 通过 |
| Headless 实测日期筛选 | 修复前“今天→今天”为 0 条；修复后同一天记录正常显示 |
| Headless 实测编辑窗口关闭 | 修复前停留在编辑页；修复后返回仪表盘 |
| Headless 实测 ja / zh-TW 设置页 | 不再出现英文/简体混排 |

修复阶段的临时服务、临时数据库和测试记录同样已删除；仓库中仅保留代码修复、回归测试、`BUG_REPORT.md` 及修复记录。

