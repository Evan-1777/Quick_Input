# Plan：MyGo 原生前端重构与紧凑 MD3 风格

**状态**：DONE（交付段完成，2026-11-05）  
**日期**：2026-11-05  
**版本**：v1.1（Review 修订：任务删除边界、生命周期、构建链、键盘模型、退出标志、主题接线）  
**修订依据**：外部审阅报告 15 条，逐条对照 `github.com/egoist/mygo v0.7.4` 源码核实后采纳 13 条、部分采纳 1 条、驳回 1 条（见 §8）  
**回归结论**：`go test ./...`、`go vet ./...`、`go tool mygo vet`、`GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...` 全部通过；面板/管理列表/侧边页/设置页/确认框/空态/深色主题经 `ui.Render` 与 `ui.Tester` 截图核对；本地 GTK 启动通过，Wayland 开发会话的热键与托盘能力受环境限制，留待 Windows 11 实机验收

---

## 1. 背景与目标

现状是 Wails v2 + React 18 + Tailwind 的 WebView 前端（`frontend/dist` 由 `go:embed` 嵌入），窗口、热键、托盘、键入均以 Win32 手写实现（`internal/platform/platform_windows.go`，874 行）。本次交付把整个前端换成 MyGo 原生 UI（`github.com/egoist/mygo/ui`，GPU 直绘，无 HTML/JS/WebView），并把视觉语言从 Fluent 2 改为紧凑 Material 3：tonal surface 色阶表达层级、8dp 网格、pill 按钮、12px 容器圆角，去除 Wails/React/Tailwind/TypeScript 全部依赖与产物。

保留的行为契约：

- 全局唤出快捷键（可配置、失败回退、设置页切换并持久化；设置页展示持久化值，与既有 `Project.md` 契约一致）
- 悬浮面板检索：关键字 + 单选分组 + 多选标签交集，上下键循环移动、Enter 键入、Esc 逐级退出、失焦收起
- 键入：唤出时快照目标窗口 → 隐藏面板 → 恢复目标焦点 → 等待修饰键释放 → 剪贴板换入 → Ctrl+V → 序列号未变时还原剪贴板
- 管理窗口：分组/标签增删改、预设文本增删改、批量移组与批量标签、二次确认、错误就地展示
- 托盘常驻、单实例、关闭即隐藏、托盘退出真正退出

## 2. 阶段划分

### Phase 1：基座迁移

| 项目 | 内容 |
|------|------|
| **输入** | 现有 store、platform 键入实现、Wails 接线 |
| **输出** | MyGo 应用骨架：双窗口（panel + manager）、托盘、全局快捷键、单实例与退出判定；MD3 主题令牌与内联 SVG 图标集；`platform.Paster` 新增（与旧 Bridge 并存）；最终切换时删除旧实现与 `frontend/` |
| **验收标准** | `CGO_ENABLED=0 go build ./...`、`GOOS=windows go build ./...`、`go vet ./...`、`go tool mygo vet ./...` 通过；应用在本地 GTK 环境可启动；无 Wails/React 残留引用 |

### Phase 2：悬浮检索面板

| 项目 | 内容 |
|------|------|
| **输入** | Phase 1 的 App 状态与主题；`filter.ts` 的过滤/导航语义 |
| **输出** | 560×380 无边框面板：MD3 搜索栏、结果列表（标题/内容预览/分组/标签 chip）、分组与标签筛选栏、空态、唯一按键路径与插入链路 |
| **验收标准** | 输入即过滤；↑↓ 循环移动并滚动可见；Enter 调用插入；标签交集与分组筛选正确；Esc 三级语义正确；空态提供下一步动作 |

### Phase 3：管理窗口

| 项目 | 内容 |
|------|------|
| **输入** | Phase 1-2 的主题、图标与 App 数据操作 |
| **输出** | 960×640 原生标题栏窗口：侧栏（分组/标签 CRUD）、工具栏（搜索/新建）、列表（多选与批量栏）、编辑侧边页、删除确认、设置页（快捷键单选） |
| **验收标准** | 全部 CRUD 生效并即时刷新；批量移组/加标签/解标签/删除可用；删分组与标签需确认且文案准确；快捷键切换失败保留原组合并就地报错；点击当前组合不报错 |

### Phase 4：验证、文档与归档

| 项目 | 内容 |
|------|------|
| **输入** | Phase 1-3 全部改动 |
| **输出** | `ui.NewTester` 视图测试、过滤纯函数测试、store 回归、交叉编译与 vet、界面渲染核对、`Project.md` 同步、CI 改为 MyGo 构建、归档与提交 |
| **验收标准** | `go test ./...`、`go vet ./...`、`go tool mygo vet ./...`、`gofmt -l`、交叉编译全部通过；根目录无 Plan/Tasks 残留；工作区干净 |

## 3. 架构决策

| 决策项 | 选择 | 理由 | 替代方案（为何不选） |
|--------|------|------|----------------------|
| 窗口模型 | 双窗口：`Panel: true` 的悬浮面板 + 常规管理窗口 | MyGo 面板原生支持「悬浮于所有应用、不进任务栏、可与全局快捷键配合」；管理窗口用系统标题栏，省掉自绘窗口按钮 | 单窗口切模式（旧方案）：原生 UI 下切换 Frameless/尺寸会重排闪烁，且面板与常规窗口语义不同 |
| 生命周期 | `RequestSingleInstanceLock → SetName → SetActivationPolicy → WhenReady{建窗口/托盘/快捷键} → App.Run()` | `NewWindow`、`GlobalShortcut.Register`、`NewTray` 在应用未 ready 时 panic 或阻塞（`app.go`、`modules.go:686`） | 在 main 顶层直接建窗口（启动即 panic） |
| 退出 | `quitting` 标志：托盘退出置位后 `App.Quit()`；`Window.OnClose` 仅在未退出时 `PreventDefault + Hide` | `Application.Quit` 可被任一窗口的 `OnClose` 取消，否则托盘「退出」退化为隐藏窗口 | 关闭全部窗口再退出（窗口被拦截，同样卡住） |
| 依赖方向 | `main → app → store / platform`，视图与状态同在 `app` 包；`App.Attach(panel, manager)` 注入窗口 | 原生 UI 无 IPC，视图直接读应用状态；窗口注入让视图可驱动隐藏/定位 | 保留绑定层（MyGo typed bindings 只服务于 Web 前端，此处无收益） |
| 数据层 | `internal/store` 原样保留（SQLite/modernc） | 已有测试与事务语义，无迁移成本；视图只用全量读取 + 内存过滤，store 的过滤参数保持既有契约 | MyGo sqlite 插件（重写全部查询，无收益） |
| 键入实现 | 保留 Win32 剪贴板 + SendInput，抽为 `platform.Paster`（Windows 实现 + 非 Windows 桩） | MyGo 不提供向他人窗口注入按键；这是产品核心能力 | 仅写剪贴板让用户自己 Ctrl+V（丢核心行为） |
| 键入顺序 | 唤出：`CaptureTarget → 定位 → Show → Focus`；插入：主线程 `Hide` 完成后再 `Paste` | 早于 Show 快照才能取到用户前台窗口；Hide 未落地时 `SetForegroundWindow` 会被面板抢回 | 在 Paste 内快照（此时前台已是本进程） |
| 快捷键模型 | 直接使用 MyGo accelerator 字符串，持久化写预设原文，不做规范化 | 启动注册、失败回退、运行期切换由 `GlobalShortcut` 提供；格式不漂移 | 自研解析（重复实现且与 MyGo 语法漂移） |
| 快捷键展示 | 设置页展示持久化值（`a.saved`），实际注册值仅用于注册与回滚 | 与 `Project.md` 既有契约一致，避免双状态 | 展示实际生效值（需第二状态通道，且与既有契约冲突） |
| 面板定位 | 光标所在显示器：水平居中、纵向 1/3 | MyGo 无「取输入光标矩形」的跨平台 API；Spotlight/Raycast 同款位置 | Win32 `GetGUIThreadInfo` 跟随插入符（仅 Windows、混合 DPI 下仍需换算，收益低于复杂度） |
| 悬浮面板键盘 | 搜索框保持焦点；↑↓ 由输入控件 `OnTextKey` 接管，Esc 由根容器统一处理，Enter 取 `Submitted()`（焦点不在文本输入时才退回根级快捷键）；列表不绑定 `Selected`、不参与键盘 | 文本输入会消费 ↑↓，根级快捷键收不到；在输入控件内接管可保持单一按键路径、循环语义与无重复触发 | 依赖 List 内建键盘（循环/Enter 语义不符，且与根级快捷键二选一） |
| 视觉层级 | MD3 tonal surface：`surface` / `surface-container-low` / `-high` 表达层级，仅侧边页与确认弹窗叠 scrim | 紧凑 MD3 的核心表达是色阶而非投影 | Fluent 的 Mica/Acrylic（与本次风格目标相反） |
| 圆角 | 容器 12、输入 8、按钮 pill、侧边页 16（四档 token 集中在 `theme.go`） | MD3 shape scale；紧凑版整体收一档 | 统一 4-8px（旧 Fluent 语言） |
| 主题接线 | 每个视图根调用 `useMD3(c)`：每帧以 `c.Theme().Dark` 重建 MD3 主题并 `SetTheme` | 上下文主题每帧重置为系统默认，不重建则自定义主题不生效或不再跟随外观 | 启动时 SetTheme 一次（外观切换失效） |
| 图标 | 全部内联 SVG（`ui.MustParseSVG`），仅图标按钮配 `Label`/tooltip | 用户要求「能 SVG 就不写字」；MyGo 的 Icon 继承文字色 | 文字按钮（信息密度低、破坏节奏） |
| 批量选择 | 使用 `ui.ListState.Selection`（Ctrl/Cmd 点击、Shift 范围、Ctrl+A） | 复用控件原生语义，去掉自绘 checkbox 与全选状态 | 自绘勾选框（重复实现键盘与范围选择） |
| 键盘提示条 | 删除 | 资深用户默认，无需引导；MD3 也不以提示条承载操作说明 | 保留 ↑↓/Enter 提示（噪声） |
| 构建链 | `go tool mygo build -platform windows/amd64`（`go.mod` 声明 `tool` 指令） | 图标、manifest、版本资源由 `mygo build` 写入 `.syso`（`cmd/mygo/build.go:519`），裸 `go build` 会丢失并影响 DPI 行为 | 直接 `go build -H windowsgui`（无图标/清单/版本信息） |
| 新增依赖 | `github.com/egoist/mygo`（含 `ui`、CLI tool），删除 `wails` | 框架替换本身 | 无 |

## 4. 风险清单

| 风险 | 等级 | 缓解措施 |
|------|------|----------|
| MyGo 0.7.4 要求 Go ≥1.27.3，本机为 1.27.0 | 🟡 中 | `GOTOOLCHAIN=auto` + 代理已实测可构建；CI 固定 `go-version: 1.27.x` |
| `ui.Element` 不可跨帧保存 | 🔴 高 | 只存 `ui.Handle`/`ListState`/普通字段；`go tool mygo vet` 兜底 |
| 面板焦点与目标窗口：Windows 上全局快捷键让面板取得前台 | 🟡 中 | 唤出时先 `CaptureTarget`（排除自身进程），插入前 `SetForegroundWindow` 并等待 100ms |
| `GlobalShortcut.Register` 重复注册同一组合报错 | 🟡 中 | `SetHotkey` 对「当前已注册组合」直接返回 nil；注册失败恢复旧组合 |
| 剪贴板句柄绑定线程 | 🟡 中 | 读/写各自 `runtime.LockOSThread()`；保留 150ms 后按序列号还原 |
| 跨显示器混合 DPI：`Screen.CursorScreenPoint` 按光标所在显示器 DPI 转 DIP，窗口定位按自身 DPI 换算 | 🟢 低 | 面板仅按显示器工作区居中，不依赖跨屏坐标换算；用户 Windows 实机验收 100%/125%/150% 三档，偏差记录为已知限制 |
| 本机无法运行 Windows 产物 | 🟡 中 | Linux 上用 `ui.Render`/真实窗口截图核对四态；Windows 交叉编译 + `go vet`；实机行为由用户验收 |
| 视图测试定位元素依赖可访问名称 | 🟢 低 | 图标按钮统一 `.Label(...)`；测试用 `tt.Click("新建")` 等稳定名称 |
| headless 测试中窗口指针为 nil | 🟢 低 | 所有窗口调用统一 nil 保护；测试注入零值窗口或不注入 |
| 原生控件外观与 MD3 规范存在差异（字体、按钮形态） | 🟢 低 | 色板/圆角/间距/字号集中在 `theme.go`，按钮与输入形状由助手统一套用 |

## 5. Phase 依赖关系

```
Phase 1 ──→ Phase 2 ──→ Phase 3 ──→ Phase 4
```

Phase 2 与 Phase 3 共用 Phase 1 的主题与数据操作；Phase 3 的编辑侧边页依赖 Phase 2 已确立的列表行样式与 chip 组件。

## 6. Subagent 协同规划

| 阶段 | 上下文依赖 | 并行收益 | 执行主体 | 决策理由 |
|------|-----------|---------|----------|----------|
| Phase 1：基座迁移 | 强（store 契约、键入实现、窗口生命周期需连续推演） | 无（后续全部依赖） | 主线 | 迁移期全仓库耦合，分支并行只会制造冲突 |
| Phase 2：悬浮面板 | 强（依赖 Phase 1 的主题、图标、App 状态） | 低（单文件视图，与 Phase 3 共享样式） | 主线 | 与 Phase 3 共享组件，拆开会产生重复实现 |
| Phase 3：管理窗口 | 强（依赖 Phase 1-2） | 低 | 主线 | 同上 |
| Phase 4：验证与文档 | 强（依赖全部改动与实时测试结果） | 无（串行收口） | 主线 | 需按失败结果即时修复 |

> **协同结论**：本次是跨全栈的单体替换，阶段间强串行且共享同一套主题/图标/组件，委托会造成重复加载与契约漂移，全部由主线执行，不派发 Subagent。

## 7. 实施要点（探索期已核实）

| 主题 | 结论 |
|------|------|
| 工具链 | `github.com/egoist/mygo v0.7.4` 已在模块缓存；`go.mod` 加 `tool github.com/egoist/mygo/cmd/mygo` 后 `go tool mygo version` 可用（已实测）；本机 GTK3、Noto Sans SC 可用 |
| 无窗口走查 | `ui.Render(view, w, h, scale)` 返回 `*image.RGBA`，可在 `go test` 中把视图渲染为 PNG 做设计核对，无需启动窗口 |
| 布局默认值 | `ui.Row` 不纵向拉伸子元素；主内容行须 `.AlignItems(ui.Stretch)` 才能撑满窗口高度 |
| 颜色 | 混色是方法 `ui.Color.Mix(other, t)`，包级不存在 `ui.Mix` |
| 控件 | `ui.SearchField` 默认占位文案为英文 "Search"，必须显式 `Placeholder`；非空时自行消费 Esc 清空文本（`ui/combobox.go`）；`Element.Disabled(v)` 只有 setter，读取用 `IsDisabled()` |
| 列表键盘 | `ui.List` 的 ↑↓ 不循环（`ui/list.go` 的 `step`）、聚焦时自行把 Enter 记为 Submitted；面板与设置页不依赖其键盘行为 |
| 文本域 | `TextArea.Lines(min, max int)` 为区间签名；`TextInput` 无长度上限 API，长度限制需自行在 `Changed` 中截断或不设 |
| 退出 | `Application.Quit` 可被任一窗口 `OnClose` 取消（`app.go` 注释明示），必须先置退出标志 |
| 构建 | `mygo build` 生成 `mygo_windows_<arch>.syso`（图标、manifest、版本）；官方 CI 用 `go tool mygo build -platform …` |
| 窗口 | 面板用 `Panel: true` + `Frameless` + `Transparent` + `CornerRadius`，`OnClose` 转 `Hide`、`OnBlur` 隐藏；管理窗口保留系统标题栏 |
| 托盘 | 图标以代码绘制 32×32 PNG，与界面标识同源，不引入位图缩放链路 |
| 设计验证 | 面板、管理列表、编辑侧边页、设置页四态的视图草案已用 `ui.Render` 走查过排版与色阶 |

## 8. 审阅意见处置

| # | 意见 | 处置 |
|---|------|------|
| 1 | TASK-002 删除边界导致中间态不可构建 | 采纳：TASK-002 改为纯增量，旧 Bridge 与旧测试保留，删除并入 TASK-004 |
| 2 | 窗口/托盘/快捷键未落到 `WhenReady` | 采纳：TASK-004 写死生命周期顺序，TASK-003 增加 `Attach` 注入 |
| 3 | CI 用 `go build` 丢失图标/清单/版本/DPI | 采纳：`go.mod` 声明 tool 指令，CI 改 `go tool mygo build -platform windows/amd64` |
| 4 | 面板键盘与控件内建行为冲突 | 采纳：搜索框保持焦点，根容器唯一按键路径，列表不参与键盘；Esc 改为三级 |
| 5 | 托盘退出被 `OnClose` 拦截 | 采纳：`quitting` 标志 |
| 6 | 主题未接线、缺少派生色与形状落地 | 采纳：`useMD3` 每帧接线、补充 hover/pressed 与 error-container、形状助手 |
| 7 | 搜索框占位文案缺失 | 采纳：两处显式 `Placeholder`，测试断言无英文占位 |
| 8 | `maxLength` 无 API、`Lines(n)` 签名不符 | 采纳：去掉 120 字上限，改用 `Lines(6, 12)` |
| 9 | 设置页语义与既有契约冲突、重复注册报错 | 采纳：展示持久化值；`spec == 当前注册值` 短路 |
| 10 | 唤出/插入顺序未定义 | 采纳：写死顺序约束 |
| 11 | 混合 DPI 未纳入验收，建议加 Win32 定位钩子 | 部分采纳：纳入用户实机验收并记录已知限制；驳回 Win32 定位钩子（个人单机工具，收益不抵复杂度） |
| 12 | 过滤语义漂移与双实现 | 采纳语义澄清（trim、`title+"\n"+content`）与测试用例；驳回删除 store 过滤参数（既有数据层契约，删除扩大 blast radius） |
| 13 | 快捷键持久化格式 | 采纳：写预设原文，不做规范化 |
| 14 | `siconFolder` 拼写与 `ActivationPolicyAccessory` 常量 | 采纳：修正为 `iconFolder` 与 `mygo.ActivationPolicyAccessory` |
| 15 | headless 测试 nil 窗口 | 采纳：窗口调用统一 nil 保护 |

## 9. 交付差异

| 差异 | 原因 | 影响 |
|------|------|------|
| 面板搜索栏由 `ui.SearchField` 改为自绘 `ui.TextInput` 行 | `SearchField` 内部的文本框会消费 ↑↓，根级快捷键收不到；自绘行在输入控件内用 `OnTextKey` 接管上下键 | 占位文案、清除按钮与 MD3 外观保持一致，键盘仍只有一条路径 |
| 设置页每行用 `ui.Button` 承载键帽 chip，`Label` 为组合原文 | 让测试与辅助技术能按组合定位整行，同时保留键帽视觉 | 不改变设置页排版与选择逻辑 |
| 管理列表归属 chip 最多 3 枚 | 避免长标签挤占标题与操作区 | 与 Task 007 的约束一致 |