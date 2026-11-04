# Tasks：Win11 轻量级快捷文本键入工具

**关联 Plan**：`Plan.md` —— Win11 轻量级快捷文本键入工具 v1.1  
**状态**：DONE  
**完成日期**：2026-11-04  
**回归测试结论**：2026-11-04 通过 —— 前端 `npm test` 7/7、`tsc --noEmit` 与 `vite build` 通过；后端 `go test ./internal/...` 12/12、`go vet ./internal/...` 与 `GOOS=windows go vet ./...` 零错误。  
**总计 Task**：11 个  

<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; border: 1px solid #d4d4d8; border-radius: 6px; padding: 14px; background: #fafafa; color: #18181b; margin: 12px 0;">
  <div style="font-size: 13px; font-weight: 600; color: #3f3f46; margin-bottom: 10px; border-bottom: 1px solid #e4e4e7; padding-bottom: 6px;">任务拓扑与阶段交付矩阵</div>
  <div style="display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; font-size: 11px;">
    <div style="border: 1px solid #d4d4d8; border-radius: 4px; background: #ffffff; padding: 8px;">
      <div style="font-weight: 600; color: #18181b; margin-bottom: 4px;">Phase 1：数据存储层</div>
      <div style="color: #52525b; line-height: 1.5;">
        • TASK-001：初始化 Go/Wails 工程配置<br/>
        • TASK-002：实现 SQLite 存储与批量操作<br/>
        • TASK-003：编写存储层单元测试
      </div>
    </div>
    <div style="border: 1px solid #d4d4d8; border-radius: 4px; background: #ffffff; padding: 8px;">
      <div style="font-weight: 600; color: #18181b; margin-bottom: 4px;">Phase 2：平台与服务层</div>
      <div style="color: #52525b; line-height: 1.5;">
        • TASK-004：定义平台接口与边界裁剪函数<br/>
        • TASK-005：实现 Win32 交互与 Linux 测试桩<br/>
        • TASK-006：实现 App 绑定服务与入口
      </div>
    </div>
    <div style="border: 1px solid #d4d4d8; border-radius: 4px; background: #ffffff; padding: 8px;">
      <div style="font-weight: 600; color: #18181b; margin-bottom: 4px;">Phase 3：双视图前端</div>
      <div style="color: #52525b; line-height: 1.5;">
        • TASK-007：搭建前端配置与纯函数单测<br/>
        • TASK-008：实现快捷悬浮窗组件<br/>
        • TASK-009：实现主管理窗口与批量操作
      </div>
    </div>
    <div style="border: 1px solid #d4d4d8; border-radius: 4px; background: #ffffff; padding: 8px;">
      <div style="font-weight: 600; color: #18181b; margin-bottom: 4px;">Phase 4：验证与交付</div>
      <div style="color: #52525b; line-height: 1.5;">
        • TASK-010：跨平台检查与 CI 工作流配置<br/>
        • TASK-011：编写 `.docs/Project.md` 基线文档
      </div>
    </div>
  </div>
</div>

---

## Phase 1：项目初始化与本地数据存储层

### TASK-001：初始化 Go 模块与 Wails 项目配置

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：无
- **Description**：在项目根目录创建 `go.mod` 与 `wails.json`，声明 Go 模块路径、Wails v2 依赖、纯 Go SQLite 驱动 `modernc.org/sqlite` 及 `golang.org/x/sys` 依赖。
- **Details**：
  - 创建 `wails.json`，仅包含 Wails CLI 标准构建与前端目录配置字段：`name: "quick-input"`、`outputfilename: "quick-input"`、`frontend:dir: "frontend"`、`frontend:install: "npm install"`、`frontend:build: "npm run build"`、`frontend:dev:watcher: "npm run dev"`、`frontend:dev:serverUrl: "auto"`（窗口启动属性统一在 `main.go` 的 `options.App` 中设置）。
  - 创建 `go.mod`（模块名 `quick-input`），引入 `github.com/wailsapp/wails/v2`、`modernc.org/sqlite`、`golang.org/x/sys` 并执行 `go mod tidy` 生成 `go.sum`。
- **Acceptance Criteria**：
  - 根目录存在合法的 `go.mod`、`go.sum` 与 `wails.json`。
  - 执行 `go env GOMOD` 指向当前工作区 `go.mod`。

### TASK-002：实现 SQLite 存储层与批量归类事务

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-001
- **Description**：在 `internal/store/store.go` 中定义领域结构体与 `Store` 类型，实现 SQLite 连接初始化、表结构创建、预设文本/分组/标签 CRUD、多维过滤查询及批量分组/标签事务方法。
- **Details**：
  - 定义领域结构体（全库统一约定 `GroupID *int64` 中 `nil` 表示「未分组」）：
    - `Group`：`ID int64`、`Name string`、`CreatedAt string`
    - `Tag`：`ID int64`、`Name string`、`CreatedAt string`
    - `Snippet`：`ID int64`、`Title string`、`Content string`、`GroupID *int64`、`GroupName string`、`Tags []Tag`、`UpdatedAt string`
    - `SnippetFilter`：`Keyword string`、`GroupScope string`（`"all"` 全部、`"ungrouped"` 未分组、`"group"` 指定分组）、`GroupID *int64`（仅当 `GroupScope == "group"` 时读取）、`TagIDs []int64`（多选标签交集过滤）
    - `SnippetInput`：`ID int64`、`Title string`、`Content string`、`GroupID *int64`（`nil` 表示未分组）、`TagIDs []int64`
  - 实现 `NewStore(dbPath string) (*Store, error)` 与 `Close() error`：
    - 当 `dbPath != ":memory:"` 且非 `file:` 前缀时，先调用 `os.MkdirAll(filepath.Dir(dbPath), 0o755)` 创建父目录。
    - 在 SQLite DSN 中附带 `_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)`，并调用 `db.SetMaxOpenConns(1)` 与 `db.SetMaxIdleConns(1)`，确保 `:memory:` 测试与文件数据库共享单连接且外键约束始终生效。
    - 执行幂等建表语句：
      - `groups (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE COLLATE NOCASE, created_at TEXT NOT NULL)`
      - `tags (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE COLLATE NOCASE, created_at TEXT NOT NULL)`
      - `snippets (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL, content TEXT NOT NULL, group_id INTEGER REFERENCES groups(id) ON DELETE SET NULL, updated_at TEXT NOT NULL)`
      - `snippet_tags (snippet_id INTEGER NOT NULL REFERENCES snippets(id) ON DELETE CASCADE, tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE, PRIMARY KEY (snippet_id, tag_id))`
  - 定义领域错误 `ErrDuplicateName = errors.New("名称已存在")` 与 `ErrInvalidInput = errors.New("输入参数不能为空")`，当触发 SQLite `UNIQUE constraint failed` 时映射并返回 `ErrDuplicateName`。
  - 实现分组管理函数：`ListGroups() ([]Group, error)`、`SaveGroup(id int64, name string) (Group, error)`、`DeleteGroup(id int64) error`（事务内将该分组下预设文本的 `group_id` 置为 `NULL` 后删除分组）。
  - 实现标签管理函数：`ListTags() ([]Tag, error)`、`SaveTag(id int64, name string) (Tag, error)`、`DeleteTag(id int64) error`（级联删除 `snippet_tags` 关联记录）。
  - 实现预设文本管理函数：`ListSnippets(filter SnippetFilter) ([]Snippet, error)`、`GetSnippetByID(id int64) (Snippet, error)`、`SaveSnippet(in SnippetInput) (Snippet, error)`（单事务内执行 `snippets` 新增或更新，随后 `DELETE FROM snippet_tags WHERE snippet_id = ?` 并全量重新插入 `in.TagIDs`）、`DeleteSnippets(ids []int64) error`。
  - 实现批量操作事务函数：
    - `BatchMoveGroup(snippetIDs []int64, groupID *int64) error`：单事务批量更新指定预设文本的分组归属（`groupID == nil` 时设为未分组）。
    - `BatchUpdateTags(snippetIDs []int64, addTagIDs []int64, removeTagIDs []int64) error`：单事务内使用 `INSERT OR IGNORE INTO snippet_tags` 批量绑定标签，并按 `DELETE FROM snippet_tags` 批量解绑标签。
- **Acceptance Criteria**：
  - `go build ./internal/store/...` 编译通过，无错误。
  - 所有写操作包含非空参数校验与唯一约束映射。

### TASK-003：编写存储层单元测试

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-002
- **Description**：在 `internal/store/store_test.go` 中编写单元测试，验证父目录自动创建、唯一性冲突、CRUD、单选分组归属、标签多维交集过滤及批量操作事务。
- **Details**：
  - 验证传入不存在的多级子目录路径（`filepath.Join(t.TempDir(), "nested", "dir", "quick_input.db")`）时 `NewStore` 自动创建父目录并成功初始化库表。
  - 验证分组与标签创建、重命名、大小写不敏感重名冲突（断言 `errors.Is(err, ErrDuplicateName)`）以及删除后的关联清理（删除分组后原文本 `GroupID == nil`，删除标签后原文本 `Tags` 不再包含该标签）。
  - 验证 `SaveSnippet` 编辑时在单事务内全量替换标签列表（原绑定 `[A, B]` 更新为 `[B, C]` 后仅保留 `[B, C]`）。
  - 验证 `ListSnippets` 按关键字、`GroupScope`（`"all"` / `"ungrouped"` / `"group"`）与多选标签交集（同时包含标签 A 与标签 B）组合筛选。
  - 验证 `BatchMoveGroup` 对多条预设文本一键切换分组及重置为未分组（`nil`）。
  - 验证 `BatchUpdateTags` 对多条预设文本批量新增标签（含重复添加幂等性）与批量移除标签。
- **Acceptance Criteria**：
  - 执行 `go test -v ./internal/store/...` 全部用例通过，退出码为 0。

---

## Phase 2：原生平台交互层与应用编排服务层

### TASK-004：定义平台抽象接口与工作区边界裁剪纯函数

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-001
- **Description**：在 `internal/platform/platform.go` 与 `internal/platform/platform_test.go` 中定义平台交互接口 `Bridge`、窗口模式常量、物理像素坐标与 DPI 换算纯函数及单元测试。
- **Details**：
  - 定义窗口模式常量：`ModeOverlay = "overlay"`、`ModeManager = "manager"`，默认逻辑尺寸常量：`OverlayLogicalWidth = 560`、`OverlayLogicalHeight = 380`、`ManagerLogicalWidth = 960`、`ManagerLogicalHeight = 640`。
  - 定义结构体 `Point{ X, Y int }`、`Rect{ Left, Top, Right, Bottom int }` 与回调配置 `Callbacks{ OnToggleOverlay func(), OnOverlayBlur func(), OnTrayNewSnippet func(), OnTrayOpenManager func(), OnTrayQuit func() }`。
  - 定义纯函数 `ScaleByDPI(logicalPx, dpi int) int`（当 `dpi <= 0` 时按 `96` 处理，返回 `(logicalPx * dpi + 48) / 96`）与 `ClampWindowPosition(anchor Point, physWidth, physHeight int, workArea Rect) Point`（基于光标/鼠标物理坐标计算悬浮窗左上角坐标，默认偏移 `8px`，当超出显示器 `workArea` 右/下/左/上边界时裁剪至可见工作区内）。
  - 定义 `Bridge` 接口：
    - `Start(cb Callbacks) error`：启动全局热键与系统托盘消息循环。
    - `Stop()`：注销热键与托盘图标。
    - `ShowOverlayAtCursor(logicalWidth, logicalHeight int) Point`：若当前窗口未处于悬浮窗可见态，先快照记录非本进程的当前前景窗口句柄 `targetHWND`；随后获取输入光标物理坐标（失败回退鼠标坐标），按窗口 DPI 换算物理宽高并裁剪至当前显示器工作区，将窗口置顶移动并正常激活显示（使搜索框可直接接收键盘输入）。
    - `ShowManagerWindow(logicalWidth, logicalHeight int)`：取消置顶工具窗样式，按 DPI 换算尺寸并居中于主显示器工作区，正常激活显示至前台。
    - `HideWindow()`：隐藏当前窗口。
    - `MinimizeWindow()`：最小化当前窗口。
    - `PasteTextToTarget(text string) error`：隐藏窗口后，将先前快照的 `targetHWND` 恢复为前景窗口并轮询确认焦点就绪（上限 `100ms`）；轮询等待物理修饰键（`Alt`、`Shift`）释放（上限 `300ms`）；备份当前剪贴板 `CF_UNICODETEXT` 文本，写入目标文本并记录 `GetClipboardSequenceNumber()`；通过 `SendInput` 发送 `Ctrl+V` 按键序列；异步等待 `150ms` 后若 `GetClipboardSequenceNumber()` 未发生变化则恢复原纯文本剪贴板。
  - 在 `internal/platform/platform_test.go` 中编写 `ScaleByDPI`（100%/125%/150% 缩放）与 `ClampWindowPosition`（常规位置、右边缘溢出、下边缘溢出、左上角越界）的单元测试。
- **Acceptance Criteria**：
  - 执行 `go test -v ./internal/platform/...` 全部通过。

### TASK-005：实现 Windows 原生系统交互与 Linux 编译测试桩

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-004
- **Description**：创建 `internal/platform/platform_windows.go`（`//go:build windows`）与 `internal/platform/platform_stub.go`（`//go:build !windows`），分别实现 Win32 原生能力与 Linux 测试环境桩。
- **Details**：
  - 在 `internal/platform/platform_windows.go` 中通过 `golang.org/x/sys/windows` 与 `syscall.NewLazyDLL`（`user32.dll`、`kernel32.dll`、`shell32.dll`）实现 `NewBridge(windowTitle string) Bridge`：
    - **本进程主窗口定位**：实现 `resolveOwnHWND()`，通过 `EnumWindows` + `GetWindowThreadProcessId` 匹配当前进程 `uint32(os.Getpid())`（排除后台隐藏消息窗口句柄）定位并缓存 Wails 主窗口句柄，避免仅依赖标题查找。
    - **消息线程与托盘/热键**：在 `runtime.LockOSThread()` 协程中创建消息窗口，调用 `RegisterHotKey` 注册全局快捷键（默认 `Alt+Space`，失败回退 `Alt+Q`），并通过 `Shell_NotifyIconW` 注册系统托盘图标；在 `WndProc` 中处理 `WM_HOTKEY` 与托盘右键菜单（`CreatePopupMenu` 提供「新建预设文本」「打开管理窗口」「退出」）。
    - **句柄快照、光标定位与 DPI 适配**：在 `ShowOverlayAtCursor` 中，先通过 `GetForegroundWindow` 获取当前活动窗口 `fg`，若 `fg != 0 && fg != ownHWND` 则更新 `targetHWND = fg`；调用 `GetGUIThreadInfo` 获取 `rcCaret` 并经 `ClientToScreen` 转为屏幕物理坐标，若 `hwndCaret == 0` 或矩形为空则调用 `GetCursorPos` 回退至鼠标物理坐标；通过 `GetDpiForWindow(ownHWND)` 获取 DPI 并调用 `ScaleByDPI` 计算物理宽高，配合 `MonitorFromPoint` + `GetMonitorInfoW` 的 `rcWork` 调用 `ClampWindowPosition`；最后调用 `SetWindowPos`（置顶 `HWND_TOPMOST`）、`ShowWindow(SW_SHOW)` 与 `SetForegroundWindow(ownHWND)` 激活悬浮窗。
    - **原窗口焦点还原与安全粘贴**：在 `PasteTextToTarget` 中，先调用 `ShowWindow(ownHWND, SW_HIDE)`；若 `targetHWND != 0` 则调用 `SetForegroundWindow(targetHWND)` 并以 `10ms` 步长轮询 `GetForegroundWindow() == targetHWND`（最多 `100ms`）；以 `10ms` 步长轮询 `GetAsyncKeyState(VK_MENU)` 与 `GetAsyncKeyState(VK_SHIFT)` 直至高位清零（最多 `300ms`）；在剪贴板操作中读取原 `CF_UNICODETEXT`（若存在），写入目标 UTF-16 文本并记录 `seq := GetClipboardSequenceNumber()`；调用 `SendInput` 发送 `VK_CONTROL` 按下、`'V'` 按下、`'V'` 抬起、`VK_CONTROL` 抬起共 4 个 `INPUT` 事件；启动后台定时器在 `150ms` 后检查 `GetClipboardSequenceNumber() == seq`，相同时还原旧文本或清空临时文本。
  - 在 `internal/platform/platform_stub.go` 中实现非 Windows 环境的内存测试桩（记录 `LastPastedText`、`Visible`、`CurrentMode`、`LastPosition` 状态，供服务层单测直接断言）。
- **Acceptance Criteria**：
  - Linux 本机执行 `go test -v ./internal/platform/...` 通过。
  - Windows 交叉静态检查 `GOOS=windows GOARCH=amd64 go vet ./internal/platform/...` 零错误通过。

### TASK-006：实现应用编排服务层与 Wails 单实例启动入口

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-002, TASK-005
- **Description**：在 `internal/app/app.go`、`internal/app/app_test.go` 与根目录 `main.go` 中实现前后端绑定服务与单实例窗口生命周期编排。
- **Details**：
  - 在 `internal/app/app.go` 中定义事件常量 `EventWindowMode = "window:mode"`、窗口状态负载 `WindowStatePayload{ Mode string, OpenCreator bool }` 及启动数据包 `BootstrapPayload{ Groups []store.Group, Tags []store.Tag, Snippets []store.Snippet, WindowState WindowStatePayload }`。
  - `App` 结构体持有 `ctx context.Context`、`store *store.Store`、`bridge platform.Bridge`、`state WindowStatePayload` 及可注入的 `emit func(eventName string, data any)`（生产环境在 `Startup(ctx)` 中绑定为 `runtime.EventsEmit(ctx, ...)`，单元测试可注入切片收集器）。
  - 暴露供前端调用的绑定方法（保持单一数据源，不设冗余查询接口）：
    - 数据读写：`GetBootstrapData() (BootstrapPayload, error)`、`SaveSnippet(in store.SnippetInput) (store.Snippet, error)`、`DeleteSnippets(ids []int64) error`、`SaveGroup(id int64, name string) (store.Group, error)`、`DeleteGroup(id int64) error`、`SaveTag(id int64, name string) (store.Tag, error)`、`DeleteTag(id int64) error`。
    - 批量操作：`BatchMoveGroup(snippetIDs []int64, groupID *int64) error`、`BatchUpdateTags(snippetIDs []int64, addTagIDs []int64, removeTagIDs []int64) error`。
    - 窗口与键入控制：`TriggerPaste(snippetID int64) error`（通过 `store.GetSnippetByID(snippetID)` 读取最新 `Content`，调用 `bridge.PasteTextToTarget` 并更新窗口状态）、`HideWindow()`、`MinimizeWindow()`、`SwitchToManager(openCreator bool)`（调用 `bridge.ShowManagerWindow` 并发射 `EventWindowMode` 事件）、`SwitchToOverlay()`（调用 `bridge.ShowOverlayAtCursor` 并发射 `EventWindowMode` 事件）。
  - 在 `internal/app/app_test.go` 中编写服务层单元测试，验证：
    - 热键唤出悬浮窗、托盘触发「新建预设文本」与「打开管理窗口」时发射的 `window:mode` 事件序列与 `OpenCreator` 标志完全匹配。
    - 新建预设文本并修改内容后调用 `TriggerPaste(id)`，平台测试桩收到的粘贴内容为数据库内最新修改后的文本（验证单一数据源）。
    - 批量移动分组与批量增删标签后，`GetBootstrapData()` 返回的关联结果准确同步。
  - 在根目录 `main.go` 中配置 `wails.Run(&options.App{...})`：
    - 显式设置 `Frameless: true`、`StartHidden: true`、`HideWindowOnClose: true`。
    - 配置 `SingleInstanceLock: &options.SingleInstanceLock{ UniqueId: "quick-input-win11-lock", OnSecondInstanceLaunch: func(_ options.SecondInstanceData) { application.SwitchToManager(false) } }`，确保双击可执行文件时唤醒已有实例并打开管理窗口。
- **Acceptance Criteria**：
  - 执行 `go test -v ./internal/app/...` 全部通过。
  - 执行 `GOOS=windows GOARCH=amd64 go vet ./internal/...` 零错误通过。

---

## Phase 3：单实例双视图前端界面实现

### TASK-007：搭建锁定版本的前端工程、纯函数过滤模块与单元测试

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-006
- **Description**：在 `frontend/` 下创建锁定依赖版本的 React 18 + TypeScript + Vite + TailwindCSS v3 工程，实现类型定义、后端绑定封装、纯函数多维过滤与键盘导航逻辑及其自动化单元测试。
- **Details**：
  - 创建 `frontend/package.json`，显式锁定兼容版本：`react: "18.3.1"`、`react-dom: "18.3.1"`、`typescript: "5.6.3"`、`vite: "5.4.11"`、`@vitejs/plugin-react: "4.3.3"`、`tailwindcss: "3.4.17"`、`postcss: "8.4.49"`、`autoprefixer: "10.4.20"`，并配置 `"test": "node --test src/filter.test.ts"`。
  - 创建 `frontend/tsconfig.json`、`frontend/vite.config.ts`、`frontend/tailwind.config.js`、`frontend/postcss.config.js`、`frontend/index.html`、`frontend/src/styles.css` 与 `frontend/src/main.tsx`。
  - 在 `frontend/src/types.ts` 中定义 `Group`、`Tag`、`Snippet`、`SnippetInput`、`GroupScope`（`'all' | 'ungrouped' | 'group'`）、`FilterState`、`WindowMode`（`'overlay' | 'manager'`）及 `WindowStatePayload`。
  - 在 `frontend/src/filter.ts` 中实现纯函数：
    - `filterSnippets(snippets: Snippet[], filter: FilterState): Snippet[]`：支持按关键字（大小写不敏感匹配 `title` 或 `content`）、单选分组（`'all'` 全部 / `'ungrouped'` 未分组即 `groupId === null` / `'group'` 指定分组 ID）与多选标签交集（`selectedTagIds` 中每个标签 ID 均存在于条目的 `tags` 中）联合过滤。
    - `nextActiveIndex(currentIndex: number, total: number, direction: 'up' | 'down'): number`：计算键盘上下方向键循环导航索引（`total <= 0` 时返回 `0`）。
    - `toggleSelection(selectedIds: number[], targetId: number): number[]`：切换多选复选框集合。
  - 在 `frontend/src/filter.test.ts` 中使用 Node 24 内置 `node:test` 与 `node:assert/strict` 编写单元测试，覆盖关键字检索、全部/未分组/指定分组单选切换、多标签 AND 交集过滤、空结果与循环键盘导航边界。
  - 在 `frontend/src/api.ts` 中封装对 `window.go.app.App` 与 `window.runtime.EventsOn` 的调用；非 Wails 宿主环境下仅返回最小静态空状态兜底（不维护 `localStorage` 模拟数据库）。
- **Acceptance Criteria**：
  - 在 `frontend/` 下执行 `npm test`（`node --test src/filter.test.ts`）全部断言通过。
  - 在 `frontend/` 下执行 `npx tsc --noEmit` 零类型错误。

### TASK-008：实现快捷悬浮窗组件（链路一）

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-007
- **Description**：在 `frontend/src/components/QuickOverlay.tsx` 中实现无边框快捷悬浮窗界面，包含左侧搜索框与键盘导航列表、右侧分组单选与标签多维过滤区，以及窗口失焦自动隐藏处理。
- **Details**：
  - **布局结构**：紧凑双栏无边框面板（适配 `560x380` 逻辑尺寸）：
    - **左侧主区（占宽 62%）**：顶部为自动聚焦（`autoFocus`）的即时检索输入框与快捷键提示（`↑↓` 选择、`Enter` 键入、`Esc` 隐藏），下方为预设文本列表（展示标题、所属分组徽标、关联标签及单行内容预览）。
    - **右侧筛选区（占宽 38%）**：上半部分为「分组（单选归属）」列表（含「全部」「未分组」及各具体分组），下半部分为「标签（多维过滤）」胶囊区（支持多选叠加取交集过滤，提供清空筛选按钮），底部提供打开管理窗口入口按钮。
  - **键盘、鼠标与失焦交互**：
    - 使用 `filterSnippets` 实时计算过滤列表，过滤条件变更时自动将激活索引重置为 `0`。
    - 监听键盘 `ArrowDown` / `ArrowUp`（调用 `nextActiveIndex`）、`Enter`（调用 `onTriggerPaste(activeItem.id)` 隐藏悬浮窗并由后端从数据库读取文本键入目标窗口）、`Escape`（调用 `onHide()` 隐藏悬浮窗）。
    - 点击右侧分组或标签后自动将键盘焦点保持在搜索输入框（`searchInputRef.current?.focus()`），确保鼠标点击筛选与键盘方向键/回车键入无缝衔接。
    - 监听窗口 `blur` 事件：当悬浮窗失去前台焦点时调用 `onHide()` 自动收起悬浮窗。
- **Acceptance Criteria**：
  - 组件通过 TypeScript 严格类型检查，绑定的过滤与导航逻辑全部复用经过单测验证的 `filter.ts` 纯函数。

### TASK-009：实现带自定义标题栏的主管理窗口组件与双视图容器（链路二、链路三）

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-008
- **Description**：在 `frontend/src/components/ManagerView.tsx` 与 `frontend/src/App.tsx` 中实现带自定义拖拽标题栏的主管理窗口界面（快速新建/编辑、列表快速多选、批量移入分组、批量增删标签、分组与标签 CRUD）及顶层双视图容器。
- **Details**：
  - 在 `frontend/src/components/ManagerView.tsx` 中实现：
    1. **顶部自定义窗口标题栏**：设置 `style={{ '--wails-draggable': 'drag' }}` 支持无边框窗口鼠标拖拽移动；右侧放置「切换悬浮窗预览」「最小化」（调用 `api.minimizeWindow()`）与「关闭隐藏至托盘」（调用 `api.hideWindow()`）控制按钮（按钮区域设置 `--wails-draggable: no-drag`）。
    2. **左侧分类管理栏**：展示分组列表与标签列表，支持内联新建、重命名及带二次确认提示的删除操作（提示将该分组下文本重置为未分组或解绑该标签），并支持点击筛选右侧列表。
    3. **中间预设文本列表与批量操作栏**：
       - 顶部提供关键字搜索框、「新建预设文本」按钮、「全选当前列表 / 取消全选」复选框。
       - 当选中项 `selectedIds.length > 0` 时，显式展示**批量操作工具栏**，支持：
         - **批量移入分组**：选择目标分组或「设为未分组（`null`）」，调用 `api.batchMoveGroup` 并在完成后刷新全局数据集。
         - **批量添加标签**：多选目标标签，调用 `api.batchUpdateTags(selectedIds, addTagIds, [])`。
         - **批量移除标签**：多选目标标签，调用 `api.batchUpdateTags(selectedIds, [], removeTagIds)`。
         - **批量删除**：经二次确认后调用 `api.deleteSnippets(selectedIds)`。
    4. **快速录入与编辑弹窗/抽屉**：支持录入标题、多行文本内容、单选归属分组、多选绑定已有标签（并预留快捷调用 `api.saveTag` 新建标签入口）；当后端随 `window:mode` 传入 `openCreator: true`（对应托盘右键「新建预设文本」）时自动展开新建表单并聚焦标题输入框；保存成功后立即刷新顶层共享状态，使快捷悬浮窗即刻生效。
  - 在 `frontend/src/App.tsx` 中维护单一 `groups / tags / snippets` 状态源，初始化时调用 `api.getBootstrapData()`，订阅后端 `window:mode` 事件切换 `QuickOverlay` 与 `ManagerView`。
- **Acceptance Criteria**：
  - 在 `frontend/` 下执行 `npm run build` 成功生成 `frontend/dist` 生产资源，且 `npx tsc --noEmit` 零错误。

---

## Phase 4：自动化验证、云端构建配置与项目文档建立

### TASK-010：配置云端 Windows 构建流水线并执行全栈跨平台验证闭环

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-006, TASK-009
- **Description**：创建 `.github/workflows/build.yml` 实现 Windows 11 原生产物云端构建与测试流水线，并在本地 WSL 环境执行内部包单测、前端单测与生产构建、Windows 全量交叉静态检查闭环。
- **Details**：
  - 创建 `.github/workflows/build.yml`：在 `windows-latest` 环境配置 Go、Node.js 与 Wails CLI，依次执行 `cd frontend && npm ci && npm test && npm run build`、`go test -v ./...`、`wails build -platform windows/amd64`，并将生成的 `build/bin/quick-input.exe` 上传为工作流构建产物（Artifact）。
  - 在本地 WSL 环境按顺序执行以下验证命令：
    1. `cd frontend && npm test && npx tsc --noEmit && npm run build`（验证前端过滤/导航单测、类型检查并生成 `frontend/dist` 供 `main.go` 的 `embed.FS` 嵌入）。
    2. `go test -v ./internal/...` 与 `go vet ./internal/...`（在 WSL Linux 无 GTK 环境下验证全部业务与存储/平台测试）。
    3. `GOOS=windows GOARCH=amd64 go vet ./...`（在 Windows 目标平台下对包含 `main.go` 与 `platform_windows.go` 的全量代码执行静态类型与调用检查）。
  - 清理临时构建缓存，确认无残留后台进程。
- **Acceptance Criteria**：
  - `.github/workflows/build.yml` 存在且步骤完整。
  - 本地执行上述 3 组验证命令退出码全部为 0，无残留开发服务器进程。

### TASK-011：创建 `.docs/Project.md` 项目基线文档

- **Status**：DONE
- **Executor**：主线
- **Priority**：P1
- **Depends On**：TASK-010
- **Description**：对照 `.docs/Project.example.md` 格式与代码库实际结构，创建 `.docs/Project.md`，记录项目概述、环境与运行（含 WSL 本地测试命令与 Windows 云端构建说明）、目录结构与数据流、代码约定、关键架构决策及 Windows 11 验收核对项。
- **Details**：
  - 完整描述 `main.go`、`internal/store`、`internal/platform`、`internal/app`、`frontend/` 与 `.github/workflows/build.yml` 的职责及依赖方向。
  - 标注数据库默认路径（`%APPDATA%\QuickInput\quick_input.db`）、单连接外键配置与测试隔离方式（`:memory:`）。
  - 记录单实例无边框双视图切换、唤出前前景句柄快照与激活恢复、剪贴板序列号校验粘贴、UIPI 及非文本剪贴板边界说明。
- **Acceptance Criteria**：
  - `.docs/Project.md` 内容与代码库实际文件路径、模块命名及依赖关系完全一致。
