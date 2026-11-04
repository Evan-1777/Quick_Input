# Plan：Quick Input 0.1.0 首次发布

**状态**：IN_PROGRESS  
**日期**：2026-11-04  
**版本**：v1.0  
**回归测试结论**：待发布流水线验收后回填。

---

## 1. 背景与目标

Quick Input 功能实现已完成并通过本地测试，但项目尚未纳入 Git 版本管理，也没有公开代码仓库与版本发布通道。本次交付建立首次公开发布的完整链路：

- 初始化 Git 仓库并完成首个提交
- 创建公开 GitHub 仓库 `Quick_Input` 并推送 `main`
- 推送触发 `windows-latest` 云端构建，产物可下载
- 打 `v0.1.0` 标签，自动构建并发布带 Windows 可执行文件的 GitHub Release

## 2. 阶段划分

### Phase 1：仓库基线与忽略规则

| 项目 | 内容 |
|------|------|
| **输入** | 现有源码、`frontend/node_modules`、`frontend/dist`、`build/` |
| **输出** | `.gitignore`、Git 仓库与首个提交 |
| **验收标准** | `git status` 干净；提交不含依赖、构建产物与本地工具配置 |

### Phase 2：CI 构建与发布流水线

| 项目 | 内容 |
|------|------|
| **输入** | 现有 `.github/workflows/build.yml`、`wails.json` |
| **输出** | 支持 `v*` 标签触发构建与 Release 的工作流；`wails.json` 声明 0.1.0 版本元数据 |
| **验收标准** | 工作流 YAML 合法；标签触发路径具备 `contents: write` 权限并上传 `quick-input.exe` |

### Phase 3：公开仓库创建与推送

| 项目 | 内容 |
|------|------|
| **输入** | 本地 `main` 分支 |
| **输出** | `Evan-1777/Quick_Input` 公开仓库，`main` 已推送 |
| **验收标准** | 远端仓库可见性为 public；本地与远端 `main` 指向同一提交 |

### Phase 4：0.1.0 版本发布与验收

| 项目 | 内容 |
|------|------|
| **输入** | 已推送的 `v0.1.0` 标签 |
| **输出** | 成功的 Actions 运行与 GitHub Release（含 `quick-input.exe`） |
| **验收标准** | 构建任务成功；Release 存在且资产可下载、大小非零 |

## 3. 架构决策

| 决策项 | 选择 | 理由 | 替代方案（为何不选） |
|--------|------|------|----------------------|
| 仓库名 | `Quick_Input` | 与项目目录同名，符合用户既有 PascalCase 仓库命名习惯 | `quick-input`（与目录不同名，违背「同名」要求） |
| 发布触发 | 推送 `v*` 标签触发构建 + Release | 流程可复现，发布动作由 CI 完成，无需本地上传二进制 | 本地 `gh release create` 手工上传（产物来源不可追溯，依赖本机下载） |
| Release 创建方式 | Runner 预装 `gh` CLI | 不引入第三方 Action 依赖 | `softprops/action-gh-release`（新增外部依赖，收益相同） |
| 前端产物 | 忽略 `frontend/dist` | CI 在 `go test` 与 `wails build` 前先构建前端；仓库不存二进制产物 | 提交 dist（噪声大，易与源码不一致） |
| 本地工具目录 | 忽略 `.codebuddy/`、`.agents/`、`.dsh/`、`.mcp.json` | 属本机 Agent 运行时配置，含本地路径与环境假设，不属于产品源码 | 提交（公开仓库混入本机配置） |
| 版本号位置 | `wails.json` 的 `info.productVersion` | Wails 构建时写入 Windows 可执行文件元数据 | 仅在标签中体现（二进制无法读出 0.1.0） |

## 4. 风险清单

| 风险 | 等级 | 缓解措施 |
|------|------|----------|
| `go.mod` 要求 Go 1.26，`setup-go` 按文件解析失败或版本不可用 | 🟡 中 | 首次运行若失败，改为显式 `go-version: '1.26'` 并复查日志 |
| Wails v2.16.0 在 `windows-latest` 首次构建失败 | 🟡 中 | 保留现有构建步骤，读取 Actions 日志定位；必要时固定 WebView2 无关的纯编译路径 |
| 新仓库默认 `GITHUB_TOKEN` 只读，Release 创建被拒 | 🟡 中 | 工作流显式声明 `permissions: contents: write` |
| 标签推送重复触发一次构建，耗时增加 | 🟢 低 | 可接受；标签构建同时承担发布职责，语义清晰 |
| 仓库公开后暴露本机配置或密钥 | 🔴 高 | 提交前审查 `git status`，忽略工具目录；`.mcp.json` 与 settings 不入库 |

## 5. Phase 依赖关系

```
Phase 1 ──→ Phase 3 ──→ Phase 4
Phase 2 ──→ Phase 3
```

Phase 1 与 Phase 2 的产出共同构成首次提交；Phase 3 推送后触发 Phase 4。

## 6. Subagent 协同规划

| 阶段 | 上下文依赖 | 并行收益 | 执行主体 | 决策理由 |
|------|-----------|---------|----------|----------|
| Phase 1-4 | 强（发布链路串行，依赖本机凭据与实时 CI 状态） | 无 | 主线 | 任务短、强串行且需即时读取 Actions/Release 状态，委托无收益 |
