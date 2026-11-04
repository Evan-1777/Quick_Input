# Tasks：设置页、可配置唤出快捷键与项目图标

**关联 Plan**：`Plan.md` —— 设置页、可配置唤出快捷键与项目图标 v1.1  
**状态**：DONE  
**完成日期**：2026-11-04  
**回归测试结论**：`npm test` 7/7 通过，`npm run build`、`go test ./internal/...`、`go vet ./internal/...`、`GOOS=windows GOARCH=amd64 go vet ./...`、`gofmt -l internal main.go` 与 `git diff --check` 全部通过；SVG 在 32/256px 渲染无裁切。  
**总计 Task**：12 个

---

## Phase 1：设置持久化与快捷键模型

### TASK-001：新增 settings 键值表与读写方法

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：无
- **Description**：在 `internal/store/store.go` 的建表清单中加入 `settings` 表，提供通用键值读写；在 `internal/store/store_test.go` 补充往返测试。
- **Details**：
  - `schema` 追加：`CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)`
  - 导出常量 `const SettingHotkey = "hotkey"`，供 `main` 与 `app` 引用，避免字符串分散
  - `func (s *Store) GetSetting(key string) (string, error)`：未命中（`sql.ErrNoRows`）返回 `"", nil`，其他错误原样返回
  - `func (s *Store) SetSetting(key, value string) error`：使用 `INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`，保证重复写入为覆盖
  - 测试 `TestSettingsRoundTrip`：新库读取不存在键返回空；写入后读取一致；再次写入同键返回最新值
- **Acceptance Criteria**：
  - `go test ./internal/store/...` 全部通过
  - `:memory:` 库可正常建表与读写
  - 既有测试无回归

### TASK-002：实现快捷键解析与规范格式化

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：无
- **Description**：在 `internal/platform/platform.go` 增加跨平台快捷键模型、错误值与解析函数；在 `internal/platform/platform_test.go` 增加表驱动测试。
- **Details**：
  - 常量：`DefaultHotkey = "Alt+Space"`；修饰键位 `ModControl = 0x0002`、`ModAlt = 0x0001`、`ModShift = 0x0004`、`ModWin = 0x0008`（与 Win32 `MOD_*` 对齐）
  - 错误：`var ErrInvalidHotkey = errors.New("快捷键格式无效")`
  - 类型：`type Hotkey struct { Modifiers uint32; Key uint32 }`
  - `func ParseHotkey(spec string) (Hotkey, error)`：
    - 按 `+` 拆分，去除空 token，token 大小写不敏感
    - 修饰键接受 `Ctrl`、`Alt`、`Shift`、`Win`，顺序任意、至少一个、不得重复
    - 键名映射：`A`–`Z` → `0x41`–`0x5A`；`0`–`9` → `0x30`–`0x39`；`Space` → `0x20`；`Enter` → `0x0D`；`Tab` → `0x09`；`F1`–`F12` → `0x70`–`0x7B`
    - 无修饰键、多个非修饰键、未知 token、空串均返回 `ErrInvalidHotkey`
  - `func (h Hotkey) String() string`：固定输出顺序 `Ctrl+Alt+Shift+Win+Key`，键名使用上述规范写法；未知键码返回空串
  - 测试覆盖：`Alt+Space`、`ctrl+shift+q`、`F12+Ctrl`（乱序）、`Ctrl+Alt+1`；`String()` 往返；`""`、`Space`、`Ctrl+`、`Alt+Q+Space`、`Alt+Alt+Q`、`Alt+F13`、`Hyper+Q` 均报错
- **Acceptance Criteria**：
  - `go test ./internal/platform/...` 全部通过
  - 合法输入的 `ParseHotkey` → `String()` 往返结果稳定

---

## Phase 2：平台热键热切换与绑定

### TASK-003：平台层支持启动注册与运行期热键切换

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-001, TASK-002
- **Description**：扩展 `Bridge` 契约与两种实现：启动时按传入组合注册，运行期通过消息窗口在消息线程内切换；`main.go` 读取持久化组合后传入；在 `platform_test.go` 为启动回退序列补纯函数测试。
- **Details**：
  - `internal/platform/platform.go`
    - `Bridge` 增加 `SetHotkey(spec string) error`，注释说明「可从任意 goroutine 调用，注册必须发生在消息线程」
    - `NewBridge(windowTitle, initialHotkey string) Bridge`：`initialHotkey` 为空时使用 `DefaultHotkey`
    - 新增 `func hotkeyFallbacks(initial string) []string`：按顺序返回去重后的启动尝试序列——初始组合合法时排首位（用 `ParseHotkey` + `String()` 规范化），其后拼接 `DefaultHotkey`、`Alt+Q`；初始为空或非法时直接从 `DefaultHotkey` 开始
  - `internal/platform/platform_stub.go`
    - 构造函数保存初始组合（空则 `DefaultHotkey`），新增导出字段 `HotkeySpec string` 供本地断言
    - `SetHotkey` 先 `ParseHotkey` 校验，成功则写 `HotkeySpec = parsed.String()`
  - `internal/platform/platform_windows.go`
    - 常量：`wmSetHotkey = wmApp + 2`、`modNoRepeat = 0x4000`、`vkLWin = 0x5B`、`vkRWin = 0x5C`
    - `winBridge` 增加 `initialHotkey string`、`hotkey Hotkey`、`hotkeyCallMu sync.Mutex`（串行化调用方）、`pendingMu sync.Mutex` 与 `pending *hotkeyRequest`；`hotkeyRequest{ spec string; done chan error }`，`done` 使用容量 1 的缓冲通道
    - `initMessageWindow` 用 `registerInitialHotkey()` 取代写死的 `Alt+Space`/`Alt+Q`：遍历 `hotkeyFallbacks(b.initialHotkey)` 依次注册，首个成功者生效并记录 `b.hotkey`；全部失败保留托盘入口；注册掩码统一加 `modNoRepeat`
    - `SetHotkey`：先 `ParseHotkey` 校验；`b.hwnd == 0` 返回「平台层未就绪」；持有 `hotkeyCallMu`；把请求写入 `pending` 后 `PostMessageW(b.hwnd, wmSetHotkey, 0, 0)`；等待 `done` 或 2 秒超时
    - 超时返回「切换快捷键超时」。请求可能已在消息线程执行；`done` 带缓冲，迟到结果不会阻塞消息线程；超时路径不重复投递，也不额外定义迟到结果的同步通道（该场景仅在消息线程被占用时出现，用户重试后状态收敛）
    - `wndProc` 增加 `wmSetHotkey` 分支，调用 `applyPendingHotkey()`
    - `applyPendingHotkey`：`pending` 为空时立即 return（覆盖超时后迟到的空消息）；否则取出并清空，先 `UnregisterHotKey(hwnd, hotkeyID)`，再注册新组合；失败则重新注册旧组合并返回中文错误「快捷键 %s 注册失败，可能已被其他程序占用」；成功更新 `b.hotkey` 并写回 `done`
    - `b.hwnd` 在 `Start` 完成后只读、销毁路径清零；`SetHotkey` 读取时不加锁（与既有 `Stop` 同模式），避免与 `hotkeyCallMu` 形成锁序
    - `waitModifiersReleased` 同时检查 `vkCtrl`、`vkMenu`、`vkShift`、`vkLWin`、`vkRWin` 均为释放状态，避免自定义组合残留修饰键导致 `Ctrl+Shift+V`
    - 退出清理继续注销当前热键，无需改动 ID
  - `main.go`：启动时 `st.GetSetting(store.SettingHotkey)`，读取失败仅 `log.Printf` 并按空值处理；以该值调用 `NewBridge`
  - `internal/platform/platform_test.go`：`TestHotkeyFallbacks` 覆盖空串、默认组合、自定义组合、非法组合、小写输入，断言顺序与去重
  - 不修改热键 ID、托盘菜单与窗口行为；不新增依赖
- **Acceptance Criteria**：
  - `gofmt -l internal main.go` 无输出
  - `go test ./internal/...` 通过（含 `TestHotkeyFallbacks`）
  - `GOOS=windows GOARCH=amd64 go vet ./...` 零错误（前置：`frontend/dist` 存在；缺失时先 `cd frontend && npm run build`）
  - 代码检查确认 `RegisterHotKey` 只出现在消息线程路径（初始化或 `applyPendingHotkey`）

### TASK-004：应用绑定层暴露快捷键设置并纳入 Bootstrap

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-003
- **Description**：在 `internal/app/app.go` 增加设置负载与 `SetHotkey` 绑定方法，Bootstrap 返回当前快捷键；同步 `internal/app/app_test.go` 的测试桩与用例。
- **Details**：
  - `type SettingsPayload struct { Hotkey string `json:"hotkey"` }`
  - `BootstrapPayload` 增加 `Settings SettingsPayload `json:"settings"``
  - `GetBootstrapData`：读取 `store.SettingHotkey`，为空时用 `platform.DefaultHotkey`；读取失败返回错误
  - `func (a *App) SetHotkey(spec string) error`：读取当前设置作为回滚值（为空用 `platform.DefaultHotkey`）→ `platform.ParseHotkey` 校验 → 以 `h.String()` 调 `a.bridge.SetHotkey` → 成功后再 `a.store.SetSetting(store.SettingHotkey, h.String())`
    - 读取设置失败时直接返回，不注册
    - 注册失败：直接返回注册错误，数据库保持旧值
    - 写库失败：以操作前的组合调用 `a.bridge.SetHotkey(prev)` 回滚，再返回「保存设置失败，已恢复原快捷键」包装错误；回滚本身失败时返回「保存设置失败且无法恢复原快捷键」并保留底层错误
  - 测试桩 `fakeBridge` 增加 `SetHotkey` 与记录字段（最后收到的规范组合、调用序列、可配置错误）
  - 新增测试：
    - 设置合法组合后，桩收到规范组合、数据库持久化、Bootstrap `settings.hotkey` 同步
    - 非法组合返回错误，桩未被调用，数据库仍为空（Bootstrap 返回默认值）
    - 桩返回错误时，应用返回错误且数据库不写入
  - 写库失败回滚分支由 `store` 为具体类型、无法在应用层注入写失败，不为此引入接口抽象；以代码审查确认回滚调用与错误文案
- **Acceptance Criteria**：
  - `go test ./internal/app/...` 全部通过
  - Bootstrap 不含 nil 切片等既有行为回归

---

## Phase 3：设置页与导航

### TASK-005：前端类型、API 与全局动作接入快捷键设置

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-004
- **Description**：在 `frontend/src/types.ts`、`frontend/src/api.ts`、`frontend/src/App.tsx` 补齐设置负载与 `setHotkey` 调用链。
- **Details**：
  - `types.ts`：新增 `SettingsPayload { hotkey: string }`；`BootstrapPayload` 增加 `settings`；`AppActions` 增加 `setHotkey(spec: string): Promise<void>`
  - `api.ts`：`WailsBridge` 增加 `SetHotkey(spec: string): Promise<void>`；新增 `setHotkey` 包装，未连接后端时与 `saveSnippet` 一致返回 reject（`未连接应用后端`）；`emptyPayload` 增加 `settings: { hotkey: 'Alt+Space' }`
  - `App.tsx`：`actions.setHotkey` 先 `api.setHotkey(spec)` 再 `refresh()`；管理分支将 `settings={data.settings}` 传给 `ManagerView`
- **Acceptance Criteria**：
  - `cd frontend && npx tsc --noEmit` 通过
  - 无新增依赖

### TASK-006：新增设置页组件与快捷键选择

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-005
- **Description**：在 `frontend/src/components/Icons.tsx` 增加设置与返回图标，新建 `frontend/src/components/SettingsPage.tsx` 实现快捷键单选页。
- **Details**：
  - 图标：`SettingsIcon`（中心圆 + 八向短齿，16 viewBox、沿用 `Svg` 包装）；`ChevronLeftIcon`（`M9.8 3.8 5.6 8l4.2 4.2`）
  - 预设组合常量（顺序即展示顺序）：`Alt+Space`、`Alt+Q`、`Ctrl+Alt+Space`、`Ctrl+Alt+Q`、`Ctrl+Shift+Space`、`Ctrl+Shift+Q`
  - 组件 Props：`{ hotkey: string; onBack: () => void; onSelect: (spec: string) => Promise<void> }`
  - 结构：
    - 顶部子页头：高 48px、白底、下边框；返回 `IconButton`（label「返回预设文本」）+ 标题「设置」13px 半粗
    - 内容区：`bg-canvas` 可滚动，内层 `mx-auto w-full max-w-[560px]`，上边距 20px
    - 分区标题「快捷键」11.5–12px、`text-muted`、半粗
    - 白色卡片：1px `border-line`、8px 圆角；首行说明「唤出快捷键」+ 副文案「在任意应用中唤出快捷输入面板。」；下方为选项列表
    - 每个选项为整行可点区域：原生 `<input type="radio" name="hotkey">` 使用 `sr-only`，外层 `<label>` 绑定，`focus-within` 显示 2px 品牌色焦点环；行内左侧以 `kbd` 芯片展示组合（按 `+` 拆分，边框与底色沿用 QuickOverlay 现有 kbd，字号取底栏一致的 `text-[10.5px]`），右侧选中时显示 `CheckIcon` 与「使用中」
    - 选中行使用 `bg-brand-soft/60` 与 `text-brand`；未选中悬停 `bg-black/[0.025]`
    - 列表下方错误区：`role="alert"`，`text-danger` 11.5px，展示后端返回消息
  - 交互：点击未选中项时置本地 `pending` 为该组合（即时高亮），`await onSelect(spec)`；成功由父级刷新数据后清除 `pending`；失败清除 `pending` 并展示错误；请求中禁用全部单选
  - 无障碍：选项容器 `role="radiogroup"` + `aria-label="唤出快捷键"`；不引入自定义键盘导航
  - 仅使用现有 Tailwind 语义色（`canvas/surface/ink/muted/faint/line/brand/brand-soft/danger`）与 4–8px 圆角，不新增依赖
- **Acceptance Criteria**：
  - `npx tsc --noEmit` 通过
  - 六个预设均可选中并触发 `onSelect`
  - 失败时选中态回到 `hotkey` 且错误可读

### TASK-007：管理窗口接入设置入口与页面切换

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-006
- **Description**：修改 `frontend/src/components/ManagerView.tsx`，增加设置页状态、标题栏齿轮入口与页面渲染切换。
- **Details**：
  - Props 增加 `settings: SettingsPayload`
  - 新增 `page: 'library' | 'settings'` 状态，默认 `library`
  - 全局标题栏右侧按钮组最左侧加 `IconButton`（label「设置」，`SettingsIcon`），点击 `setPage('settings')`
  - 设置态下用 `SettingsPage` 替换「侧栏 + 主列表」整个主体区域，全局标题栏保留；`onBack` 回到 `library`
  - `SettingsPage` 的 `hotkey` 取 `settings.hotkey`，`onSelect` 调用 `actions.setHotkey`
  - `creatorRequested` 生效时同时 `setPage('library')`，保证托盘「新建预设文本」不会被设置页遮挡
  - 编辑器抽屉、确认弹窗、错误条仅在列表态渲染；不改动列表/批量操作逻辑
- **Acceptance Criteria**：
  - `npm run build` 通过
  - 点击齿轮进入设置页、返回恢复列表；进入设置页不丢失分组筛选状态（组件状态保留）
  - 托盘新建动作回到列表并打开编辑器

---

## Phase 4：项目 SVG 图标

### TASK-008：绘制 build/appicon.svg

- **Status**：DONE
- **Executor**：主线
- **Priority**：P1
- **Depends On**：无
- **Description**：新建 `build/appicon.svg`，以矢量复刻现有应用图标的视觉语言，并做适度精修。
- **Details**：
  - 画布 `viewBox="0 0 512 512"`，`role="img"`、`aria-label="Quick Input"`，仅使用内联 `defs`，禁止外部引用与文字
  - 背景：全幅 `rect` `rx=116`，纵向线性渐变 `#0a84e0` → `#0066b8`
  - 卡片：白色圆角矩形 `x=112 y=84 width=288 height=344 rx=36`，加 `feDropShadow`（`dy=10`、`stdDeviation=14`、深蓝 20% 透明度）
  - 内容：两条品牌蓝 `#0078d4` 文本条 `x=152 y=176/228 width=208 height=24 rx=12`；下方两条浅蓝 `#9cc7e9` 短条 `y=280`，分别为 `width=116`、`width=52`（第二条 `x=284`），`height=20 rx=10`
  - 保持与 `build/appicon.png` 同构：蓝色圆角底、白色文本卡片、深/浅蓝横条；不引入字母或额外装饰
  - 不替换 `build/appicon.png`，不改动 Wails 构建配置
- **Acceptance Criteria**：
  - `python3 -c "import xml.etree.ElementTree as ET; ET.parse('build/appicon.svg')"` 通过（或 `xmllint --noout`）
  - 32px 截图下两条主条与白色卡片可分辨（几何换算：条高 1.5px、条间距 1.75px）
  - 64px 与 256px 截图无裁切、无越界，缩略轮廓完整
  - 文件不含 `<script>`、外链 `href` 或位图

### TASK-009：以简化标识替换管理窗口标题栏图标

- **Status**：DONE
- **Executor**：主线
- **Priority**：P1
- **Depends On**：TASK-007, TASK-008
- **Description**：在 `frontend/src/components/Icons.tsx` 增加独立 24 视图的 `AppMarkIcon`，并在 `ManagerView.tsx` 标题栏使用，移除占位标签图标。
- **Details**：
  - `AppMarkIcon` 不复用 `Svg` 包装（其固定 `viewBox="0 0 16 16"`、`fill="none"`、`stroke="currentColor"`，与实色标识冲突）；改为独立 `<svg viewBox="0 0 24 24" stroke="none" aria-hidden="true" className={...}>`，`className` 默认 `h-4 w-4`
  - 24 视图几何（与 512 图标同源同色，非同几何）：
    - 背景：`rect width=24 height=24 rx=5.4`，实色 `#0072c6`
    - 卡片：`rect x=4.8 y=3.6 width=14.4 height=16.8 rx=1.8`，白色
    - 主条两条：`x=7.2 width=9.6 height=2.2 rx=1.1`，`y=6.8` 与 `y=10.8`，`#0078d4`
    - 短条：`x=7.2 y=15 width=5.6 height=1.8 rx=0.9`，`#a8cdec`
  - 尺寸换算：18px 下主条约 1.65px、条间距约 1.35px、短条约 1.35px，满足最小可分辨间隙
  - `ManagerView` 标题栏左侧：删除 `bg-brand` 包裹的 `TagIcon`，改为 `<AppMarkIcon className="h-[18px] w-[18px]" />`，文字「Quick Input」保持
  - 清理顺序：先替换引用并移除 `TagIcon` 导入，再删除 `Icons.tsx` 中的 `TagIcon` 导出，最后 `grep -rn "TagIcon" frontend/src` 确认无结果
- **Acceptance Criteria**：
  - `npm run build` 通过
  - 18px 下两条主条可分辨（按几何换算条间距 ≥1.3px），轮廓不依赖渐变或投影
  - 无未使用导入或图标残留

---

## Phase 5：回归验证、文档与归档

### TASK-010：执行全量回归验证

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-001 ~ TASK-009
- **Description**：执行后端、前端与交叉静态检查，确认无编译错误与回归。
- **Details**：
  - 先构建前端：`cd frontend && npm test && npm run build`（`node_modules` 缺失时先 `npm ci`）；`frontend/dist` 是 `main.go` 的 `//go:embed` 前置，必须在 Go 检查前生成
  - `go test ./internal/...`
  - `go vet ./internal/...`
  - `GOOS=windows GOARCH=amd64 go vet ./...`（依赖上一步生成的 `frontend/dist`，干净检出时不可跳过前端构建）
  - `gofmt -l internal main.go`（应无输出）
  - `git diff --check`
  - 若本机 Chrome 可用：将 `build/appicon.svg` 渲染为 32px 与 256px 截图并目视检查，临时文件放 `temp/` 并在检查后删除
- **Acceptance Criteria**：
  - 所有命令退出码为 0
  - `npm test` 通过数 ≥7 且 fail 为 0，生产构建成功
  - 无临时文件残留

### TASK-011：同步 Project.md

- **Status**：DONE
- **Executor**：主线
- **Priority**：P1
- **Depends On**：TASK-010
- **Description**：按代码实际状态更新 `.docs/Project.md`。
- **Details**：
  - 「结构」：store 说明补充设置键值；新增 `frontend/src/components/SettingsPage.tsx` 与 `build/appicon.svg`
  - 「数据流」：Bootstrap 增加 `settings`；补充「设置页 → `SetHotkey` → 平台消息线程重注册 → 写入 settings → 刷新 Bootstrap」链路
  - 「约定」或「约束与决策」：记录快捷键以规范字符串持久化、仅消息线程注册、预设组合集合、失败回滚不写库、启动回退的已知限制
  - 「术语」：补充「唤出快捷键」定义
  - 不修改与本次无关的章节
- **Acceptance Criteria**：
  - 文档中的路径、方法与实际代码一致
  - 无过期描述或重复段落

### TASK-012：归档计划文档并提交

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-010, TASK-011
- **Description**：将根目录 `Plan.md` / `Tasks.md` 标记完成并归档，审查变更后创建中文 Conventional Commit。
- **Details**：
  - 在 `Plan.md` 与 `Tasks.md` 头部回填 `状态：DONE`、完成日期 2026-11-04、回归测试结论（引用 TASK-010 实际结果）
  - 归档目录按 `.docs/MM-DD-vN` 生成，查询同日目录取最大 N + 1（当前预期 `11-04-v3`）
  - 移动两个文件至归档目录，确认根目录无残留
  - `git status` / `git diff` / `git log --oneline` 审查；提交信息形如 `feat: 新增设置页与可配置唤出快捷键及项目图标`
  - 不推送远端，不修改 CI 配置
- **Acceptance Criteria**：
  - `.docs/<归档目录>/` 同时包含 `Plan.md` 与 `Tasks.md` 且状态为 DONE
  - 根目录无 `Plan.md` / `Tasks.md`
  - 提交成功，工作区干净（忽略已忽略目录）
