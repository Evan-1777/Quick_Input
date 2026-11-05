# Tasks：手动排序与快捷面板分组操作

**关联 Plan**：`Plan.md` —— 手动排序与快捷面板分组操作 v1.1  
**状态**：DONE  
**完成日期**：2026-11-05  
**回归结论**：`gofmt -l .`、`go vet ./...`、`go test ./...`、`go tool mygo vet`、`GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...` 全部通过；面板 560×380、管理/设置 960×640 的 headless 用例通过  
**总计 Task**：11 个

---

## Phase 1：数据层顺序持久化

### TASK-001：为 snippets 增加 position 列并回填旧库

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：无
- **Description**：在 `internal/store/store.go` 的 snippets 建表语句中加入 `position INTEGER NOT NULL DEFAULT 0`，并在打开旧库时补列、按原显示顺序回填。
- **Details**：
  - `schema` 的 snippets 建表直接包含 position。
  - 新增 `ensureSnippetPosition(db *sql.DB) error`：用 `PRAGMA table_info(snippets)` 检查 position，读完并关闭 rows（`NewStore` 只保留一条连接，未关闭的游标会阻塞后续语句）；缺失时再开单事务执行 `ALTER TABLE snippets ADD COLUMN position INTEGER NOT NULL DEFAULT 0`，按 `updated_at DESC, id DESC` 读取 id 列表，逐条 `UPDATE snippets SET position = ? WHERE id = ?` 写入 0..n-1，最后提交。
  - `NewStore` 执行完全部建表语句后调用该函数；返回错误时关闭数据库并包装为「初始化表结构: %w」。
  - 不引入 schema 版本表，列存在性就是迁移判据。
  - 在 `internal/store/store_test.go` 增加 `TestSnippetPositionMigration`：在 `t.TempDir()` 生成文件库，用原始 `sql.Open("sqlite", path)` 建旧版 snippets 表（`id, title, content, group_id, updated_at`，无 position），插入三条不同 updated_at 的记录；关闭后用 `NewStore(path)` 打开，断言 `ListSnippets(SnippetFilter{GroupScope: "all"})` 的标题顺序与旧查询 `updated_at DESC, id DESC` 一致。
- **Acceptance Criteria**：
  - `go test ./internal/store` 通过。
  - 旧库迁移后显示顺序与升级前一致，position 依次为 0..n-1 且无重复。
  - 新库建表即含 position，迁移函数对已有列不产生改动。
  - `gofmt -l internal/store` 无输出。

### TASK-002：按 position 查询、新建置顶与 ReorderSnippets

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-001
- **Description**：把列表查询切换到 position 顺序，新建条目置顶，并提供事务化的拖动重排接口。
- **Details**：
  - `ListSnippets` 的排序改为 `ORDER BY s.position, s.id`；其余过滤与 tag 装配逻辑不变。
  - `SaveSnippet` 新建分支：在事务内先 `SELECT COALESCE(MIN(position), 0) - 1 FROM snippets` 得到 position，再 `INSERT INTO snippets (title, content, group_id, updated_at, position) VALUES (?, ?, ?, ?, ?)`；编辑分支的 UPDATE 不写 position，编辑不再移动条目。
  - 新增 `func (s *Store) ReorderSnippets(ids []int64, before *int64) error`：
    - `uniqueIDs` 去重；空切片直接返回 nil。
    - 单事务内读取全部 id（`ORDER BY position, id`），读完关闭 rows，剔除要移动的 id 得到其余顺序。
    - `before == nil` 或该 id 不在其余顺序中时插到末尾；否则插到该 id 之前；被移动项按入参顺序连续插入。
    - 按最终顺序写回 `position = 0..n-1`，提交。
    - 添加注释，说明每次落位全量重编号的 O(n) 写放大与「改用间隔编号」的升级路径。
  - 在 `internal/store/store_test.go` 增加 `TestSnippetOrderingAndReorder`：使用文件库验证新建 B 在 A 之前、编辑 A 后顺序仍为 B/A、`ReorderSnippets([A], &B)` 得 A/B、`ReorderSnippets([B], &A)` 得 B/A、`ReorderSnippets([A], nil)` 得 B/A、未知 before 落到末尾、空 ids 不改变顺序；关闭并重新 `NewStore` 后顺序保持。
  - 增加 `TestReorderSnippetsMultipleIDs`：三条记录顺序为 C/B/A 时，`ReorderSnippets([A, C], &B)` 得到 A/C/B，验证非相邻多 id 按入参顺序连续插入，其余项相对顺序不变。
- **Acceptance Criteria**：
  - 新条目位于列表最前；编辑任意条目后顺序不变。
  - 重排接口在 before 为空、首行、中间、末尾、未知 id、空 ids 六种情况下结果符合定义。
  - 非相邻多 id 用例通过；插入顺序与入参顺序一致。
  - 顺序跨进程重新打开保持。
  - `go test ./internal/store` 通过。

---

## Phase 2：管理列表拖动排序

### TASK-003：App 重排映射与管理列表接线

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-002
- **Description**：在 `internal/app/app.go` 增加重排映射，在 `internal/app/view_manager.go` 的管理列表上设置 `ListState.Reorder`。
- **Details**：
  - `App` 增加 `func (a *App) reorderSnippets(items []store.Snippet, rows []int, to int)`：
    - 遍历 rows，越界行直接返回；收集 id 与移动集合。
    - 从 `to` 开始向后找第一条不在移动集合中的可见行作为 before 锚点；找不到则 before 为 nil（末尾）。
    - 调用 `a.store.ReorderSnippets(ids, before)`；失败写 `a.lib.err` 并返回，成功调用 `a.reload()`。
  - `libraryMain` 在构建 List 之前设置 `a.lib.list.Reorder = func(rows []int, to int) { a.reorderSnippets(items, rows, to) }`。
  - 不改动 `a.lib.list.Selection` 与行内点击、双击逻辑。
- **Acceptance Criteria**：
  - `go build ./...` 通过。
  - 未过滤时拖动把行放到目标行之前；多选时 rows 内的条目整体移动。
  - 过滤视图下锚点取「目标索引之后第一条未移动的可见行」。
  - 写库失败只更新错误条、不改变当前顺序（由 TASK-004 的 `TestManagerReorderStoreError` 覆盖）。

### TASK-004：拖动排序视图测试

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-003
- **Description**：在 `internal/app/view_manager_test.go` 覆盖管理列表的拖动排序、过滤视图落位与拖动不触发编辑。
- **Details**：
  - 新增 `TestManagerReorderSnippets`：`newFixture` 后初始全局顺序为 `[Mixed, Ungrouped, Farewell, Greeting]`；`ShowManager(false)`，用 `ui.NewTester` 定位 Farewell 与 Mixed 的文本矩形；`Press` Farewell 中心，按 4 步 `Move` 到 Mixed 顶部（`mixed.Y+2`），`Release`；断言 `fx.app.editor == nil`，且 `fx.app.snippets` 的标题顺序为 `[Farewell, Mixed, Ungrouped, Greeting]`。
  - 新增 `TestManagerReorderWithinFilter`：设置 `fx.app.lib.filter = FilterState{Scope: scopeGroup, GroupID: &fx.groupA}`，初始可见 `[Mixed, Greeting]`；把 Greeting 拖到 Mixed 顶部；断言全局顺序为 `[Greeting, Mixed, Ungrouped, Farewell]`，并在重新构建测试器后断言可见顺序为 `[Greeting, Mixed]`。
  - 新增 `TestManagerReorderMultipleSelection`：初始 `[Mixed, Ungrouped, Farewell, Greeting]`；`tt.Click("Mixed")` 后 `tt.ClickWith(ui.Cmd, "Greeting")` 选中两项（选中后批量栏出现，需重新 `Find` 取得 Mixed 与 Farewell 的新矩形）；把 Mixed 拖到 Farewell 顶部；断言全局顺序为 `[Ungrouped, Mixed, Greeting, Farewell]`，且 `fx.app.lib.chosen.Len() == 2`。该用例覆盖非相邻多选、移动集合跳过锚点与入参顺序。
  - 新增 `TestManagerReorderStoreError`：`newFixture` 后关闭 `fx.app.store`，调用 `fx.app.reorderSnippets(fx.app.snippets, []int{0}, 2)`；断言 `fx.app.lib.err` 非空，且 `fx.app.snippets` 的标题顺序不变（关闭的 store 由夹具的 Cleanup 再次关闭，忽略重复关闭的返回值）。
  - 断言拖动后未打开编辑侧边页，避免与双击路径混淆。
- **Acceptance Criteria**：
  - 四个用例通过，不依赖像素截图。
  - 覆盖写库失败分支：错误写入错误条，内存顺序不变。
  - 断言覆盖持久化后的内存快照顺序，而不是仅断言当前帧的行索引。
  - 既有管理列表用例不受影响。

---

## Phase 3：快捷面板分组快捷键与未分组

### TASK-005：面板未分组显示偏好与持久化

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：无
- **Description**：在 `internal/store/store.go` 增加设置键，在 `internal/app/app.go` 增加 `showUngrouped` 字段、启动读取与保存方法。
- **Details**：
  - 新增 `const SettingOverlayShowUngrouped = "overlay_show_ungrouped"`，值使用 `"1"` / `"0"`，键缺失视为关闭。
  - `App` 增加 `showUngrouped bool`；`New` 读取该设置，读取失败或值不是 `"1"` 时保持 false。
  - 新增 `func (a *App) SetShowUngrouped(show bool) error`：先 `SetSetting`，失败返回 `保存设置失败: %w`；成功后更新字段；关闭时若 `a.overlay.filter.Scope == scopeUngrouped`，直接 `a.overlay.filter = FilterState{Scope: scopeAll}`（与 `resetOverlay` 一致，同时清空关键字与标签），避免面板停留在不可见的未分组筛选上。
  - 新增 `internal/app/app_test.go`：默认 false；`SetShowUngrouped(true)` 后字段为 true 且用同一 store 重新 `New` 仍为 true；开启时把面板筛选置为未分组，关闭后筛选回到全部且关键字与标签清空；关闭的 store 上调用返回错误且字段不变。
- **Acceptance Criteria**：
  - `go test ./internal/app` 通过。
  - 设置跨应用实例生效；写库失败不改变内存状态。
  - 关闭开关时不会把面板留在「未分组」筛选上。

### TASK-006：过滤纯函数与方括号解析

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：无
- **Description**：在 `internal/app/filter.go` 增加面板专用的纯函数，在 `internal/app/filter_test.go` 覆盖。
- **Details**：
  - `func groupedOnly(items []store.Snippet) []store.Snippet`：保持原顺序，只保留 `GroupID != nil` 的条目；入参为空时返回空切片。
  - `func groupSteps(groups []store.Group, ungrouped bool) []FilterState`：首项 `{Scope: scopeAll}`；ungrouped 为真时追加 `{Scope: scopeUngrouped}`；随后按分组顺序追加 `{Scope: scopeGroup, GroupID: &id}`。
  - `func cycleGroup(f FilterState, steps []FilterState, step int) FilterState`：空 Scope 先归一为 `scopeAll`；steps 为空时直接返回归一化后的 f；把 step 钳制到 -1/0/+1（一次按键最多翻一步，多枚方括号或卡顿期间的按键累积不会越界）；按 Scope 与 GroupID（group 作用域比较 GroupID 指向的值）找当前位置，未命中按 0 处理；用 `nextIndex` 环绕取目标；只替换 Scope 与 GroupID，保留 Keyword 与 TagIDs。
  - `func splitBrackets(keyword string) (string, int)`：移除全部 `[` 与 `]`，`[` 计 -1、`]` 计 +1；没有方括号时原样返回且步数为 0。
  - 在 `filter_test.go` 增加表驱动用例：保留未分组过滤、分组步进顺序、开启后含未分组、双向环绕、关键字与标签保留、空步骤列表、方括号计数与剥离、无方括号原样返回；`cycleGroup` 在 `step = ±4`、steps 长度为 3 时钳制为一步且不越界，steps 长度为 1/2 时同样不 panic。
- **Acceptance Criteria**：
  - `go test ./internal/app -run 'TestGroupedOnly|TestGroupSteps|TestCycleGroup|TestSplitBrackets'` 通过。
  - 对空步骤列表不 panic（返回入参的归一化结果）。
  - 对任意 `|step| > len(steps)` 的输入，结果等于方向一致的一步。
  - 不改变 `filterSnippets` 与管理窗口的既有语义。

### TASK-007：面板方括号切换分组与未分组隐藏

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-005, TASK-006
- **Description**：在 `internal/app/view_overlay.go` 接入方括号循环与未分组过滤，并修正面板用例。
- **Details**：
  - `OverlayView` 在计算 items 之前调用 `a.consumeGroupBrackets()`；返回 true 时把 `a.overlay.chosen` 归零并 `ScrollTo(0, ui.Start)`。
  - `consumeGroupBrackets`：用 `splitBrackets` 处理 `a.overlay.filter.Keyword`；字符串未变化直接返回 false；写入剥离后的关键字；步数非 0 时调用 `cycleGroup`，步骤来自 `groupSteps(a.groups, a.showUngrouped)`；返回 true。在函数上方注释原因：框架的 `OnTextKey` 不接收可打印按键，方括号先写入绑定值，因此在这里剥离并作为分组指令；方括号不能同时作为搜索字符。
  - items 计算改为先 `a.filteredSnippets(a.overlay.filter)`，再在 `!a.showUngrouped` 时套 `groupedOnly`。
  - `overlayFilters` 中「未分组」行仅在 `a.showUngrouped` 时渲染。
  - `groupRow` 的 `Clicked` 分支、标签 `Toggle` 的 `Changed` 分支与空态「清除筛选」分支在更新筛选后调用 `a.overlay.search.Focus()`。搜索框是唯一按键入口；点击分组会清空焦点、点击标签会把焦点交给 Toggle，回焦后上下键与方括号都恢复可用。
  - 空态：保持有筛选时的「没有匹配的预设文本」；无筛选且 items 为空、`a.snippets` 非空、`!a.showUngrouped` 时显示「暂无分组预设文本」与 `textButton("打开管理窗口")`（调用 `a.ShowManager(false)`）；其余情况保持现有的「暂无预设文本」与「新建预设文本」。
  - 不新增根级方括号快捷键，保持「面板键盘只有一条路径」。
  - 修正 `view_overlay_test.go` 现有断言：初始行不再包含 Ungrouped；`TestOverlayFilterAndKeyboard` 清除关键字后按一次 Down 即到 Farewell（列表变为 Mixed/Farewell/Greeting），Enter 仍键入 Goodbye world。
  - 新增 `TestOverlayGroupBracketsAndUngrouped`：
    - 默认初始不含 Ungrouped，且不含「未分组」筛选项。
    - 用 `tt.TypeKey(0, ui.KeyBracketRight, "]")` 依次切到工作、生活、全部；用 `tt.TypeKey(0, ui.KeyBracketLeft, "[")` 从全部反向回到生活；断言 Scope/GroupID 与可见标题。`ui.KeyBracketLeft` / `ui.KeyBracketRight` 已在 `ui/keys.go` 定义，直接使用。
    - 输入 `hello` 后按 `]`，关键字保留为 `hello`、不含方括号、无「没有匹配的预设文本」闪现。
    - 点击「工作」分组后再输入 `hello`：关键字应进入搜索框；随后 `tt.TypeKey(0, ui.KeyBracketRight, "]")` 切到下一组且关键字保留。点击标签「urgent」后再输入一个字符，同样应进入搜索框；输入无匹配关键字后点击空态「清除筛选」，再输入字符也应进入搜索框。
    - 一次性 `tt.Type("[[[[")` 在 steps 长度为 3 时只翻一步且不 panic；`tt.Type("]]")` 向后一步。
    - `SetShowUngrouped(true)` 后 ShowOverlay，Ungrouped 与「未分组」出现；`]` 首次切到未分组，只显示 Ungrouped。
    - 把所有条目移为未分组并关闭开关后，空态显示「暂无分组预设文本」与「打开管理窗口」。
- **Acceptance Criteria**：
  - 上述用例通过；原有面板用例按新列表顺序修正后通过。
  - 面板首帧不出现方括号导致的空结果。
  - 点击分组、标签或空态「清除筛选」后焦点回到搜索框，随后输入字符与方括号均生效。
  - 单帧多枚方括号只翻一步且不越界。
  - 关键字、标签筛选在循环切换后保持不变。
  - 开关关闭时未分组既不在列表、也不在筛选项与循环中。

### TASK-008：设置页「显示未分组」开关

- **Status**：DONE
- **Executor**：主线
- **Priority**：P1
- **Depends On**：TASK-005
- **Description**：在 `internal/app/view_settings.go` 的快捷键卡片之后新增「快捷面板」分区与开关行，并在管理视图测试中覆盖。
- **Details**：
  - 新增分区标题「快捷面板」（12、Bold，与「唤出快捷键」同级）。
  - 卡片沿用 `radiusCard`、`p.SurfaceLow`、`p.OutlineVariant` 样式；单行高 48、`Padding(0, 16)`。
  - 行内：`ui.Text(c, "显示未分组")` 12 号字 `Grow(1)`；右侧 `ui.Switch(c, &on).Label("显示未分组开关")`，其中 `on := a.showUngrouped`。可见文字与开关的可访问名称分开，测试据此定位开关。
  - 删除快捷键卡片内部现存的 `a.lib.err` 文本，改为在两张卡片之后统一渲染页面级错误行（`p.Error`、11 号、`Wrap`）。页面级错误行同时承载快捷键与开关的错误，避免开关错误显示在快捷键卡片里造成误读。
  - 开关 `Changed()` 时调用 `a.SetShowUngrouped(on)`；失败写 `a.lib.err`，成功清空错误；失败时行内开关保持原状态（下一帧从字段重建）。
  - 在 `internal/app/view_manager_test.go` 增加 `TestSettingsShowUngrouped`：进入设置页，断言「显示未分组」可见；点击「显示未分组开关」一次后 `fx.app.showUngrouped` 为 true；再次点击恢复 false。
- **Acceptance Criteria**：
  - 设置页在 960×640 下显示完整，不与快捷键卡片重叠。
  - 错误行位于设置页页面级（两张卡片之后），不在快捷键卡片内部。
  - 开关状态跨页面切换与重启保持（由 TASK-005 的持久化保证）。
  - 保存失败不会静默改变 UI 状态。
  - `go test ./internal/app` 通过。

---

## Phase 4：验证、文档与归档

### TASK-009：全量回归与界面走查

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-004, TASK-007, TASK-008
- **Description**：执行构建、测试、静态检查、Windows 交叉编译，并用 headless 测试器走查面板与设置页。
- **Details**：
  - `gofmt -l .`（应无输出）、`go vet ./...`、`go test ./...`、`go tool mygo vet`、`GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...`。
  - 用 `ui.NewTester` 分别以 560×380 与 960×640 构建面板、管理列表、设置页，检查开关行、空态、拖动后的行序与文字不溢出；必要时用 `Tester.Image()` 目视核对，不落盘临时 PNG。
  - 检查无遗留 scratch/探针测试与临时文件。
- **Acceptance Criteria**：
  - 所有命令退出码为 0；`gofmt -l .` 无输出。
  - 面板默认隐藏未分组、开关后可显示；方括号切换与拖动排序均可复现。
  - 工作区没有新增临时产物。

### TASK-010：同步 Project.md

- **Status**：DONE
- **Executor**：主线
- **Priority**：P1
- **Depends On**：TASK-009
- **Description**：按实际实现更新 `.docs/Project.md`，删除已过期的排序与过滤描述。
- **Details**：
  - 概述：阶段行补充本次交付（手动排序、面板分组快捷键、未分组默认隐藏）。
  - 数据流：说明列表顺序由 snippets.position 决定，新建置顶、编辑不改位置。
  - 约定：新增排序规则；过滤语义补充「面板默认隐藏未分组，管理窗口不变」；快捷键补充面板内 `[` / `]` 的方向与循环范围；设置数据流补充 `overlay_show_ungrouped`。
  - 约束与决策：新增 2026-11-05 条目，记录 position 全量重编号、方括号走绑定值剥离、未分组开关默认关闭、排序仅作用于预设文本。
  - 已知限制：输入法全角方括号不触发切换；方括号不能作为搜索字符，粘贴含方括号的文本会被消费，净方向非 0 时还会触发翻组。
  - 术语：新增「手动顺序」= 由 position 升序决定的预设文本展示顺序；不新增不存在的模块或目录。
- **Acceptance Criteria**：
  - Project.md 不再声称列表按更新时间排序。
  - schema、设置键、快捷键、默认行为与代码一致。
  - 文档无重复或前后矛盾的段落。

### TASK-011：归档与提交

- **Status**：DONE
- **Executor**：主线
- **Priority**：P0
- **Depends On**：TASK-010
- **Description**：回填计划状态，归档根目录文档并在审查后提交。
- **Details**：
  - 将 `Plan.md` 与 `Tasks.md` 的状态改为 `DONE`，补充完成日期与 TASK-009 的回归测试结论。
  - 归档到 `.docs/11-05-v2/`（当前 `.docs` 下同日已有 v1），确认目录内同时含 Plan.md 与 Tasks.md，根目录无残留。
  - 执行 `git status`、`git diff`、`git log --oneline -8`，审查改动与提交风格。
  - 提交信息使用中文 Conventional Commits，例如 `feat: 支持预设文本拖动排序与快捷面板分组切换`；不推送远端，不使用 `--amend`。
- **Acceptance Criteria**：
  - 归档目录含两个状态为 DONE 的文档，根目录无 Plan.md / Tasks.md。
  - 提交成功，提交信息符合项目约定。
  - `git status` 干净（忽略已忽略目录）。
