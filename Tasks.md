# Tasks：面板分组切换修复、分组置顶默认与分组拖动排序

**关联 Plan**：`Plan.md` —— v1.1  
**总计 Task**：8 个  
**执行顺序**：TASK-001 → 002 → 003 → 004 → 005 → 006 → 007 → 008（全部主线，严格串行）

---

## Phase 1：存储层分组手动顺序

### TASK-001：通用位置迁移并为分组表加 position 列

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends**：无
- **Description**：在 `internal/store/store.go` 中把 `ensureSnippetPosition(db)` 重构为 `ensurePosition(db *sql.DB, table, backfillOrder string) error`，并对分组表复用。
- **Details**：
  - `schema` 中 `groups` 建表语句追加 `position INTEGER NOT NULL DEFAULT 0`
  - `ensurePosition` 以 `PRAGMA table_info(<table>)` 检测列存在性；缺列时单事务内 `ALTER TABLE` 并按 `backfillOrder` 回填 0..n-1；`table` 与 `backfillOrder` 只接收包内常量
  - `NewStore` 依次调用 `ensurePosition(db, "snippets", "updated_at DESC, id DESC")` 与 `ensurePosition(db, "groups", "name COLLATE NOCASE, id")`
  - `ListGroups` 改为 `ORDER BY position, id`，函数注释改为按手动顺序
  - `SaveGroup` 新建分支写入 `position = COALESCE(MAX(position), -1) + 1`（追加末尾）；重命名分支不触碰 position
- **Acceptance Criteria**：
  - `TestSnippetPositionMigration` 原样通过
  - 新增 `TestGroupPositionMigration`：以无 position 列的旧 `groups` 表建库（参照预设文本迁移测试的构造方式），插入名称乱序的分组，`NewStore` 后 `ListGroups` 按名称不区分大小写排序，position 为 0..n-1；再次 `NewStore` 同一库不改变顺序
  - 新增断言：依次新建 `B`、`A` 两个分组后 `ListGroups` 顺序为 `B, A`

### TASK-002：通用重排并新增 ReorderGroups

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends**：TASK-001
- **Description**：把 `ReorderSnippets` 的主体提取为私有 `func (s *Store) reorder(table string, ids []int64, before *int64) error`，`ReorderSnippets` 与新增 `ReorderGroups` 为一行包装。
- **Details**：
  - `reorder` 内的查询与更新语句以 `table` 拼接，`table` 仅来自两个包装函数的常量
  - 保留原注释中全量重编号的取舍说明，移至 `reorder`
  - `ReorderGroups(ids []int64, before *int64) error` 注释说明语义同 `ReorderSnippets`
- **Acceptance Criteria**：
  - `TestSnippetOrderingAndReorder`、`TestReorderSnippetsMultipleIDs` 原样通过
  - 新增 `TestReorderGroups`：三个分组 `X, Y, Z`，`ReorderGroups([Z], &X)` 后 `ListGroups` 为 `Z, X, Y`；`ReorderGroups([Z], nil)` 后为 `X, Y, Z`

---

## Phase 2：面板分组切换修复与排列

### TASK-003：全角方括号等效翻组

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends**：无
- **Description**：在 `internal/app/filter.go` 的 `splitBrackets` 中把 `【` 映射为向前、`】` 映射为向后，与 `[` `]` 一致剥离。
- **Details**：
  - `strings.ContainsAny` 的字符集与 `strings.Map` 的分支同步加入 `【` `】`
  - 更新函数注释：半角与全角方括号均为翻组指令，均不能作为搜索字符
  - `view_overlay.go` 中 `consumeGroupBrackets` 注释同步提及全角
- **Acceptance Criteria**：
  - `TestSplitBrackets` 新增用例：`"hel】lo"` → `"hello", 1`；`"x【"` → `"x", -1`；`"【a]"` → `"a", 0`
  - `view_overlay_test.go` 中以 `tt.TypeKey(0, ui.KeyBracketRight, "】")` 翻组：作用域前进一步，`overlay.filter.Keyword` 不含 `】`

### TASK-004：分组栏顺序与默认作用域

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends**：TASK-001, TASK-003
- **Description**：面板分组栏与 `[` `]` 循环统一为「各分组 → 全部 → 未分组（开关开启时）」，唤出默认取第一步。
- **Details**：
  - `filter.go`：`groupSteps` 先追加各分组，再追加 `scopeAll`，开关开启时最后追加 `scopeUngrouped`；更新注释
  - `filter.go`：新增 `sameScope(a, b FilterState) bool`（Scope 相同，且 `scopeGroup` 时 GroupID 均非空且相等）；`cycleGroup` 的当前步查找与 `view_overlay.go` 中 `groupRow` 的 `active` 判定改用它
  - `app.go`：新增 `func (a *App) defaultOverlayScope() FilterState { return groupSteps(a.groups, a.showUngrouped)[0] }`；`resetOverlay` 以它赋值 `overlay.filter`
  - `view_overlay.go` `overlayFilters`：先渲染各分组行，再渲染「全部」，开关开启时最后渲染「未分组」
  - `view_overlay.go` `overlaySearch` 的 Esc 第二阶段：以 `base := a.defaultOverlayScope()` 取得基准，条件改为 `!sameScope(filter, base) || len(TagIDs) > 0`，动作为恢复基准作用域并清空标签；否则隐藏面板
  - `SetShowUngrouped` 关闭时的重置改为 `defaultOverlayScope()`
  - `overlayEmpty` 的「清除筛选」保持重置为 `FilterState{Scope: scopeAll}`，`filtered` 判定不变
- **Acceptance Criteria**：
  - `TestGroupSteps` 改为断言：隐藏未分组时 `[group10, group20, all]`；显示时 `[group10, group20, all, ungrouped]`
  - `TestCycleGroup` 按新步骤顺序改写，覆盖前进、后退、回绕、钳制一步与保留关键字/标签
  - `TestOverlayGroupBracketsAndUngrouped` 改写：唤出后作用域为 `groupA`（工作）且只显示 Greeting、Mixed；`]` → `groupB`；`]` → 全部；`]` → `groupA`；`[` → 全部；开启未分组后先点击「全部」，再 `]` → 未分组
  - 新增断言：分组栏中「工作」「生活」的纵坐标小于「全部」（`tt.Find` 比较 `Y`）
  - 新增断言：唤出后无关键字直接 Esc，`overlay.visible == false`
  - `TestOverlayFilterAndKeyboard`、`TestOverlayWrapUpAndTagIntersection`、`TestOverlayEmptyState` 按默认 `groupA` 调整期望（需全量行时先显式设置 `overlay.filter` 为全部）并通过
  - 无分组的库唤出面板时作用域为 `scopeAll`（以空 store 构造 App，不经过 `newFixture`）

---

## Phase 3：管理窗口分组拖动排序

### TASK-005：应用层重排锚点复用与 reorderGroups

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends**：TASK-002
- **Description**：在 `internal/app/app.go` 中把 `reorderSnippets` 内“行号 → 移动 ID 与锚点 ID”的计算提取为纯函数，新增 `reorderGroups`。
- **Details**：
  - 新增 `func reorderTarget(ids []int64, rows []int, to int) (moving []int64, before *int64, ok bool)`：`rows` 越界返回 `ok=false`；锚点为 `to` 起第一个非移动行，无则 `nil`
  - `reorderSnippets` 改为收集 `items` 的 ID 后调用 `reorderTarget` 与 `store.ReorderSnippets`，外部行为不变
  - 新增 `func (a *App) reorderGroups(rows []int, to int)`：基于 `a.groups` 调用 `reorderTarget` 与 `store.ReorderGroups`，失败写 `a.lib.err` 且不刷新，成功 `a.reload()`
- **Acceptance Criteria**：
  - `TestManagerReorder*` 四个既有测试原样通过
  - 新增 `TestReorderGroupsStoreError`：关闭 store 后调用 `reorderGroups([]int{1}, 0)`，`lib.err` 非空且 `a.groups` 顺序不变

### TASK-006：侧栏分组区改为可拖动列表

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends**：TASK-005
- **Description**：在 `internal/app/view_manager.go` 的 `sidebar` 中把分组循环改为 `ui.List`，接入 `ListState.Reorder`。
- **Details**：
  - `app.go` 的 `libraryState` 新增 `groupList ui.ListState`
  - `sidebar` 中「全部」「未分组」两行保持原位与原实现；其后以 `ui.List(c, &a.lib.groupList, len(a.groups), func(i int){...})` 渲染分组行，设置 `Key`（分组 ID）、`Label`（分组名）、`Reorder = a.reorderGroups`，不设 `Selection` / `Selected`
  - List 为滚动容器：高度取 `len(a.groups) × 33`（行高 30 + 间距 3），`Gap(3)`，宽度撑满侧栏（沿用父级拉伸，必要时显式 `Width`）；`len(a.groups) == 0` 时不构建 List，避免空容器占位
  - `sidebarGroup` 的分组行补 `.Key(group.ID)`；点击筛选、悬停重命名/删除、重命名输入框行为不变
- **Acceptance Criteria**：
  - 新增 `TestManagerReorderGroups`：用既有 `dragRow` 把「生活」拖到「工作」上方，`a.groups` 顺序为 `生活, 工作`；新建 `ui.NewTester(fx.app.OverlayView, ...)` 并 `ShowOverlay` 后默认作用域为 `groupB`
  - `TestManagerReorderGroups` 中分组名会与预设文本行的分组 chip 同名；测试优先使用唯一分组名或按侧栏区域定位，避免 `tt.Find` 命中错误节点
  - 拖动后 `fx.app.editor == nil` 且 `lib.filter` 未被改变
  - 新增 `TestManagerSidebarGroupsMany`（或并入现有测试）：构造足够多的分组（如 10 个）时末行可见、侧栏可滚动、List 内部无独立滚动条
  - 拖动后从默认分组按 `]` 切到下一分组，验证 `groupSteps` 顺序跟随新的分组顺序
  - `TestManagerSidebarCRUD`、`TestManagerFiltersAndSettings` 原样通过
  - 点击「生活」分组后 `lib.filter` 为 `scopeGroup` + `groupB`（新增断言）

---

## Phase 4：回归与文档

### TASK-007：全量本地验证与 Windows 实机输入法验收

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P0
- **Depends**：TASK-001 ~ TASK-006
- **Description**：在项目根执行本地验证命令，并完成 Windows 实机中文输入法验收记录。
- **Details**：
  - `go test ./...`
  - `go vet ./...`
  - `go tool mygo vet`
  - `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...`（不保留产物）
  - Windows 11 实机 + 微软拼音/搜狗：输入 `【`/`】`，确认分组切换且关键字无残留（人工，由用户执行；不计入本地四条命令的退出码门槛）
- **Acceptance Criteria**：
  - 四条命令全部退出码 0
  - Windows 实机输入法人工验收通过并记录结论（由用户执行）
  - 结论回填至 `Plan.md` 的「回归测试结论」

### TASK-008：同步 Project.md

- **Status**：PENDING
- **Executor**：主线
- **Priority**：P1
- **Depends**：TASK-007
- **Description**：按 `Project.md` 维护表同步本次变更。
- **Details**：
  - 「概述 · 阶段」补充本次三项变更
  - 「结构」中 `store.go` 描述补充分组顺序；「数据流」补充分组按 `groups.position` 升序、新建追加末尾、侧栏拖动后全量重编号
  - 「约定 · 排序」改为：预设文本与分组按 `position` 升序、同值按 `id`；标签按名称排序，不参与拖动
  - 「约束与决策」新增 2026-11-06 条目：全角方括号等效翻组且不再作为搜索字符；面板循环顺序「各分组 → 全部 → 未分组」与默认第一步；Esc 第二阶段恢复默认作用域；分组拖动仅在管理窗口；管理侧栏「全部 / 未分组」位置保持现状（需求 #2 范围假设）
  - 改写 2026-11-05「面板分组循环」与「已知限制」条目中与全角方括号、循环范围冲突的表述，删除过期内容
- **Acceptance Criteria**：
  - `Project.md` 中不再出现“全角方括号不触发面板切换”与“分组与标签继续按名称排序”等过期表述
  - 文档描述的函数、字段与代码一致
