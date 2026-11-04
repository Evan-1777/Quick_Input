# Plan：Win11 轻量级快捷文本键入工具

**状态**：DONE  
**日期**：2026-11-04  
**版本**：v1.1  
**回归测试结论**：2026-11-04 通过 —— 后端 `go test ./internal/...` 共 12 项用例全绿、`go vet ./internal/...` 零错误；前端 `npm test` 7 项断言通过、`tsc --noEmit` 与 `vite build` 通过；`GOOS=windows GOARCH=amd64 go vet ./...` 与 Windows 链接构建零错误。  

---

## 1. 背景与目标

本项目构建运行于 Windows 11 的个人桌面效率工具，用于在常规权限活动窗口中通过全局快捷键唤出悬浮面板、按分组与标签检索预设文本，并将选中内容直接键入原输入焦点位置；同时提供系统托盘常驻与主管理界面，支持快速新建及多选批量归类。代码在 WSL Linux 环境下开发与执行自动化验证，Windows 11 原生产物由云端持续集成流水线构建。

系统围绕三条核心业务链路与四层正交结构展开：

<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; border: 1px solid #d4d4d8; border-radius: 6px; padding: 16px; background: #fafafa; color: #18181b; margin: 12px 0;">
  <div style="font-size: 13px; font-weight: 600; color: #3f3f46; margin-bottom: 12px; border-bottom: 1px solid #e4e4e7; padding-bottom: 6px;">系统分层架构与三条核心交互链路</div>
  <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin-bottom: 14px;">
    <div style="border: 1px solid #d4d4d8; border-radius: 4px; background: #ffffff; padding: 10px;">
      <div style="font-size: 12px; font-weight: 600; color: #18181b; margin-bottom: 6px;">链路一：快捷唤出与键入</div>
      <div style="font-size: 11px; color: #52525b; line-height: 1.5;">
        1. 从隐藏态唤出悬浮窗前，快照记录非本进程的前景窗口句柄<br/>
        2. 获取输入光标物理像素坐标（失败回退鼠标坐标），按窗口 DPI 换算尺寸并裁剪至显示器工作区<br/>
        3. 激活悬浮窗并自动聚焦搜索框，支持键盘输入过滤、方向键导航及鼠标点击右侧分组/标签筛选（失焦自动收起）<br/>
        4. 选中文本后立即隐藏悬浮窗，将原目标窗口恢复为前景焦点<br/>
        5. 等待物理修饰键释放，执行剪贴板纯文本备份与写入、发送合成粘贴指令、按剪贴板序列号安全还原纯文本
      </div>
    </div>
    <div style="border: 1px solid #d4d4d8; border-radius: 4px; background: #ffffff; padding: 10px;">
      <div style="font-size: 12px; font-weight: 600; color: #18181b; margin-bottom: 6px;">链路二：系统托盘与快速新建</div>
      <div style="font-size: 11px; color: #52525b; line-height: 1.5;">
        1. 单实例锁启动后无边框隐藏驻留，挂载系统托盘图标<br/>
        2. 窗口关闭动作拦截为隐藏至后台，不退出进程<br/>
        3. 右键菜单提供「新建预设文本」「打开管理窗口」「退出」<br/>
        4. 切换管理模式时按 DPI 居中展示带自定义拖拽标题栏的管理视图，保存后更新前端共享数据集，悬浮窗即时生效
      </div>
    </div>
    <div style="border: 1px solid #d4d4d8; border-radius: 4px; background: #ffffff; padding: 10px;">
      <div style="font-size: 12px; font-weight: 600; color: #18181b; margin-bottom: 6px;">链路三：批量管理与组织</div>
      <div style="font-size: 11px; color: #52525b; line-height: 1.5;">
        1. 主管理视图展示全量预设文本、分组与标签，包含自定义窗口拖拽与最小化/关闭控制栏<br/>
        2. 列表支持复选框多选、全选当前筛选结果与清空选择<br/>
        3. 批量操作栏支持一键移入指定分组、批量添加标签、批量移除标签与二次确认批量删除<br/>
        4. 独立管理分组与标签的增删改查，删除前二次确认并在事务内清理关联引用
      </div>
    </div>
  </div>
  <div style="display: flex; flex-direction: column; gap: 6px; font-size: 11px;">
    <div style="display: flex; align-items: center; border: 1px solid #a1a1aa; border-radius: 4px; background: #f4f4f5; padding: 6px 10px;">
      <span style="font-weight: 600; width: 140px; color: #18181b;">前端双视图层</span>
      <span style="color: #52525b;">单实例无边框 WebView2 容器内承载「快捷悬浮窗视图」与「带自定义标题栏的主管理视图」，基于单一内存数据集执行即时过滤</span>
    </div>
    <div style="display: flex; align-items: center; border: 1px solid #a1a1aa; border-radius: 4px; background: #f4f4f5; padding: 6px 10px;">
      <span style="font-weight: 600; width: 140px; color: #18181b;">应用编排服务层</span>
      <span style="color: #52525b;">维护运行时上下文，暴露前后端绑定接口，按主键从存储层读取最新文本触发键入，统一派发窗口模式切换事件</span>
    </div>
    <div style="display: flex; align-items: center; border: 1px solid #a1a1aa; border-radius: 4px; background: #f4f4f5; padding: 6px 10px;">
      <span style="font-weight: 600; width: 140px; color: #18181b;">原生平台适配层</span>
      <span style="color: #52525b;">通过编译约束隔离 Windows 原生系统调用（热键、光标与 DPI 换算、进程窗口枚举、焦点快照恢复、剪贴板序列号校验粘贴、托盘）与 Linux 测试桩</span>
    </div>
    <div style="display: flex; align-items: center; border: 1px solid #a1a1aa; border-radius: 4px; background: #f4f4f5; padding: 6px 10px;">
      <span style="font-weight: 600; width: 140px; color: #18181b;">本地持久化存储层</span>
      <span style="color: #52525b;">纯 Go SQLite 驱动维护单连接外键约束数据库，包含预设文本表、唯一名称分组表、唯一名称标签表及多对多关联表</span>
    </div>
  </div>
</div>

核心架构目标量化指标：
- **唤醒响应**：单实例窗口启动即预加载隐藏，热键触发时仅执行物理像素坐标计算、尺寸位置调整与视图状态切换，无渲染进程冷启动开销。
- **焦点衔接与键盘检索**：唤出前锁定记录原活动窗口句柄，唤出后正常激活悬浮窗以支持键盘搜索与方向键选择；即便用户在悬浮窗内点击分组、标签或输入搜索词，选中后依托当前前台权限即可将焦点归还原目标窗口并完成粘贴。
- **输入兼容**：等待物理修饰键抬起后，采用剪贴板纯文本备份、写入目标文本、发送合成粘贴按键、校验剪贴板序列号未变后恢复原纯文本的机制，绕过中文输入法拦截并消除长文本逐字延迟。

---

## 2. 阶段划分

### Phase 1：项目初始化与本地数据存储层

| 项目 | 内容 |
|------|------|
| **输入** | 业务领域模型需求（预设文本、单选归属分组、多维关联标签、批量管理规则） |
| **输出** | 前后端工程配置、SQLite 存储模块（含父目录自动创建、单连接与外键参数初始化、名称大小写不敏感唯一约束、预设文本与标签关联全量替换事务、批量分组迁移与批量标签增删事务）及其单元测试 |
| **验收标准** | 在 WSL Linux 下通过存储层全部单元测试；首次启动目录创建、名称唯一性冲突校验、分组单选归属、标签多选交集过滤、批量移入分组、批量增删标签以及分组/标签删除后的关联清理逻辑全部验证通过 |

### Phase 2：原生平台交互层与应用编排服务层

| 项目 | 内容 |
|------|------|
| **输入** | Phase 1 的数据存储模块与三条核心链路的系统级交互契约 |
| **输出** | 跨平台抽象接口、Windows 原生实现（全局热键与托盘消息循环、按进程 ID 枚举主窗口句柄、输入光标与鼠标物理坐标获取及 DPI 尺寸换算、显示器工作区边界裁剪、原窗口句柄快照与焦点恢复、修饰键释放检测与剪贴板序列号保护粘贴）、Linux 编译测试桩，以及持有运行时上下文的应用编排服务与单实例启动入口 |
| **验收标准** | 在 Linux 下通过平台几何裁剪与服务层内部包单元测试及静态检查；在开启 Windows 目标平台交叉静态检查时零错误通过 |

### Phase 3：单实例双视图前端界面实现

| 项目 | 内容 |
|------|------|
| **输入** | Phase 2 暴露的应用服务接口与窗口模式切换事件契约 |
| **输出** | 锁定版本的前端工程配置、纯函数过滤与键盘导航逻辑模块及单元测试、后端接口调用封装、顶层双模式视图容器、快捷悬浮窗组件（左侧搜索框与键盘上下导航列表、右侧分组单选与标签多选过滤区、失焦自动收起）、带自定义拖拽标题栏的主管理窗口组件（快速新建/编辑表单、列表多选复选框、批量移入分组与批量增删标签操作栏、带二次确认的删除与分组/标签管理面板） |
| **验收标准** | 前端过滤与键盘导航纯函数单元测试通过，TypeScript 类型检查与生产打包通过，无编译错误 |

### Phase 4：自动化验证、云端构建配置与项目文档建立

| 项目 | 内容 |
|------|------|
| **输入** | Phase 1 至 Phase 3 完成的前后端全部实现 |
| **输出** | Linux 内部包单元测试与静态检查、前端单元测试与生产构建、覆盖入口包的 Windows 交叉静态检查、GitHub Actions Windows 原生产物自动构建工作流配置，以及与代码库一致的项目基线文档 |
| **验收标准** | 后端内部包测试、前端测试与构建、Windows 全量包交叉静态检查全部零错误通过；云端构建工作流配置完整且包含 Windows 验收核对项；临时构建缓存清理完毕；项目文档完整反映实际架构与决策 |

---

## 3. 架构决策

<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; border: 1px solid #d4d4d8; border-radius: 6px; overflow: hidden; margin: 12px 0;">
  <table style="width: 100%; border-collapse: collapse; font-size: 12px; text-align: left;">
    <thead>
      <tr style="background: #f4f4f5; color: #18181b; border-bottom: 1px solid #d4d4d8;">
        <th style="padding: 8px 10px; width: 16%;">决策项</th>
        <th style="padding: 8px 10px; width: 22%;">选定方案</th>
        <th style="padding: 8px 10px; width: 34%;">选型依据</th>
        <th style="padding: 8px 10px; width: 28%;">替代方案（淘汰原因）</th>
      </tr>
    </thead>
    <tbody style="color: #3f3f46;">
      <tr style="border-bottom: 1px solid #e4e4e7; background: #ffffff;">
        <td style="padding: 8px 10px; font-weight: 600; color: #18181b;">窗口模型与标题栏</td>
        <td style="padding: 8px 10px;">单实例无边框窗口 + 后台预加载隐藏 + 管理视图自定义拖拽标题栏</td>
        <td style="padding: 8px 10px;">仅驻留单个 WebView2 进程；悬浮窗态天然无边框，管理窗态通过前端自定义拖拽区与最小化/关闭按钮提供完整窗口控制，避免动态增删系统原生边框引发 WebView2 客户区重绘错位</td>
        <td style="padding: 8px 10px;">双独立窗口（内存占用翻倍且受限于框架单窗口模型）或运行期反复切换系统原生边框样式（易导致无边框阴影与内边距计算异常）</td>
      </tr>
      <tr style="border-bottom: 1px solid #e4e4e7; background: #fafafa;">
        <td style="padding: 8px 10px; font-weight: 600; color: #18181b;">焦点衔接与检索模型</td>
        <td style="padding: 8px 10px;">唤出前锁定快照非自身前景窗口句柄 + 正常激活悬浮窗 + 粘贴前归还前景焦点</td>
        <td style="padding: 8px 10px;">激活悬浮窗使搜索框可直接接收键盘输入与方向键导航；同时因原目标窗口句柄已在唤出瞬间快照保存，用户在悬浮窗内无论点击分组/标签还是打字搜索，选中后隐藏自身并调用前景设置接口均可合法将焦点归还原窗口</td>
        <td style="padding: 8px 10px;">设置无激活扩展窗口样式（会导致悬浮窗无法接收键盘焦点，使键盘搜索与上下方向键导航彻底失效）</td>
      </tr>
      <tr style="border-bottom: 1px solid #e4e4e7; background: #ffffff;">
        <td style="padding: 8px 10px; font-weight: 600; color: #18181b;">文本键入与剪贴板保护</td>
        <td style="padding: 8px 10px;">等待修饰键物理释放 + 剪贴板纯文本替换与合成粘贴 + 序列号校验还原</td>
        <td style="padding: 8px 10px;">单次粘贴指令不受中文输入法编码拦截，长文本与特殊符号可瞬间写入；还原前比对系统剪贴板序列号，若用户在等待期内复制了新内容则放弃还原，避免覆盖新剪贴板数据</td>
        <td style="padding: 8px 10px;">逐字符发送键盘事件（长文本存在明显延迟且易被输入法或编辑器自动补全干扰）；或手写全格式 COM 对象深拷贝（实现复杂度过高，超出个人工具边界）</td>
      </tr>
      <tr style="border-bottom: 1px solid #e4e4e7; background: #fafafa;">
        <td style="padding: 8px 10px; font-weight: 600; color: #18181b;">光标定位与 DPI 适配</td>
        <td style="padding: 8px 10px;">系统图形线程光标物理坐标优先，鼠标坐标兜底 + 按窗口 DPI 换算尺寸后裁剪工作区</td>
        <td style="padding: 8px 10px;">平台层统一在物理像素空间计算坐标与窗口宽高，适配 Windows 11 高分屏缩放；当自绘应用不暴露传统光标时平滑回退至鼠标坐标</td>
        <td style="padding: 8px 10px;">混用逻辑像素与物理像素（在 125%/150% 缩放显示器下会导致悬浮窗右下角越界或偏移）</td>
      </tr>
      <tr style="border-bottom: 1px solid #e4e4e7; background: #ffffff;">
        <td style="padding: 8px 10px; font-weight: 600; color: #18181b;">本地数据库</td>
        <td style="padding: 8px 10px;">纯 Go SQLite 驱动 + 单连接池 + 连接串开启外键约束</td>
        <td style="padding: 8px 10px;">零 C 编译器依赖，单连接设置与连接串级外键开关确保内存测试与文件数据库行为一致</td>
        <td style="padding: 8px 10px;">默认多连接池配合单次执行外键语句（会导致内存数据库多连接表不可见及新连接外键失效）</td>
      </tr>
      <tr style="background: #fafafa;">
        <td style="padding: 8px 10px; font-weight: 600; color: #18181b;">WSL 测试与跨平台验证</td>
        <td style="padding: 8px 10px;">Linux 本机运行内部包单测 + Windows 交叉静态检查覆盖全量包 + 云端 CI 构建产物</td>
        <td style="padding: 8px 10px;">绕开 WSL 缺失 Linux 桌面 GTK/WebKit 头文件对根包本机编译的限制，同时保证业务逻辑单测与 Windows 平台代码静态检查在本地闭环</td>
        <td style="padding: 8px 10px;">在无桌面依赖的 WSL 下直接对含 Linux 桌面绑定的根包执行本机测试（必然因缺失系统图形库报错）</td>
      </tr>
    </tbody>
  </table>
</div>

---

## 4. 风险清单

| 风险 | 等级 | 缓解措施 |
|------|------|----------|
| 部分自绘界面应用（如浏览器、Electron 编辑器）不向系统暴露传统输入光标矩形 | 🟡 中 | 定位逻辑对返回空矩形或零句柄的情况自动回退至当前鼠标指针物理坐标，并按显示器可用工作区裁剪边界 |
| 用户按下含修饰键的全局热键后尚未松开物理按键即触发粘贴，导致组合键冲突 | 🟡 中 | 发送粘贴指令前轮询检测物理修饰键（Alt、Shift）释放状态（设超时上限），释放后再注入粘贴按键序列 |
| 剪贴板临时替换期间可能丢失非纯文本格式（如图片、文件列表）或与用户新复制动作竞态 | 🟡 中 | 仅备份与恢复纯文本格式，并在还原前校验系统剪贴板序列号，若序列号已变动则放弃还原；在文档中声明非文本格式不作恢复 |
| Windows 用户界面特权隔离（UIPI）阻止普通权限进程向管理员提权窗口注入按键 | 🟢 低 | 作为操作系统安全边界在文档中明确说明：如需向管理员终端或提权应用键入，需以相同权限运行本工具 |
| 热键重复按下或从悬浮窗内再次触发时，若无守卫会误将悬浮窗自身记录为目标窗口 | 🔴 高 | 仅在窗口从隐藏态切换至悬浮窗态且当前前景窗口非本进程窗口时更新目标窗口句柄快照 |

---

## 5. Phase 依赖关系

<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; border: 1px solid #d4d4d8; border-radius: 6px; padding: 14px; background: #fafafa; margin: 12px 0; display: flex; align-items: center; justify-content: space-between; gap: 8px; font-size: 12px;">
  <div style="flex: 1; border: 1px solid #a1a1aa; border-radius: 4px; background: #ffffff; padding: 8px 10px; text-align: center;">
    <div style="font-weight: 600; color: #18181b;">Phase 1</div>
    <div style="font-size: 11px; color: #52525b;">工程底座与数据存储</div>
  </div>
  <div style="color: #71717a; font-weight: bold;">──→</div>
  <div style="flex: 1; border: 1px solid #a1a1aa; border-radius: 4px; background: #ffffff; padding: 8px 10px; text-align: center;">
    <div style="font-weight: 600; color: #18181b;">Phase 2</div>
    <div style="font-size: 11px; color: #52525b;">平台交互与服务编排</div>
  </div>
  <div style="color: #71717a; font-weight: bold;">──→</div>
  <div style="flex: 1; border: 1px solid #a1a1aa; border-radius: 4px; background: #ffffff; padding: 8px 10px; text-align: center;">
    <div style="font-weight: 600; color: #18181b;">Phase 3</div>
    <div style="font-size: 11px; color: #52525b;">双模式前端界面实现</div>
  </div>
  <div style="color: #71717a; font-weight: bold;">──→</div>
  <div style="flex: 1; border: 1px solid #a1a1aa; border-radius: 4px; background: #ffffff; padding: 8px 10px; text-align: center;">
    <div style="font-weight: 600; color: #18181b;">Phase 4</div>
    <div style="font-size: 11px; color: #52525b;">自动化验证与文档交付</div>
  </div>
</div>

四个阶段按依赖顺序串行推进：Phase 1 确立领域实体与存储契约；Phase 2 在此基础上构建平台能力并封装前后端绑定服务；Phase 3 消费绑定服务与窗口事件完成双视图界面及前端纯函数单测；Phase 4 执行全栈跨平台静态检查、补充云端构建流水线并输出项目基线文档。

---

## 6. Subagent 协同规划

对照三维效益模型（任务复杂度、上下文关联度、经济与时间效益）评估各阶段委托需求：

| 阶段 | 上下文依赖 | 并行收益 | 执行主体 | 决策理由 |
|------|-----------|---------|----------|----------|
| Phase 1：项目初始化与本地数据存储层 | 强（定义全局领域实体、表结构与事务语义） | 无（全栈基础契约） | 主线 | 规模精炼且作为后续阶段的前置输入，主线直接完成可消除跨会话对齐损耗 |
| Phase 2：原生平台交互层与应用编排服务层 | 强（紧密耦合窗口生命周期、句柄状态与存储接口） | 无（依赖 Phase 1 且阻塞 Phase 3） | 主线 | 涉及窗口句柄与焦点切换的时序不变量，保留主线可保障契约一致性 |
| Phase 3：单实例双视图前端界面实现 | 强（直接绑定 Phase 2 服务接口与事件负载） | 低（单页双视图共享同一状态容器） | 主线 | 组件数量少且共享数据模型，主线连续实现比拆分委托更节省上下文开销 |
| Phase 4：自动化验证、云端构建配置与项目文档建立 | 强（汇总前序全部实现细节与架构决策） | 无（收尾闭环） | 主线 | 验证命令与文档编写直接复用主线完整上下文，无需冷启动子代理重复读盘 |

> **协同结论**：本项目为边界清晰、内聚度高的轻量级桌面工具，四个阶段呈串行契约依赖且总文件数精简；启动隔离 Subagent 重建上下文的开销高于任务本身，全流程保留主线执行最具时间与经济效益。

---

> ### Task 拆解预览
>
> 实际可执行任务详见 `Tasks.md`，以下为阶段与任务映射概览：
>
> | Phase | 预估 Task 数 | 执行主体 | 核心交付内容 |
> |-------|-------------|----------|-------------|
> | Phase 1 | 3 | 主线 | 工程配置初始化、SQLite 存储与批量操作实现、存储层单元测试 |
> | Phase 2 | 3 | 主线 | 平台抽象与几何裁剪、Windows 原生交互与 Linux 桩实现、应用编排服务与单实例启动入口 |
> | Phase 3 | 3 | 主线 | 前端工程底座与纯函数过滤单测、快捷悬浮窗组件、带自定义标题栏的主管理窗口组件与批量交互 |
> | Phase 4 | 2 | 主线 | 跨平台静态检查与云端构建流水线配置、项目基线文档建立 |
>
> **总计预估**：11 个 Task。
