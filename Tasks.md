# Tasks：Quick Input 0.1.0 首次发布

**关联 Plan**：`Plan.md` —— Quick Input 0.1.0 首次发布 v1.0  
**总计 Task**：9 个

---

## Phase 1：仓库基线与忽略规则

### TASK-001：创建 .gitignore 排除依赖与产物

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends On**：无
- **Description**：在项目根目录新增 `.gitignore`，排除 `frontend/node_modules/`、`frontend/dist/`、`build/bin/`、本地数据库与 Agent 工具目录。
- **Details**：
  - 忽略 `frontend/node_modules/`、`frontend/dist/`、`build/bin/`
  - 忽略 `*.db`、`.DS_Store`、`Thumbs.db`
  - 忽略 `.codebuddy/`、`.agents/`、`.dsh/`、`.mcp.json`
  - 保留 `build/appicon.png`（CI 图标源文件）
- **Acceptance Criteria**：
  - `git status --porcelain` 不列出 `node_modules`、`dist` 或工具目录
  - `build/appicon.png` 仍可被跟踪

### TASK-002：在 wails.json 声明 0.1.0 版本元数据

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends On**：无
- **Description**：为 `wails.json` 增加 `info` 字段，写入产品名与版本号，使 Windows 产物携带 0.1.0 元数据。
- **Details**：
  - `info.productName` 为 `Quick Input`
  - `info.productVersion` 为 `0.1.0`
- **Acceptance Criteria**：
  - `wails.json` 为合法 JSON（`node -e` 解析通过）
  - 字段命名与 Wails v2 配置一致

### TASK-003：为构建工作流增加标签触发与 Release 任务

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends On**：无
- **Description**：修改 `.github/workflows/build.yml`，在 `v*` 标签推送时构建并创建 GitHub Release，上传 `quick-input.exe`。
- **Details**：
  - `on.push` 同时匹配 `branches: [main]` 与 `tags: ['v*']`
  - 顶层声明 `permissions.contents: write`
  - 新增 `release` job，`needs: windows`，仅在 `refs/tags/v*` 运行
  - 使用 `actions/download-artifact@v4` 取产物，`gh release create "$TAG"` 创建发布并附资产
- **Acceptance Criteria**：
  - YAML 可解析且 job 依赖关系正确
  - Release 步骤仅在标签触发时执行

### TASK-004：本地回归验证

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-001, TASK-003
- **Description**：执行前端测试与构建、后端测试与双平台静态检查，确认发布前基线无回归。
- **Details**：
  - `cd frontend && npm test && npm run build`
  - `go test ./internal/... && go vet ./internal/...`
  - `GOOS=windows GOARCH=amd64 go vet ./...`
- **Acceptance Criteria**：
  - 全部命令退出码为 0

### TASK-005：初始化 Git 仓库并创建首个提交

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-002, TASK-004
- **Description**：执行 `git init -b main`，暂存源码与文档，创建符合 Conventional Commits 的首个中文提交。
- **Details**：
  - 提交前用 `git status` 审查暂存清单，确认无密钥与本机配置
  - 提交信息形如 `feat: 首次提交 Quick Input 桌面快捷文本键入工具`
- **Acceptance Criteria**：
  - `git log --oneline` 存在首个提交
  - 工作区无未跟踪源码文件

---

## Phase 2：公开仓库与发布

### TASK-006：创建公开 GitHub 仓库并推送 main

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-005
- **Description**：使用 `gh repo create Quick_Input --public --source=. --remote=origin --push` 创建并推送。
- **Details**：
  - 仓库归属当前登录账号 `Evan-1777`
  - 推送后确认 `origin/main` 与本地 HEAD 一致
- **Acceptance Criteria**：
  - `gh repo view` 显示 `public`
  - `git rev-parse HEAD` 与 `origin/main` 相同

### TASK-007：确认 main 推送触发 CI 构建成功

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-006
- **Description**：等待 `build` 工作流对 main 的首次运行结束，读取结论。
- **Details**：
  - `gh run list --branch main` 获取运行 ID
  - `gh run watch <id> --exit-status` 等待结果
- **Acceptance Criteria**：
  - 运行结论为 `success`

### TASK-008：创建并推送 v0.1.0 标签完成发布

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-007
- **Description**：创建附注标签 `v0.1.0` 并推送，触发标签构建与 Release 任务。
- **Details**：
  - `git tag -a v0.1.0 -m "v0.1.0"`
  - `git push origin v0.1.0`
  - 等待该次运行结束，确认 `release` job 成功
- **Acceptance Criteria**：
  - 标签运行结论为 `success`
  - `gh release view v0.1.0` 返回发布信息

### TASK-009：验收 Release 资产并同步项目文档

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P1
- **Depends On**：TASK-008
- **Description**：验证 Release 资产存在且非空，更新 `Project.md` 记录仓库、CI 与发布方式，归档计划文档。
- **Details**：
  - `gh release view v0.1.0 --json assets` 确认 `quick-input.exe` 大小大于 0
  - 更新 `.docs/Project.md` 的外部集成与阶段信息
  - 将根目录 `Plan.md` / `Tasks.md` 移入 `.docs/11-04-v2/` 并提交
- **Acceptance Criteria**：
  - Release 资产可列出且大小非零
  - 根目录无残留 Plan.md / Tasks.md
  - 最终提交包含文档更新与归档
