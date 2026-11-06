# Project

> 项目的单一事实来源，供 Agent 动手前建立全局认知。定位与设计原则见 `SCOPE.md`，工作流规则见 `AGENTS.md`。

## 概述

- **定位**：Windows 11 个人桌面效率工具，全局快捷键唤出悬浮面板，按分组与标签检索预设文本并直接键入原输入位置，常驻系统托盘并提供批量管理界面。
- **阶段**：v0.2.1 已发布（2026-11-05），完成 MyGo 原生 UI 重构与紧凑 MD3 改版；预设文本与分组均支持手动排序，快捷面板默认打开第一个分组并支持半角/全角方括号循环分组，等待 Windows 11 实机验收
- **非目标**：跨平台发行、多用户同步、富文本或图片剪贴板备份、云存储（详见 `SCOPE.md`）

## 环境与运行

- **开发平台**：WSL Linux x86_64；产物目标 Windows 11 x64
- **工具链**：Go 1.27.3、MyGo v0.7.4（`go.mod` 的 `tool` 指令提供 `go tool mygo`）；不再需要 Node.js、前端构建或 Wails CLI
- **依赖**：`github.com/egoist/mygo v0.7.4`、`modernc.org/sqlite v1.60.1`、`golang.org/x/sys v0.48.0`（见 `go.mod`）
- **本地验证**：
  - `go test ./...`
  - `go vet ./...`
  - `go tool mygo vet`（无包参数，在项目根执行）
  - `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...`
  - `go run .` 可在本机 GTK3 下启动；无 GPU 时 MyGo 自动退回 CPU 渲染
- **产物构建**：仅由 `.github/workflows/build.yml` 在 `windows-latest` 执行 `go tool mygo build -platform windows/amd64`，输出 `build/windows-amd64/Quick Input.exe`；本机不生成发布产物
- ★ 本机 Wayland 会话下全局热键依赖桌面 portal 与已安装的桌面条目、托盘依赖 `libayatana-appindicator3`，二者不作为本地验收项，以 Windows 实机为准
- ★ `mygo.json` 固定应用名、标识符、版本、图标与输出目录；图标源为 `build/appicon.png`
- ★ 数据库默认路径 `%APPDATA%\QuickInput\quick_input.db`；单元测试使用 `:memory:`，不触碰真实数据
- ★ 托盘图标由 `main.go` 在运行时绘制 32×32 PNG，与应用标识同源，不引入额外位图资源

## 结构

```
quick-input/
├─ main.go                      # MyGo 生命周期、双窗口、托盘、热键回退、数据库路径与托盘图标
├─ mygo.json                    # MyGo 构建配置（名称/标识/版本/图标/输出）
├─ build/appicon.png            # 应用图标源文件
├─ internal/
│  ├─ store/store.go            # SQLite 连接、建表与迁移、CRUD、过滤查询、预设文本与分组顺序重排、批量事务与键值设置
│  ├─ platform/
│  │  ├─ platform.go            # Paster 接口与剪贴板恢复判定
│  │  ├─ paste_windows.go       # Win32 前台快照、剪贴板替换与 Ctrl+V 合成
│  │  ├─ paste_win32.go         # 键入所需的 Win32 声明与助手（仅 windows）
│  │  └─ paste_stub.go          # 非 Windows 空实现
│  └─ app/
│     ├─ app.go                 # 应用状态、窗口编排、CRUD、快捷键切换、插入链路
│     ├─ filter.go              # 纯函数：关键字/分组/标签过滤、分组循环、方括号解析、多选切换
│     ├─ theme.go               # MD3 紧凑色板、形状 token、每帧主题接线
│     ├─ icons.go               # 内联 SVG 图标与应用标识
│     ├─ widgets.go             # 图标按钮、pill 按钮、chip、kbd chip 等样板助手
│     ├─ view_overlay.go        # 560×380 悬浮检索面板
│     ├─ view_manager.go        # 管理窗口、侧栏、列表、批量栏、确认框
│     ├─ view_editor.go         # 新建/编辑侧边页
│     └─ view_settings.go       # 设置子页、快捷键切换与面板显示开关
└─ .github/workflows/build.yml  # Windows 云端构建、测试与产物上传
```

- **数据流**：热键/托盘 → MyGo 回调 → `App.ShowOverlay` / `App.ShowManager` → 原生视图；写入路径为视图 → `App` 数据方法 → `store` → SQLite → `Reload` 刷新快照；键入路径为 `App.insert` → 隐藏面板 → 后台读取内容并 `platform.Paster.Paste`。预设文本按 `snippets.position` 升序展示：新建置顶，编辑不改位置，拖动重排后全量重编号。分组按 `groups.position` 升序展示：新建追加末尾，管理窗口侧栏拖动后全量重编号；面板分组栏与 `[` `]` 循环沿用同一顺序。
- **无 IPC**：原生视图直接读取 `App` 的内存快照，不再有 Wails 绑定或前端事件通道。
- **设置数据流**：设置页点击组合 → `App.SetHotkey` 先注册新组合 → 写 `settings` 表 → 注销旧组合；任一步失败保留原组合并在页面级错误行展示中文错误。设置页展示持久化值 `saved`，实际注册值 `hotkey` 只用于注册与回滚。`overlay_show_ungrouped`（`1`/`0`）保存面板是否展示未分组，缺失视为关闭；写库成功后才更新内存值。
- **依赖方向**：`main → app → store / platform`；`store` 与 `platform` 互不依赖，`platform` 不感知业务模型；视图与状态同在 `app` 包。

## 约定

- **命名**：Go 导出类型 PascalCase，内部函数 camelCase；视图方法 `OverlayView` / `ManagerView`
- **语言**：界面文案、错误信息与提交信息用中文；代码标识符用英文
- **单一数据源**：视图只读取 `App` 的内存快照，任何写操作完成后 `Reload` 全量刷新；不在视图或全局变量中维护第二份业务状态
- **纯函数**：过滤、分组循环、方括号解析、循环索引与多选切换集中在 `filter.go`，视图直接复用
- **过滤语义**：关键字先 `TrimSpace` 再小写，与 `Title+"\n"+Content` 的小写结果做包含匹配；分组单选；标签取交集；store 的过滤参数保持既有契约；快捷面板默认在取数后剔除未分组，管理窗口始终展示未分组
- **UI 状态**：跨帧只保存 `ui.Handle`、`ui.ListState`、`ui.Selection` 与普通字段，禁止在 `App` 中保存 `ui.Element`
- **主题**：色值只出现在 `theme.go`；每个视图根调用 `useMD3(c)` 重建主题；容器 12、输入 8、chip 6、按钮 pill、侧边页 16
- **图标**：界面图标全部为内联 SVG，仅图标按钮必须提供 `Label` 与 `Tooltip`
- **排序**：预设文本与分组均按 `position` 升序、同值时按 `id` 升序；新建预设文本置顶、新建分组追加末尾；标签按名称排序，不参与拖动
- **破坏性操作**：删除预设文本、分组、标签前必须经 `ui.AlertDialog` 二次确认
- **错误处理**：存储层以 `ErrDuplicateName`、`ErrInvalidInput` 区分可展示错误，视图在错误条或侧边页底部就地展示

## 约束与决策

- 2026-11-05 双窗口：`Panel: true` 的无边框悬浮面板 + 系统标题栏管理窗口，不启用运行期切换窗口模式的单窗口方案
- 2026-11-05 生命周期固定为 `RequestSingleInstanceLock → SetName → SetActivationPolicy → WhenReady{建窗口/Attach/注册热键/托盘} → App.Run`；MyGo 的窗口、托盘与全局快捷键必须在应用 ready 后创建
- 2026-11-05 退出用 `quitting` 标志：托盘退出置位后 `App.Quit`；两个窗口的 `OnClose` 仅在未退出时 `PreventDefault + Hide`，避免拦截退出
- 2026-11-05 面板定位取光标所在显示器工作区：水平居中、纵向 1/3；不跟随输入光标，混合 DPI 由用户实机验收
- 2026-11-05 面板键盘只有一条路径：搜索框保持焦点，上下键在输入控件的 `OnTextKey` 中循环移动，Enter 取 `Submitted`，Esc 依次清关键字 → 清筛选 → 隐藏；列表不参与键盘
- 2026-11-05 搜索栏使用自绘输入行而非 `ui.SearchField`，以便在输入控件内接管上下键并保持唯一按键路径；占位文案为中文
- 2026-11-05 键入：唤出时 `CaptureTarget`（排除本进程窗口）→ 隐藏面板 → 恢复目标焦点 → 等待修饰键释放 → 剪贴板备份/替换 → 合成 Ctrl+V → 150ms 后序列号未变才还原；只恢复 `CF_UNICODETEXT`
- 2026-11-05 全局快捷键直接使用 MyGo accelerator 字符串，持久化预设原文；注册失败按保存值 → `Alt+Space` → `Alt+Q` 回退，全部失败仍保留托盘入口
- 2026-11-05 主题为紧凑 MD3 tonal surface：`surface` / `surface-container-low` / `-high` 表达层级，侧边页与确认框叠 scrim；每帧重建以跟随系统明暗
- 2026-11-05 管理列表最多展示 3 枚归属 chip（分组 + 标签），避免长标签挤占标题
- 2026-11-05 本地不做 Windows 运行验收；`go tool mygo build` 与安装包由云端 CI 生成
- 2026-11-05 发布以 `v*` 标签为唯一入口：`main` 推送触发构建，标签推送创建 Release；`build/windows-*` 不入库
- 2026-11-05 手动排序：预设文本按 `snippets.position` 升序展示，新建取 `MIN(position)-1` 置顶，编辑不触碰 position；每次落位在单事务内全量重编号为 0..n-1，个人库规模下可接受，大库再改间隔编号；分组顺序规则见 2026-11-06 分组手动排序
- 2026-11-05 面板分组循环：`[` 向前、`]` 向后，范围为「各分组 + 全部」（开关开启时末尾追加未分组）；MyGo 的 `OnTextKey` 不接收可打印按键，方括号先进入绑定关键字，面板构建时剥离并作为指令，一次最多翻一步
- 2026-11-05 未分组开关：默认关闭，面板既不展示未分组条目也不展示「未分组」筛选项；管理窗口语义不变；关闭时若面板正筛选未分组则重置为默认作用域
- 2026-11-05 已知限制：启动时持久化组合被占用则回退并仍显示保存值；UIPI 下无法向管理员窗口键入；Linux 开发会话的热键与托盘能力受桌面环境限制；方括号（半角 `[]` 与全角 `【】`）不能作为搜索字符（粘贴含方括号文本会被消费，净方向非 0 时触发翻组）
- 2026-11-06 分组手动排序：`groups.position` 升序决定分组展示顺序，新建分组追加末尾，编辑（重命名）不触碰 position；管理窗口侧栏分组区改用 `ui.List` 拖动，每次落位在单事务内全量重编号为 0..n-1；面板分组栏与循环顺序随之变化，面板内不提供拖动
- 2026-11-06 面板默认作用域：取分组循环的第一步（第一个分组，无分组时退回全部），唤出与 Esc 第二阶段共用；「清除筛选」按钮仍重置为全部
- 2026-11-06 全角方括号：`【` `】` 与 `[` `]` 等效翻组，在方括号解析纯函数中统一剥离；代价是方括号不能再作为搜索字符
- 2026-11-06 管理窗口侧栏：「全部 / 未分组」的位置与默认「全部」筛选保持不变，仅分组区可拖动

## 外部集成

- **GitHub Actions**：公开仓库 `Evan-1777/Quick_Input`；`main` 推送在 `windows-latest` 执行测试与 `go tool mygo build` 并上传 `Quick Input.exe`，`v*` 标签额外创建 GitHub Release；无运行时外部服务，不含任何密钥或凭据

## 术语

- **预设文本（snippet）**：可被检索并键入的一段纯文本，归属至多一个分组、可绑定多个标签
- **悬浮面板（overlay）**：快捷键唤出的 `560×380` 检索面板，失焦自动收起
- **管理窗口（manager）**：`960×640` 主管理视图，含侧栏、批量操作与分类管理
- **设置页**：管理窗口内的二级子页面，承载唤出快捷键等低频全局配置
- **侧边页（side sheet）**：覆盖管理窗口右侧的 `420` 宽新建/编辑表单
- **唤出快捷键**：从任意应用唤出悬浮面板的全局组合键，可在设置页切换并持久化
- **目标窗口**：唤出面板前快照记录的前台窗口句柄，键入前恢复其焦点
- **未分组**：`group_id IS NULL` 的预设文本
- **手动顺序**：由 `position` 升序决定的预设文本与分组展示顺序，可通过管理窗口拖动调整；新建预设文本置顶、新建分组追加末尾

## 维护

| 变更 | 同步章节 |
|---|---|
| 运行、测试方式或环境前提变化 | 环境与运行 |
| 依赖增删或升级 | 环境与运行；属关键选型时同步「约束与决策」 |
| 模块增删、重构，依赖方向变化 | 结构 |
| 命名、语言、日志规范调整 | 约定 |
| 新坑、新约束、选型定型 | 约束与决策 |
| 外部服务接入或下线 | 外部集成 |
| 新增项目特有名词 | 术语 |
