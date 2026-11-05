# Tasks：MyGo 原生前端重构与紧凑 MD3 风格

**关联 Plan**：`Plan.md` —— MyGo 原生前端重构与紧凑 MD3 风格 v1.1  
**状态**：DONE（2026-11-05 完成全部 14 个 Task）  
**回归结论**：`go test ./...`、`go vet ./...`、`go tool mygo vet`、`gofmt -l .`（无输出）与 Windows 交叉编译全部通过；本地 GTK 启动通过；Wayland 开发会话的热键/托盘限制已记录，Windows 行为由 CI 产物实机验收  
**总计 Task**：14 个

---

## Phase 1：基座迁移

### TASK-001：建立 MD3 主题、形状 token 与 SVG 图标基座

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P0
- **Depends On**：无
- **Description**：新增 `internal/app/theme.go` 与 `internal/app/icons.go`，以 MD3 紧凑色板覆盖 `ui.Theme`，提供每帧主题接线、形状 token、全量内联 SVG 图标与样板助手。
- **Details**：
  - `theme.go`：`type mdPalette struct` 持有容器色阶；`func mdTheme(dark bool) *ui.Theme` 从 `ui.LightTheme()` / `ui.DarkTheme()` 取副本后覆写：
    - 色板（light）：`surface #F9F9FF`、`surface-container-low #F3F3FA`、`surface-container #EDEDF4`、`surface-container-high #E7E8EE`、`surface-container-highest #E1E2E8`、`on-surface #1A1C1E`、`on-surface-variant #44474E`、`outline #74777F`、`outline-variant #C4C6D0`、`primary #0B57D0`、`on-primary #FFFFFF`、`primary-container #D3E3FD`、`on-primary-container #041E49`、`error #BA1A1A`、`error-container #FFDAD6`、`on-error-container #410002`、`inverse-surface #2F3033`、`inverse-on-surface #F1F0F4`
    - 色板（dark）：`surface #111318`、`-low #1A1C20`、`-container #1F2125`、`-high #292C31`、`-highest #33353A`、`on-surface #E2E2E9`、`on-surface-variant #C4C6D0`、`outline #8E9099`、`outline-variant #44474E`、`primary #A8C7FA`、`on-primary #062E6F`、`primary-container #0842A0`、`on-primary-container #D3E3FD`、`error #FFB4AB`、`error-container #93000A`、`on-error-container #FFDAD6`、`inverse-surface #E2E2E9`、`inverse-on-surface #2F3033`
    - 映射与派生：`Background=surface`、`Surface=surface-container`、`SurfaceHover=…high`、`SurfacePressed=…highest`、`Border=outline-variant`、`Text=on-surface`、`TextMuted=on-surface-variant`、`Accent=primary`、`AccentText=on-primary`、`AccentHover=primary.Mix(onPrimary, 0.08)`、`AccentPressed=primary.Mix(onPrimary, 0.14)`、`Danger=error`、`Selection=primary-container`、`Focus=primary`、`Inverse=inverse-surface`、`InverseText=inverse-on-surface`
    - 度量：`Radius=12`、`Spacing=4`、`FontSize=13`、`ScrollbarWidth=3`
  - 形状 token 常量：`radiusSheet=16`、`radiusCard=12`、`radiusField=8`、`radiusChip=6`、`radiusPill=999`
  - `func useMD3(c *ui.Context) *ui.Theme`：`t := mdTheme(c.Theme().Dark); c.SetTheme(t); return t`（上下文主题每帧重置为系统默认，必须每帧重建）
  - `func mdColors(t *ui.Theme) mdPalette`：按 `t.Dark` 返回当前色板，供容器色阶使用
  - `icons.go`：`ui.MustParseSVG` 定义 24 视图线性图标（`stroke="currentColor"`、`stroke-width=1.8`）：`iconSearch`、`iconClose`、`iconPlus`、`iconPencil`、`iconTrash`、`iconSettings`、`iconChevronLeft`、`iconChevronDown`、`iconCheck`、`iconPanel`、`iconFolder`、`iconTag`、`iconInbox`、`iconAlert`
  - `func appMark(c *ui.Context, size float32) ui.Element`：主色圆角方块内两条白色文本条，作为应用标识
- **Acceptance Criteria**：
  - `go build ./internal/...` 通过
  - 色值只出现在 `theme.go`，视图不写十六进制
  - `useMD3` 供后续视图根统一调用；`go vet` 无未使用告警

### TASK-002：新增键入能力（与旧平台层并存）

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P0
- **Depends On**：无
- **Description**：在 `internal/platform` 纯增量加入 `Paster` 接口、Windows 实现与非 Windows 桩，复用旧文件已有的 Win32 声明，不删除任何旧代码，保证中间态可构建。
- **Details**：
  - `platform.go` 追加（不删除 `Bridge`/`Hotkey` 等既有内容）：
    - `type Paster interface { CaptureTarget(); Paste(text string) error }`
    - `func restoreClipboard(hadText bool, typed, current uint32) bool { return hadText && typed == current }`
  - `paste_windows.go`（`//go:build windows`，新增）：`type winPaster struct{ target windows.Handle }`、`func NewPaster() Paster`
    - `CaptureTarget`：`procGetForegroundWindow` 取前台，若 `procGetWindowThreadProcess` 得到的 PID 等于 `windows.GetCurrentProcessId()` 则保留旧快照，否则记录
    - `Paste`：空文本返回错误 → 目标有效则 `procSetForegroundWindow` 并 `waitUntil(100ms, 前台等于目标)` → `waitModifiersReleased(300ms)` → `readClipboardText` 备份 → `writeClipboardText(text)` → `clipboardSequence()` → `sendPasteKeystroke()` → `time.AfterFunc(150ms)` 内若 `restoreClipboard` 成立则写回备份
    - 只复用旧文件已声明的 `proc*`、`waitUntil`、`waitModifiersReleased`、`readClipboardText`、`writeClipboardText`、`clipboardSequence`、`sendPasteKeystroke`、`pointerAt`，不重复声明任何标识符（否则同包编译冲突）
  - `paste_stub.go`（`//go:build !windows`，新增）：`stubPaster`，`CaptureTarget` 空实现、`Paste` 返回 nil
  - `platform_test.go`：新增 `TestRestoreClipboard`（三例：序列号未变、用户期间复制、键入前无文本），既有快捷键/DPI 测试保持原样
- **Acceptance Criteria**：
  - `go build ./internal/...`、`GOOS=windows GOARCH=amd64 go build ./internal/...`、`go test ./internal/platform/...` 均通过
  - 旧 `Bridge`、`main.go` 调用链未被改动，仓库在 TASK-002 后仍可构建运行

### TASK-003：应用状态与数据操作

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-002
- **Description**：新增 `internal/app/app.go`、`internal/app/filter.go` 与 `internal/app/filter_test.go`，承载数据快照、CRUD、快捷键切换、插入编排与窗口注入；视图文件由后续 Task 补齐。
- **Details**：
  - `filter.go`：`FilterState{Keyword string; Scope string; GroupID *int64; TagIDs []int64}`；`filterSnippets`：关键字先 `strings.TrimSpace` 再转小写，与 `sn.Title+"\n"+sn.Content` 的小写结果做 `strings.Contains`；分组 `all/ungrouped/group` 单选；标签取交集；`nextIndex` 循环；`toggleID`、`hasID`、`hasEveryTag`、`pruneFilter`
  - `app.go`：
    - `type App struct { store *store.Store; paster platform.Paster; shortcuts Shortcuts; panel, manager *mygo.Window; groups/tags/snippets; hotkey, saved string; overlay overlayState; lib libraryState }`
    - `func (a *App) Attach(panel, manager *mygo.Window)`：注入窗口；`invalidate()` 对 nil 窗口直接跳过
    - `Reload()/reload()`：`ListGroups` + `ListTags` + `ListSnippets(GroupScope:"all")` 覆盖快照，随后 `pruneFilter` 修正两侧筛选
    - 数据操作 `saveSnippet/deleteSnippets/saveGroup/deleteGroup/saveTag/deleteTag/moveSnippets/updateTags`：写库后 `reload()`；删除后清理多选集合
    - `SetHotkey(spec string) error`：`spec == a.hotkey` 直接返回 nil（避免重复注册报错）→ `shortcuts.Register` → 写库；写库失败调 `shortcuts.Register(a.hotkey)` 回滚并返回中文错误
    - `SetStartupHotkey(registered, saved string)`：`a.hotkey, a.saved = registered, saved`（设置页展示 `saved`）
    - `ShowOverlay()`：`CaptureTarget → reload → overlay.reset() → 定位（光标所在显示器工作区水平居中、纵向 1/3）→ Show → search.Focus()`；`HideOverlay`、`ToggleOverlay`、`ShowManager(create bool)`
    - `insert(id int64)`：读取最新内容 → 主线程 `HideOverlay()` → goroutine `paster.Paste`，错误经 `mygo.RunOnMain` 回写 `lib.err` 并重绘
  - `filter_test.go`：覆盖 trim、`title+"\n"+content` 拼接、大小写、未分组、指定分组、标签交集、组合条件、`nextIndex` 边界、`toggleID`
- **Acceptance Criteria**：
  - `go test ./internal/app/...` 通过
  - 过滤语义与旧 `filter.ts` 一致（含 trim 与换行拼接）
  - 视图只做内存过滤；`store` 的过滤参数保持既有契约不变
  - 无窗口注入时（headless 测试）不 panic

### TASK-004：应用入口切换、生命周期与旧栈删除

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-001, TASK-003
- **Description**：重写 `main.go` 为 MyGo 入口（含生命周期与退出判定），新增 `mygo.json` 与托盘图标，删除旧平台层、Wails 配置与前端工程。
- **Details**：
  - `go.mod`：`go 1.27.3`；`require github.com/egoist/mygo v0.7.4`；`tool github.com/egoist/mygo/cmd/mygo`；移除 `github.com/wailsapp/wails/v2` 及专属间接依赖（`go mod tidy` 收敛）
  - `mygo.json`：`{"name":"Quick Input","identifier":"io.github.evan1777.quickinput","version":"0.2.0","icon":"build/appicon.png","out":"build"}`
  - `main.go` 顺序固定：
    1. `RequestSingleInstanceLock`（失败直接返回）
    2. `mygo.App.SetName("Quick Input")`、`mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)`、`mygo.App.OnWindowAllClosed(func(){})`
    3. 打开数据库、读 `store.SettingHotkey`（读失败仅日志）、`app.New(st, platform.NewPaster(), keys)`
    4. `mygo.App.OnSecondInstance(func(...){ application.ShowManager(false) })`
    5. `mygo.App.WhenReady(...)`：创建面板（`Panel: true, Hidden: true, Frameless: true, Transparent: true, CornerRadius: 14, Width: 560, Height: 380, DisableResize: true`）与管理窗口（`Width: 960, Height: 640, MinWidth: 760, MinHeight: 480, Hidden: true, StateKey: "manager"`）→ `application.Attach(panel, manager)` → `panel.OnClose`（`quitting` 时放行，否则 `PreventDefault + Hide`）、`panel.OnBlur(panel.Hide)`、`manager.OnClose` 同上 → `registerHotkey`（保存值 → `Alt+Space` → `Alt+Q` 依次注册，记录实际组合）→ `setUpTray`
    6. `mygo.App.Run()`
  - `quitting bool`（包级）：托盘「退出」置 `quitting = true` 后 `mygo.App.Quit()`；两个窗口的 `OnClose` 在 `quitting` 为真时不拦截
  - 托盘：`mygo.NewTray`（代码绘制 32×32 PNG 图标）+ 菜单「打开 Quick Input」「新建预设文本」、分隔、`mygo.RoleQuit`（或自定义退出项，同上置位）；`OnWillQuit` 销毁托盘
  - 删除：`frontend/`、`wails.json`、`internal/app/app_test.go`、`internal/platform/platform_windows.go`、`platform_stub.go`、旧 `platform.Bridge`/`Hotkey`/`ScaleByDPI`/`ClampWindowPosition` 及其测试；把 `winPaster` 依赖的 Win32 声明与助手迁入新的 `paste_windows.go`
- **Acceptance Criteria**：
  - `CGO_ENABLED=0 go build ./...`、`GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...`、`go vet ./...`、`go tool mygo vet ./...` 均通过
  - 应用可本地启动：托盘出现、快捷键唤出面板、关闭窗口仅隐藏
  - 托盘「退出」真正结束进程
  - 仓库无 Wails/React 残留（`grep -rn "wailsapp" --include=*.go .` 为空）

---

## Phase 2：悬浮检索面板

### TASK-005：MD3 悬浮检索面板与唯一按键路径

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-003, TASK-004
- **Description**：新增 `internal/app/view_overlay.go`，实现 560×380 面板：搜索栏 + 结果列表 + 筛选栏 + 空态，键盘只走一条路径。
- **Details**：
  - 根容器 `ui.Column(c).Fill().Background(surface).Radius(radiusSheet).Clip()`；视图首行 `useMD3(c)` 由 `App.OverlayView` 调用
  - 搜索区：`Padding(10,10)`，自绘搜索行（放大镜 + `ui.TextInput` + 清除图标按钮，`Placeholder("搜索文本或内容")`，`Radius(radiusField)`），`OnTextKey` 接管 ↑↓ 与 Esc 清空；`Changed()` 时重置 `chosen=0` 并 `list.ScrollTo(0, ui.Start)`
  - 结果列表：`ui.List(c,&list,n,row).Grow(1).Padding(6)`；`list` 只用于滚动，**不设置 `Selected`/`Selection`，列表不参与键盘**
  - 行内容：标题（13、Bold、单行）、内容预览（11.5、muted、单行）、右侧归属信息（10.5，分组与标签以「 · 」连接）；选中态 `primaryContainer` 底 + `onPrimaryContainer` 文字；`row.Hovered()` 即选中；`row.Clicked()` 触发插入
  - 键盘（搜索框保持焦点，输入控件与根级回退互斥）：
    - 搜索框 `OnTextKey` 接管 `ui.KeyDown/KeyUp` → `nextIndex` 循环移动 + `list.ScrollIntoView(chosen)`；列表不参与键盘
    - Enter：搜索框持有焦点时取 `in.Submitted()`；焦点不在输入时（`!search.FocusWithin(c)`）才取 `c.Shortcut(0, ui.KeyEnter)`，两条路径互斥，避免同帧重复插入
    - `c.Shortcut(0, ui.KeyEscape)` 统一三级：关键字非空→清空关键字并回到首行；否则存在分组/标签筛选→清除筛选；否则 `HideOverlay()`
  - 筛选栏：宽 184、`surfaceContainerLow` 底、`Pad(10,10)`，内含 `ui.Scroll`；分区标题（10.5、Bold、muted）；分组行（全部/未分组/各分组，选中 `primaryContainer`，hover `surfaceHigh`）；标签用 `ui.Toggle` 紧凑 chip 多选
  - 空态：有筛选→「没有匹配的预设文本」+「清除筛选」；无数据→「暂无预设文本」+「新建预设文本」（`ShowManager(true)`）
  - 每次 `ShowOverlay` 重置关键字、筛选、选中与滚动位置（由 TASK-003 的 `overlay.reset()` 提供）
- **Acceptance Criteria**：
  - `go build ./...` 通过
  - ↑↓ 循环、Enter 插入、Esc 三级语义与上述定义一致
  - 界面无键盘提示条与说明性文案；占位文案为中文

### TASK-006：面板视图测试

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-005
- **Description**：新增 `internal/app/view_overlay_test.go`，用 `ui.NewTester` 与内存 store 验证过滤、键盘、Esc 与插入。
- **Details**：
  - 夹具：`:memory:` store + 2 分组 3 标签 4 条预设 + `fakePaster`（记录收到的文本）；`App` 不注入窗口（nil 安全）
  - 用例：输入关键字后仅显示匹配项；↓ 两次后 Enter 插入对应内容；↑ 从首项回绕到末项；标签多选交集；关键字非空时按 Esc 清空文本且面板不隐藏；再次 Esc 清除筛选；第三次 Esc 隐藏（断言状态标志）；占位断言 `tt.HasText("Search") == false`
- **Acceptance Criteria**：
  - `go test ./internal/app/...` 通过
  - 每个用例断言状态与可见文本，不依赖像素

---

## Phase 3：管理窗口

### TASK-007：管理窗口骨架、列表与批量操作

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-001, TASK-003, TASK-004
- **Description**：新增 `internal/app/widgets.go` 与 `internal/app/view_manager.go`：应用栏、侧栏容器、工具栏、列表与批量操作栏。
- **Details**：
  - `widgets.go` 样板助手（统一形状）：`iconButton`（28×28、`radiusField`、`Label`+`Tooltip`、hover `surfaceHover`）、`filledButton`（`radiusPill`、主色底、图标+文字）、`textButton`（`radiusPill`、无底色、主色文字）、`tonalButton`（`primaryContainer`）、`dangerButton`（`errorContainer`）、`chip`（`radiusChip`、`outlineVariant` 描边）、`kbdChip`
  - 应用栏：高 52、`Padding(0,12)`；`appMark(c,22)` + 「Quick Input」（13.5、Bold）+ `Spacer` + 设置/唤出面板两个图标按钮（当前页高亮 `primaryContainer`）
  - 侧栏：宽 232、`surfaceContainerLow`、`Padding(8,8)`、内部 `ui.Scroll`；分组与标签分区标题（10.5、Bold、muted）+ 新建图标按钮；行高 30、`radiusChip`，选中 `primaryContainer`
  - 工具栏：`Padding(12,12)`、`ui.SearchField(...).Placeholder("搜索标题或内容").Grow(1)` + `filledButton(iconPlus,"新建")`
  - 列表：`ui.List(c,&list,n,row).Grow(1)`，`list.Key`/`list.Label` 绑定 ID 与标题，`list.Selection = &chosen`（`ui.Selection[int64]`）；行高 56、`Padding(8,16)`、选中 `primaryContainer`、hover `surfaceHigh`；行内标题（13、Bold）+ 预览（11.5、muted）+ 最多 3 枚 chip；hover 显示编辑/删除图标按钮；`RowDoubleClicked` 打开编辑
  - 批量栏（选择非空时）：`primaryContainer` 底、`Padding(8,12)`；「已选 N 项」+「移入分组」`ui.PopoverMenuButton`（含未分组）+「添加标签」「移除标签」+ `dangerButton("删除")` + `textButton("取消选择")`
  - 错误条：`errorContainer` 底 + `onErrorContainer` 文字 + 关闭图标按钮
  - 空态：有筛选→「清除筛选」；无数据→「新建预设文本」
  - 删除入口统一走 `ui.AlertDialog` 二次确认（取消/删除），确认后执行并清空选择
- **Acceptance Criteria**：
  - 列表随筛选与数据刷新即时更新；多选与批量栏联动
  - 全部按钮为 pill、输入为 8 圆角、卡片/容器为 12 圆角
  - 无占位或说明性文案

### TASK-008：侧栏分组与标签管理

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-007
- **Description**：补齐分组/标签的选择、内联新建、内联重命名与删除确认。
- **Details**：
  - 分组行：点击选中过滤；hover 显示重命名与删除；删除确认文案「「X」下的预设文本将变为未分组，且不可撤销。」
  - 标签行：点击切换多选过滤；hover 显示重命名与删除；确认文案「「X」将从所有预设文本中解绑，且不可撤销。」
  - 新建：点击 + 在后插入 `ui.TextInput`（`AutoFocus`、`Placeholder`、`Submitted` 提交），空值取消；草稿存于 `lib.adding.value`（不可用每帧局部变量）
  - 重命名：`ui.TextInput` 替换原行，`AutoFocus` + `Submitted` 提交，`c.Shortcut(0, ui.KeyEscape)` 取消
  - 删除后 `pruneFilter` 修正筛选；重名错误经错误条就地展示
- **Acceptance Criteria**：
  - 增删改全部生效并刷新列表与侧栏
  - 删除均需确认；取消确认不改变数据

### TASK-009：编辑侧边页

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-007
- **Description**：新增 `internal/app/view_editor.go`，以 MD3 side sheet 覆盖管理窗口右侧，承载新建与编辑。
- **Details**：
  - 结构：`ui.Overlay` + 全窗 scrim（`RGBA(0,0,0,0.32)`，点击关闭）+ 右侧面板（`Absolute().Top(8).Right(8).Bottom(8).Width(420)`、`radiusSheet`、`surfaceContainerHigh`、`Padding(20)`、`Gap(14)`）
  - 头部：标题「新建预设文本 / 编辑预设文本」+ 关闭图标按钮（`Label("关闭")`）
  - 表单：标题 `ui.Field`+`TextInput`（`AutoFocus`，不设长度上限）；内容 `ui.Field`+`TextArea(...).Lines(6, 12)`（**不要**给 `Field`/`TextArea` 加 `Grow`，否则与后续字段重叠）；分组 `ui.Select`（首项「未分组」）；标签 `ui.Toggle` chip 多选；新建标签（`TextInput` + `tonalButton("添加")`，`Submitted` 亦可提交）
  - 底部：错误文案（`p.error`）+ `Spacer` + `textButton("取消")` + `filledButton(nil,"保存")`；标题或内容 trim 为空时禁用保存
  - 保存：`saveSnippet` 成功后关闭并刷新；失败保留输入并在底部展示错误
  - Esc 关闭；打开入口：工具栏新建、行编辑/双击、托盘新建
- **Acceptance Criteria**：
  - 新建与编辑正确落库，标签全量替换语义与旧实现一致
  - 校验与错误展示不丢输入；表单无重叠

### TASK-010：设置页与快捷键切换

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P1
- **Depends On**：TASK-007, TASK-003
- **Description**：新增 `internal/app/view_settings.go`，在管理窗口内以子页承载唤出快捷键配置。
- **Details**：
  - 子页头：返回图标按钮 + 「设置」（13、Bold）
  - 内容：`ui.Scroll` + 单列 `FillWidth().MaxWidth(560).Margin(0, ui.Auto)`；分区标题「唤出快捷键」；卡片 `radiusCard` + `surface` 底 + `Clip`
  - 六个选项：`Alt+Space`、`Alt+Q`、`Ctrl+Alt+Space`、`Ctrl+Alt+Q`、`Ctrl+Shift+Space`、`Ctrl+Shift+Q`；每行高 48，键名以 `kbdChip` 逐个展示，行间 1px `outlineVariant` 分隔
  - 选中态：`primaryContainer` 底 + `onPrimaryContainer` 键帽 + 右侧 `iconCheck`；点击已选中项时 `SetHotkey` 短路返回 nil，不弹错
  - 展示值取 `a.saved`（持久化值），与实际注册值解耦；切换失败恢复选中态并在卡片内就地展示后端错误
  - 无说明性子文案；页面状态由 `lib.page` 承载，返回后保留列表筛选与选择
- **Acceptance Criteria**：
  - 六个组合均可选择并生效；失败时选择态回退且错误可读
  - 点击当前组合不产生错误

### TASK-011：管理视图测试

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-008, TASK-009, TASK-010
- **Description**：新增 `internal/app/view_manager_test.go`，覆盖新建、编辑、删除确认、筛选与设置页。
- **Details**：
  - 用例：点击「新建」→ 填写标题/内容 → 保存 → 列表出现；编辑行标题保存后更新；删除需确认后才生效；分组筛选只显示该组；标签筛选交集；切换到设置页并选择另一组合（`fakeShortcuts` 记录收到的组合）
  - `tester.SetSize(960, 640)`；窗口不注入（nil 安全）或注入零值窗口
- **Acceptance Criteria**：
  - 全部用例通过；覆盖危险操作确认分支与快捷键切换失败分支

---

## Phase 4：验证、文档与归档

### TASK-012：全量回归验证

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-001 ~ TASK-011
- **Description**：执行构建、测试、静态检查与渲染核对。
- **Details**：
  - `CGO_ENABLED=0 go build ./...`、`go vet ./...`、`go tool mygo vet ./...`、`gofmt -l .`（应无输出）、`go test ./internal/...`
  - `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...`
  - 本地用 `ui.Render` 或真实窗口截图核对四态（面板、列表、编辑侧边页、设置页）：色阶层级、间距节奏、三档圆角、文字不溢出
  - 混合 DPI：本机无 Windows 环境，记录为已知限制，由用户在 Windows 100%/125%/150% 实机验收
  - 清理 `temp/` 与临时二进制
- **Acceptance Criteria**：
  - 所有命令退出码 0；截图与设计意图一致
  - 无临时产物残留

### TASK-013：同步文档与 CI

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P1
- **Depends On**：TASK-012
- **Description**：按实际实现更新 `.docs/Project.md`、`.github/workflows/build.yml` 与 `.gitignore`。
- **Details**：
  - Project.md：概述阶段、环境与工具链（MyGo、无 Node/前端构建）、依赖（mygo + tool 指令）、结构、数据流（无 IPC）、约定（原生视图、主题收口、图标策略、过滤只在内存）、约束与决策（双窗口、生命周期、退出标志、面板定位、键入链路、快捷键展示=保存值、MD3 令牌、混合 DPI 已知限制）、术语
  - 工作流：`wails build` 改为 `go tool mygo build -platform windows/amd64`（保留 `go test`/`go vet` 步骤），产物目录与 Release 附件名按 `mygo.json` 的 `out` 写死
  - `.gitignore`：移除 `frontend/` 条目，保留 `build/bin`
- **Acceptance Criteria**：
  - 文档路径、命令、依赖与代码一致；CI YAML 语法有效且步骤可执行

### TASK-014：归档与提交

- **Status**：DONE
- **完成**：2026-11-05
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-012, TASK-013
- **Description**：回填完成状态、归档计划文档、审查后提交。
- **Details**：
  - Plan.md / Tasks.md 回填 `状态：DONE`、完成日期与回归测试结论
  - 归档至 `.docs/11-05-v1/`
  - `git status` / `git diff` 审查；提交信息：`refactor: 以 MyGo 原生 UI 重写前端并切换为紧凑 MD3 风格`
  - 不推送远端
- **Acceptance Criteria**：
  - 根目录无 Plan.md / Tasks.md；归档目录同时含两者且状态 DONE
  - 提交成功且工作区干净（忽略已忽略目录）