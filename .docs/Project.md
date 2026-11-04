# Project

> 项目的单一事实来源，供 Agent 动手前建立全局认知。定位与设计原则见 `SCOPE.md`，工作流规则见 `AGENTS.md`。

## 概述

- **定位**：Windows 11 个人桌面效率工具，全局快捷键唤出悬浮面板，按分组与标签检索预设文本并直接键入原输入位置，常驻系统托盘并提供批量管理界面。
- **阶段**：v0.1.0 已发布，等待 Windows 11 实机验收
- **非目标**：跨平台发行、多用户同步、富文本或图片剪贴板备份、云存储（详见 `SCOPE.md`）

## 环境与运行

- **开发平台**：WSL Linux x86_64；产物目标 Windows 11 x64
- **工具链**：Go 1.26、Node.js 24、Wails CLI v2.16.0
- **后端依赖**：`github.com/wailsapp/wails/v2 v2.16.0`、`modernc.org/sqlite v1.60.1`、`golang.org/x/sys v0.48.0`（见 `go.mod`）
- **前端依赖**：React 18.3.1、TypeScript 5.6.3、Vite 5.4.11、TailwindCSS 3.4.17，版本锁定于 `frontend/package.json`
- **本地验证**：
  - `go test -v ./internal/...` 与 `go vet ./internal/...`
  - `cd frontend && npm test && npx tsc --noEmit && npm run build`
  - `GOOS=windows GOARCH=amd64 go vet ./...`（全量包交叉静态检查）
- **产物构建**：仅由 `.github/workflows/build.yml` 在 `windows-latest` 执行 `wails build`，本机不构建 Windows 产物
- ★ 根包 `main.go` 依赖 Wails Linux 图形绑定（GTK/WebKit），WSL 下无法编译；本地仅实测 `internal/` 包，根包用 Windows 交叉检查与链接构建验证
- ★ `main.go` 通过 `//go:embed all:frontend/dist` 嵌入前端；执行任何 Go 侧检查前必须先 `npm run build`
- ★ 数据库默认路径 `%APPDATA%\QuickInput\quick_input.db`；单元测试使用 `:memory:`，不触碰真实数据
- ★ CI 首包需 `build/appicon.png`；`build/windows/` 下的图标、清单与版本信息由 Wails 构建时自动生成

## 结构

```
quick-input/
├─ main.go                      # 单实例窗口选项、托盘/热键回调接线、数据库路径
├─ wails.json                   # Wails 构建与前端目录配置
├─ build/appicon.png            # 应用图标源文件
├─ build/appicon.svg            # 矢量图标源文件（界面使用独立 24 视图简化标识）
├─ internal/
│  ├─ store/store.go            # SQLite 连接、建表、CRUD、过滤查询、批量事务与键值设置
│  ├─ platform/platform.go      # Bridge 接口、窗口尺寸常量、DPI 换算与工作区裁剪纯函数
│  ├─ platform/platform_windows.go  # Win32 热键/托盘/光标/焦点/剪贴板粘贴实现
│  ├─ platform/platform_stub.go     # 非 Windows 内存桩，供本地开发运行
│  └─ app/app.go                # Wails 绑定方法与窗口模式事件编排
├─ frontend/
│  ├─ src/App.tsx               # 单一数据源容器，双视图切换与事件订阅
│  ├─ src/filter.ts             # 纯函数：多维过滤、键盘导航、多选切换
│  ├─ src/api.ts                # window.go / window.runtime 封装与空态兜底
│  ├─ src/components/QuickOverlay.tsx  # 快捷悬浮窗
│  ├─ src/components/ManagerView.tsx   # 主管理窗口、批量操作、分类管理
│  ├─ src/components/SettingsPage.tsx  # 二级设置页、唤出快捷键单选
│  ├─ src/components/EditorDrawer.tsx  # 新建/编辑抽屉
│  └─ src/components/ui.tsx     # 复用按钮与图标按钮
└─ .github/workflows/build.yml  # Windows 云端构建、测试与产物上传
```

- **数据流**：热键/托盘 → `platform.Bridge` 原生消息 → `app.App` 回调 → `window:mode` 事件 → React 切换视图；写入路径 React → `api.ts` → `App` 绑定方法 → `store` → SQLite，随后重新拉取 bootstrap 数据；键入路径 `App.TriggerPaste` → `store.GetSnippetByID` 读取最新内容 → `platform.PasteTextToTarget`。
- **设置数据流**：Bootstrap 携带 `settings.hotkey` → 设置页单选 → `App.SetHotkey` 校验并规范化 → 消息线程重注册热键 → 写入 `settings` 表 → 前端刷新 Bootstrap；任一步失败保持原组合与原设置。
- **依赖方向**：`frontend → app → store / platform`；`store` 与 `platform` 互不依赖，`platform` 不感知业务模型。平台差异通过 `//go:build windows` 与 `//go:build !windows` 隔离。

## 约定

- **命名**：Go 导出类型 PascalCase，内部函数 camelCase；TS 组件 PascalCase，纯函数 camelCase
- **语言**：界面文案、错误信息与提交信息用中文；代码标识符用英文
- **单一数据源**：前端只缓存后端返回的 bootstrap 数据，任何写操作完成后刷新全量数据，不在前端维护第二份状态或 `localStorage`
- **纯函数**：过滤、键盘导航与多选切换集中在 `filter.ts`，组件直接复用，不重复实现
- **快捷键**：以规范字符串（`Ctrl+Alt+Shift+Win+Key` 顺序）在 `settings` 表持久化；解析与格式化集中在 `platform.ParseHotkey` / `Hotkey.String()`，前后端只交换该格式
- **破坏性操作**：删除预设文本、分组、标签前必须二次确认
- **错误处理**：存储层以 `ErrDuplicateName`、`ErrInvalidInput` 区分可展示错误，绑定层原样返回，界面在操作区就地展示

## 约束与决策

- 2026-11-04 单实例无边框窗口 + 前端自定义拖拽标题栏，不启用第二个窗口——避免双 WebView2 进程与运行期切换系统边框导致的重绘错位
- 2026-11-04 全局热键默认 `Alt+Space`，被占用时回退 `Alt+Q`，失败仍保留托盘入口
- 2026-11-04 唤出快捷键支持设置页单选预设组合（`Alt+Space`、`Alt+Q`、`Ctrl+Alt+Space`、`Ctrl+Alt+Q`、`Ctrl+Shift+Space`、`Ctrl+Shift+Q`）；注册只发生在消息线程，切换经自定义窗口消息投递
- 2026-11-04 快捷键切换先注册新组合、成功后才写库；注册失败保留旧组合，写库失败回滚注册，界面就地展示中文错误
- 2026-11-04 已知限制：启动时持久化组合被占用则回退至 `Alt+Space` / `Alt+Q`，设置页仍显示保存值，不做额外状态通道
- 2026-11-04 键入采用「隐藏自身 → 恢复目标窗口焦点 → 等待修饰键释放 → 剪贴板替换 + 合成 Ctrl+V → 150ms 后校验序列号还原」；只备份 `CF_UNICODETEXT`，非文本格式（图片、文件列表）不恢复
- 2026-11-04 显式设置目标窗口快照仅在当前前台非本进程窗口时更新，避免悬浮窗自身被记录为目标
- 2026-11-04 平台层统一在物理像素空间计算位置与尺寸，`GetGUIThreadInfo` 失败或光标矩形为空时回退鼠标坐标
- 2026-11-04 SQLite 使用纯 Go 驱动、单连接池并在 DSN 开启 `foreign_keys`，保证 `:memory:` 与文件库行为一致；分组删除在事务内将 `group_id` 置空，标签删除级联清理关联表
- 2026-11-04 UIPI 边界：普通权限进程无法向管理员权限窗口注入按键；需向提权窗口键入时必须以同等权限运行本工具
- 2026-11-04 本地不做 Windows 全量构建；`wails build` 与安装包验收走云端 CI，本机只跑内部包测试与交叉静态检查
- 2026-11-04 发布以 `v*` 标签为唯一入口：`main` 推送触发构建，标签推送创建 Release；`frontend/dist`、`build/bin` 与本地 Agent 配置不入库

## 外部集成

- **GitHub Actions**：公开仓库 `Evan-1777/Quick_Input`；`main` 推送在 `windows-latest` 构建 `quick-input.exe` 并上传为构建产物，`v*` 标签额外创建 GitHub Release 并附加可执行文件；无运行时外部服务，不含任何密钥或凭据

## 术语

- **预设文本（snippet）**：可被检索并键入的一段纯文本，归属至多一个分组、可绑定多个标签
- **悬浮窗（overlay）**：快捷键唤出的 `560×380` 快捷检索面板，失焦自动收起
- **管理窗口（manager）**：`960×640` 主管理视图，含批量操作与分类管理
- **设置页**：管理窗口内的二级子页面，承载唤出快捷键等低频全局配置
- **唤出快捷键**：从任意应用唤出悬浮窗的全局组合键，可在设置页切换并持久化
- **目标窗口**：唤出悬浮窗前快照记录的前台窗口句柄，键入前恢复其焦点
- **未分组**：`group_id IS NULL` 的预设文本

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
