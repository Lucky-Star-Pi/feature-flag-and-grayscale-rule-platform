# AI 辅助开发过程记录

> 说明：本文件是「Feature Flag 与灰度规则平台」开发过程中 AI 辅助开发记录。
> 按题目要求随项目一并提交。记录经过整理与裁剪（保留与开发相关的真实对话节选），
> 并附操作日志作为旁证。**所有内容均来自真实开发过程，未虚构。**

---

## 0. 提交物清单

本记录与以下文件/证据一并提交，供交叉印证：

- [x] 本文件 `docs/ai-development-log.md`（对话节选 + 决策 + 结果）
- [x] `git log` 提交记录（M5 三次提交后见操作日志）
- [x] 关键命令与测试运行输出（见操作日志；以本轮终端为准）
- [x] 关键 bug：Gin 路由冲突、Postgres 未启动（见各阶段踩坑）

---

## 1. 记录结构约定

每个开发阶段按以下四栏记录，保证「真实展示用工具辅助开发的过程」：

| 栏目 | 要写什么 |
| --- | --- |
| 我的提问 | 当时真实发给 AI 的 prompt（可略作精简，不改语义） |
| AI 输出（节选） | AI 给出的关键方案/代码/结论，保留最能说明问题的部分 |
| 我的决策与踩坑 | 我采不采纳、为什么改、踩了什么坑、怎么定位的 |
| 结果验证 | 测试是否通过、运行结果、日志/截图 |

> 编写原则：
> 1. **真实 > 完美**：如实记录让 AI 修正、自己 debug 的反复过程，评卷人看重真实感。
> 2. **裁剪可、编造不可**：可删除跑题内容、归纳长对话；不可虚构"一次成功"的假象。
> 3. **对话与结果要能对上**：每段"结果验证"尽量落到具体测试用例或命令输出，而非"运行正常"。

---

## 2. 开发阶段记录

### 阶段 1：需求分析

**我的提问**
```text
# 角色
你是一名在头部大厂有 5 年以上经验的资深全栈架构师 + 需求分析师，擅长把业务诉求拆解成"可直接落地实现、可直接写测试验证、数据一致性有保障"的工程方案。你现在帮我完成滴滴外包岗技术笔试的【前期需求分析】阶段。

# 背景与评分标准（务必围绕这四点展开，UI 美观不在考察范围）
我正在参加滴滴外包岗笔试，考核"用 AI Coding 工具独立完成全栈工程"的能力。评卷人只看四件事：
1. 业务逻辑是否正确；
2. 数据一致性是否得到保证（事务、并发、幂等、状态机、边界条件）；
3. 核心场景是否有自动化测试；
4. AI 对话记录是否真实体现"用工具辅助开发"的完整过程。

# 待分析需求（真实考题原文，技术栈已由题目强制锁定）
项目名称：Feature Flag 与灰度规则平台（轻量级 Feature Flag 平台，用于控制某项功能在不同环境下是否开启，并可根据用户属性配置有序规则）。从零实现一个可运行的全栈应用。

技术栈（强制，不可更换）：前端 React、后端 Go、数据库 PostgreSQL；不提供起始代码；框架/组件库/ORM/迁移工具/测试工具自选；必须使用 AI 辅助开发。

## 必做功能
1. Feature Flag 管理
   - 每个 Flag 至少包含：名称、Key、环境（development/staging/production）、是否启用、默认返回值（true/false）、创建时间、更新时间。
   - 关键约束：同一环境下 Key 必须唯一；不同环境可使用相同 Key。
   - 支持：Flag 列表、搜索或筛选、新建、编辑基本信息、启用和停用、查看详情。

2. 有序匹配规则
   - 每个 Flag 可配置多条带优先级的规则。每条规则至少包含：用户属性名（如 country、plan、user_id）、操作符（equals 或 in）、期望值、命中后返回值（true/false）、优先级。
   - 规则语义：按优先级从高到低或从小到大依次判断（方向由我决定，须在界面与 README 写清）；第一条命中的规则决定结果，后续规则不再计算；没有规则命中时返回 Flag 默认值；Flag 被停用时始终返回 false。
   - 重复优先级必须采用确定性策略：可以拒绝重复，也可以定义明确的二级排序；所选策略须在前后端和 README 中保持一致。

3. 在线评估控制台
   - 输入：Flag Key、环境、一段 JSON 格式的用户属性；调用后端完成评估。
   - 展示：最终返回值、是否命中规则、命中了哪条规则（未命中则说明使用了默认值）。
   - 对 JSON 格式错误、Flag 不存在等错误给出清晰反馈。

4. 操作历史
   - 记录：创建 Flag、修改基本信息、增删改规则、启用或停用。
   - 历史至少包含：操作类型、发生时间、必要的变更摘要。
   - 不要求完整登录系统，可使用固定操作者。
   - 硬约束：同一次请求中的业务变更与历史写入必须保持原子一致（同一事务）。

## 最低页面范围
Flag 列表页、新建/编辑页、Flag 详情与规则管理页、评估控制台。页面不要求精美，但要有清晰的信息层级、表单校验、加载状态、空状态、成功/失败反馈。

## 工程要求
- 提供数据库迁移和适量演示数据。
- 在【数据库层】保证"同一环境 Key 唯一"。
- 为规则评估的核心逻辑编写自动化测试。
- README 说明数据模型、规则优先级、匹配语义、启动方式、设计取舍、已完成项和未完成项。
- 随项目提交 AI 对话记录、操作日志等。

## 可选扩展（无独立分值，不替代必做项）
稳定的百分比灰度发布；配置版本/回滚/乐观锁；规则拖拽排序；环境间配置复制或发布；登录与角色权限；前端自动化测试或在线部署。

# 输出要求：严格按以下结构输出《需求分析文档》
不写代码，但方案要细化到"照着它就能直接实现和写测试"的粒度。

1. 业务概述与目标
   - 一句话说清业务；核心目标；明确列出"非目标/不做的事"（如不做登录、不做多租户，除非列为可选扩展）。

2. 角色与权限
   - 本题明确"不要求完整登录系统，可用固定操作者"，请明确默认操作者方案及其边界。

3. 功能需求清单（含验收标准）
   - 每个功能点给：用户故事 + 前置条件 + 主流程 + 可测试的验收标准（Given/When/Then）。

4. 核心业务流程
   - 用 mermaid 时序图/流程图描述 2~4 个核心流程：尤其"评估一次 Flag"的完整决策链路（停用短路 → 按优先级匹配 → 默认值兜底）、以及"业务变更 + 历史写入同事务"的流程。

5. 数据模型与一致性设计【评卷重点，务必深入，禁止一笔带过】
   - 实体、字段、主键/唯一约束、关联关系（ER 说明）。重点说清 flags 与 rules 的关系、history 表设计。
   - 【必考坑 1：Key 唯一必须落在 DB 层】明确用复合唯一索引 UNIQUE(key, environment) 而非"应用层先查后插"（否则并发创建会穿透），并说明 Go 侧如何把 PostgreSQL 的 unique_violation(23505) 翻译成"业务冲突"反馈（如 HTTP 409）。
   - 【必考坑 2：历史原子一致性】所有变更（创建/编辑/启停/规则增删改）必须与对应 history 插入放在同一事务；画出每个操作的事务边界，并说明失败时如何整体回滚。
   - 【必考坑 3：规则评估的确定性】优先级排序方向（升序/降序）在前后端+README 三处保持一致；重复优先级采用"拒绝（400）"或"二级排序（如按 id 升序）"二选一，并写清前后端各自如何保证。
   - 【必考坑 4：停用短路】enabled=false 时直接返回 false，不评估任何规则，响应体明确 matched=false 并给出原因，避免"停用了还命中规则返回 true"的 bug。
   - 【必考坑 5：属性缺失/类型不一致】用户 JSON 中属性缺失、值为 number vs string 时的匹配语义要显式定义（建议统一转 string 比较，缺失视为不命中）。
   - 失败补偿与幂等：评估接口只读可重试；写接口（创建/启停/改规则）如何处理重复提交、并发编辑（可指向乐观锁扩展项）。
   - 单独列出"最容易出 bug 的 3 个数据一致性陷阱"，每个给出具体解法 + 对应验证手段。

6. API 契约设计
   - 资源/方法/路径/请求响应字段/状态码/错误码表；逐接口标注【需事务】【需唯一约束校验】【需确定性排序】。
   - 特别定义：评估接口的响应结构（finalValue / matched / matchedRule / reason），以及"Flag 不存在""JSON 非法""Key 冲突"各自的错误码与提示。

7. 技术栈选型（大框架已被题目锁定为 React + Go + PostgreSQL，只选内部件）
   - Go：HTTP 框架（Gin/Echo/chi）、DB 访问（database/sql+pgx 或 sqlx 或 GORM）、迁移工具（golang-migrate/goose）。
   - React：Vite + TS、数据请求（TanStack Query 或 SWR）、UI 组件库（可选）。
   - 测试：Go 单测（testify）+ 集成测试（testcontainers 或 dockertest 拉起真 PostgreSQL 验证唯一约束与事务回滚）；前端可选用 Vitest + React Testing Library（可选扩展）。
   - 每个选择给出理由，优先"易写测试、易本地复现"。

8. 自动化测试策略【评卷重点】
   - 单元测试：规则评估核心逻辑（纯函数式）全覆盖，覆盖 equals/in、优先级短路、默认值、停用短路、重复优先级、属性缺失/类型不一致。
   - 集成测试（必须用真 PostgreSQL）：唯一约束拒绝同环境重复 Key、允许不同环境同 Key、变更+历史同事务原子回滚。
   - 端到端/接口测试：评估控制台主链路、Flag 生命周期、规则增删改、错误场景反馈。
   - 输出"必须覆盖的核心场景清单"，至少 10 条，每条注明断言点，至少包含：
     ① Flag 停用恒返回 false（即便有规则会命中 true）；
     ② 多规则按优先级首条命中即短路；
     ③ 无规则命中返回默认值（分别测默认 true/false）；
     ④ equals 命中与不命中；⑤ in 命中、不命中、空数组；
     ⑥ 重复优先级：拒绝策略返回 400（或二级排序结果确定）；
     ⑦ 评估接口：Flag 不存在、JSON 非法、属性缺失各自返回明确错误；
     ⑧ 同环境重复 Key 创建被 DB 唯一约束拒绝并返回业务冲突，不同环境同 Key 允许；
     ⑨ 业务变更与历史写入同事务，任一失败整体回滚；
     ⑩ 规则增删改后历史记录正确、变更摘要可读。

9. 开发排期与里程碑
   - 拆成可独立验证的小步（每步都能跑起来、都有测试），标注每步交付物；贴合"6~8 小时、48h 内一次性提交"的节奏。

10. AI 辅助开发过程记录建议
    - 建议我在每个阶段如何留痕：向 AI 提了什么问题、AI 给了什么、我做了什么决策/修正、测试结果与日志截图。
    - 给出"真实性"建议：哪些地方要体现我踩坑、debug、让 AI 修正的反复过程（评卷人非常看重真实感，模板化流水账会扣分）。

# 输出格式与语气
- 全中文、结构化 Markdown，可直接粘贴到文档。
- 宁可有争议但具体的方案，也不要正确的废话。
- 每一条"数据一致性结论"都必须说明"为什么 + 怎么用测试/断言验证"。
- 如对某点语义仍有歧义（如 in 的匹配方式、属性类型转换），请明确列出你的假设和选择，不要默默臆断。 
```

**AI 输出（节选）**
```text
需求分析定稿结论（非原始全文，由 README「已锁定语义 / 设计取舍」与现有代码反推）：

- 优先级：数字越小越高；评估与规则列表均为 priority ASC, id ASC。
- 重复优先级：拒绝 → 400 PRIORITY_CONFLICT；应用层先拦，DB UNIQUE(flag_id, priority) 兜底。
- 停用短路：enabled=false 恒返回 false，不评估任何规则（reason=disabled）。
- 匹配：equals 为字符串化后精确相等；in 的 expectedValue 解析 JSON 字符串数组，失败或 [] 跳过；
  属性缺失 / null / 对象 / 数组跳过；number 用 FormatFloat、bool 为 "true"/"false"。
- 一致性：同环境 Key 唯一只信任 DB UNIQUE(key, environment)，INSERT 捕获 23505 → 409 KEY_CONFLICT；
  写操作与 history 同一事务；评估做成 internal/eval 纯函数便于单测。
- 技术栈：Gin + sqlx/pgx + golang-migrate；Vite + React + TS + antd + TanStack Query。

完整原始输出见当时 Plan / 聊天记录，此处为最终定稿结论。
```

**我的决策与踩坑**
- 决策 1：优先级方向定为「数字越小优先级越高（升序）」，前端/后端/README 三处统一。
- 决策 2：重复优先级采用「拒绝并返回 400」，由后端唯一校验保证。

**结果验证**
- 需求分析文档已输出，关键语义（停用短路、默认值兜底、属性缺失处理）已定稿。

---

### 阶段 2：项目搭建

**我的提问**
```text
由于模型需求分析完后直接输出M0阶段的脚手架搭建，此处只补充：新增全局规则，项目阶段开发结束后，需通过自动化测评和人工审核双重校验通过才能进行下一阶段的开发。
```

**AI 输出（节选）**

```text
模型新增规则并跑通自动化测评
```

**我的决策与踩坑**
- 手动校验README.md流程是否跑通，M0阶段手动校验

**结果验证**
- `go run ./...` 与 `npm run dev` 均可启动，前后端连通。
  前端输出正常

---

### 阶段 3：数据模型与迁移

**我的提问**
```text
M0阶段已完成人工校验，请按以下提示词进行开发：
# 角色与上下文
你是资深 Go + React 全栈工程师。我们现在开始实现「Feature Flag 与灰度规则平台」（需求分析已定稿，见上文/附件）。本阶段只做 M1：项目骨架 + 数据模型与迁移，目标是"能跑起来 + 数据库结构正确 + 有最小测试验证"。不要提前实现完整业务。

# 硬约束（必须遵守）
- 技术栈：后端 Go、前端 React（Vite+TS）、数据库 PostgreSQL。
- 「同一环境 Key 唯一」必须在数据库层保证：flags 表用复合唯一索引 UNIQUE(key, environment)。
- 所有业务变更与历史写入必须同一事务（本阶段先搭好事务工具封装，M2 再实现完整 CRUD）。
- 每个交付物都要给出可运行的验证命令或测试。

# M1 交付物
1. 后端项目骨架
   - 选 HTTP 框架（建议 Gin，有更优理由可换并说明）、给出目录结构、配置管理（读 DATABASE_URL 等环境变量）。
   - 提供 /healthz 健康检查接口。
2. 数据库迁移
   - 用 golang-migrate 或 goose 建迁移脚本，含三张表：
     - flags：id, name, key, environment, enabled, default_value, created_at, updated_at；UNIQUE(key, environment)。
     - rules：id, flag_id(FK), attribute, operator(equals/in), expected_value, return_value, priority, created_at, updated_at；说明重复优先级采用"拒绝"策略落在哪一层。
     - history：id, flag_id(FK 可空), operation_type, operator, summary, created_at。
   - 附 demo 种子数据（1~2 个 flag + 若干规则）。
3. 数据库连接与错误映射工具
   - 封装 PostgreSQL 连接；封装"unique_violation(23505) 转业务冲突错误"的函数（M2 用）。
   - 封装"事务执行"辅助函数（M2 用）。
4. 前端骨架
   - Vite + TS + React Router 初始化，空页面占位（列表/详情/评估），能启动并访问。
5. 最小测试验证
   - Go 单测跑通一个示例（可先测 healthz handler）。
   - 说明后续集成测试将用真 PostgreSQL 验证唯一约束与事务（本阶段可选先搭 testcontainers/dockertest 骨架）。

# 输出要求
- 给出完整目录结构、迁移 SQL、关键 Go 代码、启动命令、测试命令。
- 明确"下一步 M2 做什么"，但本阶段不要提前实现完整 CRUD。
- 全中文说明。 
```

**AI 输出（节选）**
```text
M1 已交付，停在人工评审门禁
已按你的 M1 提示词收敛范围：只做骨架 + 迁移 + DB 工具 + 前端占位 + 最小测试；已删除提前写的完整 CRUD / eval / 业务 API。

交付摘要
项	说明
HTTP
Gin；仅 GET /healthz
迁移
flags / rules / history；UNIQUE(key, environment)；种子 2 个 Flag + 规则
重复 priority
DB：UNIQUE(flag_id, priority) 拒绝；应用层 400 留给 M2
DB 工具
db.Open、WithTx、MapUniqueViolation(23505)
前端
列表 / 详情 / 评估三页占位，可启动
测试
healthz + MapUniqueViolation 单测通过
已跑通的验证
go test ./internal/http/ ./internal/db/ -count=1   # ok
GET /healthz → {"status":"ok"}
flags 存在 UNIQUE CONSTRAINT flags_key_environment_uk
同环境重复插入 checkout_v2@development → 23505 拒绝
npm run build → 通过
关键路径
迁移：backend/migrations/0001_init.up.sql
DB 工具：backend/internal/db/db.go
启动与评测清单：README.md
下一步 M2（本阶段不实现）
Flag/规则 CRUD、history 同事务、23505→HTTP 409、真库集成测试唯一约束与回滚。

请人工评审 M0 清单后，若通过请回复：M1 通过，进入 M2
（未收到前不会开始 M2。）
```

**我的决策与踩坑**
- 决策：`flags` 表用复合唯一索引 `UNIQUE(key, environment)`，确保「同一环境 Key 唯一」落在数据库层。
- 决策：`history` 表字段含 operation_type / operator / summary / created_at。

**结果验证**
- 迁移执行成功；手动插入两条同环境同 Key 记录被数据库拒绝。

---

### 阶段 4：后端 API

**我的提问**
```text
# 角色与上下文
你是资深 Go + React 全栈工程师。我们正按里程碑开发「Feature Flag 与灰度规则平台」，M0、M1 已通过人工评审。现在只做 M2，不要提前实现 M3（规则评估逻辑）和 M4（前端对接）。

# 当前项目状态（已存在，务必复用，禁止重造）
- Go 模块名 `featureflag`，go 1.22.12；依赖已有 gin、sqlx、pgx、golang-migrate、testify。
- `internal/db/db.go` 已提供并直接使用：
  - `Open(dsn)` / `Ping(ctx)` / `Close()`
  - `WithTx(ctx, fn)`：BeginTxx + Commit/Rollback 封装，M2 所有「业务变更 + 历史写入」必须放同一 fn 内
  - `MapUniqueViolation(err)`：把 PostgreSQL 23505 映射为 `ErrFlagKeyConflict` / `ErrRulePriorityConflict`（已定义）
- `internal/http/handlers.go`：目前 `NewRouter()` 无参数、仅挂 `/healthz`。M2 需改造为 `NewRouter(deps)` 注入 `*db.DB`（或 service）；`cmd/server/main.go` 相应更新调用。
- `internal/config/config.go`：env `HTTP_ADDR` / `DATABASE_URL` / `MIGRATIONS_PATH`。
- 迁移（已应用，禁止改动）：
  - flags：id,name,key,environment,enabled,default_value,created_at,updated_at；UNIQUE(key,environment)；environment CHECK IN ('development','staging','production')
  - rules：id,flag_id(FK CASCADE),attribute,operator CHECK IN ('equals','in'),expected_value,return_value,priority,created_at,updated_at；UNIQUE(flag_id,priority)
  - history：id,flag_id(FK SET NULL 可空),operation_type,operator,summary,created_at
- `internal/model`、`internal/store`、`internal/service` 为空目录，M2 填 model/store/service；`internal/eval` 留空给 M3。
- 操作者固定 `local-admin`。
- 本阶段不需要新增迁移；如确需加索引/约束，只能新增 0003 迁移。

# M2 硬约束
1. 全部走 PostgreSQL 真库；不得用内存 mock 冒充持久化。
2. 「同环境 Key 唯一」只信任 DB 的 UNIQUE(key,environment)：直接执行写入，捕获 23505 经 `db.MapUniqueViolation` 映射。
   - `ErrFlagKeyConflict` → HTTP 409
   - `ErrRulePriorityConflict` → HTTP 400（重复优先级属于输入错误）
   （若现有代码注释与"优先级→400"语义冲突，一并修正注释保持一致）
3. 所有写操作（创建/编辑/启停/规则增删改）必须：业务写 + 对应 history 写放在同一个 `WithTx` 事务内，任一失败整体回滚。
4. 规则优先级策略 = 「拒绝重复」：应用层先校验同 flag 内 priority 已存在则 400，DB UNIQUE 兜底 400。
5. 优先级方向 = 数字越小优先级越高（升序）；本阶段在代码注释与 README 写明，M3 评估沿用。
6. 不实现评估逻辑（`/evaluate`、`internal/eval` 留到 M3）；不改前端页面（留到 M4）。

# M2 交付物

## 1. model 包
定义 `Flag` / `Rule` / `History` 结构体，db tag 与上表列名一一对应，JSON tag 用小驼峰（如 default_value → defaultValue）。

## 2. store 包（数据访问，sqlx）
- ListFlags(filter)：支持 key 模糊、environment 精确、enabled 过滤。
- GetFlag(id) / GetFlagByKeyAndEnv(key, env)。
- CreateFlag / UpdateFlag / SetFlagEnabled。
- ListRules(flagID) 按 priority 升序；CreateRule / UpdateRule / DeleteRule。
- ListHistory(flagID) 按 created_at 降序。
- 写操作接收 `*sqlx.Tx`（供 service 在事务内调用）；查询接收 `sqlx.Ext` 或 `*sqlx.DB`。

## 3. service 包（业务 + 事务编排）
- CreateFlag：校验 environment 合法、default_value 必填；`WithTx` 内 insert flag + history(CREATE_FLAG)，23505→ErrFlagKeyConflict。
- UpdateFlag：先确认存在；`WithTx` 内 update + history(UPDATE_FLAG，summary 含变更字段 旧→新)。
- EnableFlag / DisableFlag：`WithTx` 内 update enabled + history(ENABLE_FLAG/DISABLE_FLAG)。
- 规则增删改：`WithTx` 内业务写 + history(CREATE_RULE/UPDATE_RULE/DELETE_RULE)；应用层先查 priority 重复返回 ErrRulePriorityConflict，DB 23505 兜底。
- 资源不存在统一返回一个 `ErrNotFound`（HTTP 层转 404）。

## 4. http 包（路由 + 处理器）
建议契约（可微调，但必须前后端与 README 一致）：
- GET    /api/v1/flags                    列表/搜索/筛选
- POST   /api/v1/flags                    新建（201；409 Key 冲突；400 非法 environment）
- GET    /api/v1/flags/:id                详情（含按 priority 升序的 rules）
- PATCH  /api/v1/flags/:id                编辑基本信息 name/default_value（404/409/400）
- POST   /api/v1/flags/:id/enable         启用
- POST   /api/v1/flags/:id/disable        停用
- POST   /api/v1/flags/:flagId/rules      新增规则（400 重复 priority / 非法 operator / priority<0）
- PATCH  /api/v1/flags/:flagId/rules/:ruleId  编辑规则（同上校验）
- DELETE /api/v1/flags/:flagId/rules/:ruleId  删除规则（204）
- GET    /api/v1/flags/:id/history        操作历史（created_at 降序）

统一错误结构：`{"error":{"code":"...","message":"..."}}`；错误码 KEY_CONFLICT(409)/PRIORITY_CONFLICT(400)/NOT_FOUND(404)/INVALID_INPUT(400)。把 db 层 sentinel error 与 HTTP 状态码的映射集中到一个 helper（如 writeError）。

## 5. 测试（M2 重点）
A. 单元测试：保留现有 MapUniqueViolation 用例；可补充不依赖 DB 的纯逻辑校验。
B. 集成测试（真 PostgreSQL，Windows 安全，禁止 testcontainers）：
   - 通过 `TEST_DATABASE_URL` 读测试库；未设置则 `t.Skip("TEST_DATABASE_URL 未设置，跳过集成测试")`。
   - 测试前对测试库跑迁移（复用 migrateutil 或建独立 schema 后清理）。
   - 必测用例：
     ① 同环境重复 Key 创建 → 409，且**没有**对应 history 被写入（同时验证：唯一约束 + 错误映射 + 原子回滚）
     ② 不同环境同 Key 创建 → 成功
     ③ 创建 flag 成功后 flags 有记录且 history 有 CREATE_FLAG（验证同事务写入）
     ④ 同 flag 重复 priority 创建规则 → 400，且无 history 残留（原子回滚 + 优先级拒绝）
     ⑤ 编辑/启停后 history 记录正确、summary 可读
     ⑥ 对不存在的 flag 取详情/增删改规则 → 404

# 输出要求
- 给出新增/修改文件的完整代码与路径，明确 main.go、handlers.go 等哪些被改动。
- 给出 `go test ./...` 与集成测试跑法（含 Windows 下设置 TEST_DATABASE_URL 的命令）。
- 说明下一步 M3 做什么，但本阶段不实现评估逻辑。
- 更新 README 的 M2 状态、API 契约与错误码说明。
- 全中文；关键语义（优先级方向、错误码映射）务必写进 README。 
```

**AI 输出（节选）**
```text
Gin 不允许同一前缀混用 :id 和 :flagId，规则路由统一改为 :id。

M2 已交付，停在人工评审
复用了 M1 的 db.WithTx / MapUniqueViolation，未改迁移、未做评估、未改前端。

评测结果
go test ./... -count=1
ok  internal/db
ok  internal/http     # 含真库集成测试 ①～⑥
ok  internal/service  # 环境/操作符/priority 校验单测
TEST_DATABASE_URL 指向 localhost:5433 的 featureflag 库。

关键约定（已写入 README）
同环境 Key 冲突：DB 23505 → 409 KEY_CONFLICT
重复 priority：应用层先拒 + DB UNIQUE 兜底 → 400 PRIORITY_CONFLICT
优先级：数字越小越高（priority ASC）
写操作与 history 都在同一个 WithTx 里
规则路径用 :id（Gin 不能混用 :id 和 :flagId）
改动文件
新增：internal/model/model.go、internal/store/store.go、internal/service/service.go + 单测
改造：internal/http/handlers.go（NewRouter(svc)）、cmd/server/main.go
集成测试：internal/http/api_int_test.go
文档：README.md
启动 / 测试
cd backend
$env:DATABASE_URL = "postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable"
go run ./cmd/server
# GET /healthz  /api/v1/flags
$env:TEST_DATABASE_URL = "postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable"
go test ./... -count=1
下一步 M3（本阶段不实现）
internal/eval 纯函数 + POST /evaluate + 停用短路/优先级/equals/in 单测。

通过后请回复：M2 通过，进入 M3
```

**结果验证**
- `go test ./...` 全部通过（含并发与事务回滚用例）。

---

### 阶段 5：规则评估核心逻辑

**我的提问**
```text
# 角色与上下文
你是资深 Go 全栈工程师。我们按里程碑开发「Feature Flag 与灰度规则平台」，M0/M1/M2 已通过人工评审。现在只做 M3：规则评估。不要提前实现 M4（前端对接），不要动迁移，不要动前端页面。

# 当前项目状态（已存在，务必复用，禁止重造）
- Go 模块 `featureflag`；依赖 gin/sqlx/pgx/golang-migrate/testify 已装齐。
- `internal/model`：`Flag`/`Rule`/`History` 结构体已定义，JSON tag 为 camelCase（`defaultValue`/`returnValue`/`expectedValue`/`flagId` 等）。
- `internal/store`：
  - `GetFlagByKeyAndEnv(ctx, q, key, env)` → `*model.Flag`，找不到返回 `store.ErrNotFound`
  - `ListRules(ctx, q, flagID)` → `[]model.Rule`，已按 `priority ASC, id ASC` 排序，空返回空切片
- `internal/service`：`Service{DB *db.DB}`、`New(database)`、`ErrNotFound`、`ErrInvalidInput`、`ValidEnvironment(env)`、`mapStoreErr(err)`（把 store.ErrNotFound → service.ErrNotFound）。
- `internal/http`：`NewRouter(svc *service.Service)` 已挂 `/api/v1` 组；`writeError(c, err)` 已集中映射：
  - `db.ErrFlagKeyConflict`→409 KEY_CONFLICT；`db.ErrRulePriorityConflict`→400 PRIORITY_CONFLICT
  - `service.ErrNotFound`→404 NOT_FOUND（当前 message 固定"资源不存在"）
  - `service.ErrInvalidInput`→400 INVALID_INPUT（message=err.Error()）
  - 其它→500 INTERNAL_ERROR
- `internal/eval` 目录当前为空，M3 填它。
- 迁移 0001/0002 已应用且**禁止改动**。`rules.expected_value` 是 TEXT；seed 中 `in` 规则的 expected_value 为 `["pro","enterprise"]`。
- 前端仍是占位页，M3 **禁止修改**。

# M3 语义锁定（必须严格按此实现，且写进代码注释 + README；这是评卷重点）
优先级方向沿用 M2：**数字越小优先级越高**；同优先级按 `id ASC` 二级排序（确定性兜底）。

1. **停用短路**：`enabled == false` 时，不评估任何规则，直接返回 `value=false, matched=false, reason="disabled"`。
2. **有序匹配**：规则按 `priority ASC, id ASC` 逐条判断；第一条命中的规则决定结果，后续不再计算。
3. **默认值兜底**：没有规则命中 → 返回 `value=flag.default_value, matched=false, reason="default"`。
4. **equals 语义**：把用户属性值「字符串化」后与 `expected_value` 做**精确相等**比较。
5. **in 语义**：`expected_value` 是 JSON 数组字符串（如 `["pro","enterprise"]`），解析成 `[]string` 后判断「字符串化的属性值」是否在数组中；**解析失败或空数组 → 该规则视为不命中（跳过，继续下一条）**。
6. **属性缺失**：规则要求的 `attribute` 不在用户属性中 → 该规则不命中（跳过）。
7. **类型归一（字符串化规则，前后端/README 一致）**：
   - string → 原样
   - number（JSON 解析后是 float64）→ 用 `strconv.FormatFloat(v,'f',-1,64)`（整数不带小数点：123→"123"，123.5→"123.5"）
   - bool → "true"/"false"
   - null / 对象 / 数组 → 视为「缺失」，该规则跳过
8. **结果结构**（评估成功响应）：
   `{"value":bool, "matched":bool, "matchedRule":Rule|null, "reason":"disabled"|"matched"|"default"}`

# M3 交付物

## 1. internal/eval（纯函数，禁止 import db/store/http，不碰 IO）
- `type Result struct { Value bool; Matched bool; MatchedRule *model.Rule; Reason string }`
- `func Evaluate(flag model.Flag, rules []model.Rule, attrs map[string]any) Result`
  - 入口先处理停用短路；内部**自行**按 `priority ASC, id ASC` 排序（不信任调用方顺序）。
- 导出两个可单测的辅助函数（或等价实现）：
  - `Stringify(v any) (string, bool)`：按上面类型归一规则；返回 false 表示「不可比/视为缺失」。
  - `ParseInValues(expected string) ([]string, bool)`：解析 JSON 数组；失败返回 false。

## 2. internal/service
- 新增 `ErrFlagNotFound`（如 `errors.New("flag not found")`）。
- 新增 `func (s *Service) Evaluate(ctx, key, env string, attrs map[string]any) (*eval.Result, error)`：
  - `strings.TrimSpace(key)` 为空 → `ErrInvalidInput`（key 必填）
  - `!ValidEnvironment(env)` → `ErrInvalidInput`（environment 非法）
  - `store.GetFlagByKeyAndEnv` 找不到 → 返回 `ErrFlagNotFound`（不是通用 ErrNotFound）
  - `store.ListRules` → `eval.Evaluate`，返回结果。

## 3. internal/http
- 新增路由 `POST /api/v1/evaluate`，handler `Evaluate`。
- 请求体：
  `{"key":"...","environment":"...","attributes":{...}}`
  - `attributes` 类型 `map[string]any`；缺省或为 null → 视为空对象 `{}`。
  - `attributes` 不是 JSON 对象（字符串/数组）→ 400 INVALID_INPUT，message「attributes 必须是 JSON 对象」。
  - 请求体整体非法 JSON → 由 `ShouldBindJSON` 失败转 400 INVALID_INPUT（满足题目"JSON 格式错误"反馈）。
- `writeError` 增加分支：`service.ErrFlagNotFound` → 404 NOT_FOUND，message「Flag 不存在」（评估接口的 404 必须明确到"Flag 不存在"）。

## 4. 测试
A. **表驱动单测**（`internal/eval/eval_test.go`，纯函数、无 DB）至少覆盖：
   ① 停用恒 false（即便有会命中 true 的规则）
   ② 多规则按 priority 首条命中即短路（构造高优先级命中、低优先级也命中的情况，断言返回的是高优先级那条）
   ③ 无规则命中 → 默认值（分别测 default true / false）
   ④ equals 命中、不命中
   ⑤ in 命中、不命中、空数组 `[]`、解析失败（如 expected="not-json"）→ 跳过
   ⑥ 属性缺失 → 跳过
   ⑦ 类型归一：`user_id` 数字 123 与 expected "123" 命中；bool true 与 "true" 命中；null/对象/数组 → 跳过
   ⑧ 同优先级多条 → 按 id ASC 二级排序确定性命中
B. **Stringify / ParseInValues 单测**：整数不带小数点、小数、bool、null、非法 JSON。
C. **集成测试**（加到 `internal/http/api_int_test.go`，沿用 `TEST_DATABASE_URL` + `t.Skip` 模式）：
   - 用 seed 的 `checkout_v2@development`（或用 uniqueKey 新建 flag + 规则）走 `POST /api/v1/evaluate`：
     ⑨ attributes 命中规则 → matched=true、value 正确、matchedRule 非空
     ⑩ attributes 不命中任何规则 → matched=false、value=默认值、reason=default
     ⑪ Flag 不存在 → 404 NOT_FOUND，message「Flag 不存在」
     ⑫ 非法 environment → 400；attributes 非对象 → 400；请求体非法 JSON → 400

# 输出要求
- 给出新增/修改文件的完整代码与路径（eval、service、http、测试），明确哪些文件被改动。
- 说明 `go test ./...`（含 TEST_DATABASE_URL）与「仅单测不连库」的跑法。
- 更新 README：新增 `/api/v1/evaluate` 契约（请求/响应/错误码）、评估语义（停用短路、优先级方向、equals/in、类型归一、reason 枚举），并把 M3 状态改为「待人工评审」。
- 更新 `docs/ai-log.md` 的 M3 段落（提示词要求 + 实现要点 + 验证命令）。
- 全中文；关键语义必须写进代码注释与 README，保证三处一致。
```

**AI 输出（节选）**
```text
M3 已交付，停在人工评审
未改迁移、未改前端。评估语义按提示词锁定：disabled / matched / default。

评测结果
go test ./... -count=1
ok  internal/eval     # ①～⑧ + Stringify/ParseInValues
ok  internal/http     # ⑨～⑫ 真库 evaluate
ok  internal/db
ok  internal/service
新增 / 修改
路径	说明
新增 backend/internal/eval/eval.go
纯函数 Evaluate / Stringify / ParseInValues
新增 backend/internal/eval/eval_test.go
表驱动单测
改 backend/internal/service/service.go
ErrFlagNotFound + Evaluate
改 backend/internal/http/handlers.go
POST /api/v1/evaluate；404 message「Flag 不存在」
改 backend/internal/http/api_int_test.go
命中 / 默认值 / 404 / 400
改 README、docs/ai-log.md、docs/PHASE_GATES.md
M3 契约与语义
评估契约
POST /api/v1/evaluate

{"key":"checkout_v2","environment":"development","attributes":{"country":"CN"}}
成功：{"value":bool,"matched":bool,"matchedRule":Rule|null,"reason":"disabled"|"matched"|"default"}

停用：不评估规则，恒 value=false, reason=disabled
数字越小越先匹配；in 解析失败或 [] 跳过
number 123 与 "123" 命中（FormatFloat）
测试命令
cd backend
# 仅单测（不要设 TEST_DATABASE_URL，集成测试会 Skip）
go test ./internal/eval/ ./internal/db/ ./internal/service/ ./internal/http/ -count=1
$env:TEST_DATABASE_URL = "postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable"
go test ./... -count=1
下一步 M4（本阶段不实现）
四个页面调真实 API，评估控制台展示 value/matched/matchedRule/reason。

通过后请回复：M3 通过，进入 M4
```

**我的决策与踩坑**
- 评估 404 单独用 `ErrFlagNotFound`，避免和通用「资源不存在」混在一起。
- `in` 解析失败或空数组视为跳过，而不是 500。
- 更早一次 `go test` 因 Postgres 容器未启动失败；拉起 5433 后再跑全绿。

**结果验证**
- `go test ./internal/eval/` 覆盖停用短路、优先级、equals/in、类型归一。
- `go test ./...`（含 `TEST_DATABASE_URL`）评估集成 ⑨～⑫ 通过。
- **人工评审：M3 已通过。**

---

### 阶段 6：前端页面（M4）

**我的提问**
```text
只做 M4：前端对接真实 Go API。禁止修改后端代码、禁止修改迁移。
Vite proxy /api → :8080；三个占位页重写；AppLayout 去掉（M1）和硬编码 /flags/1。
```

**AI 输出（节选）**
```text
文档已按「M0–M3 人工评审通过」对齐，M4 前端也一并收尾。

文档状态
文件	调整
README.md
M0–M3 已通过；M4 待评审；补前端页面、联调与 seed 评估两条链路
docs/PHASE_GATES.md
当前停在 M4
docs/ai-log.md
M3 改为已通过，补 M4 段落
docs/ai-development-log.md
去掉「仍停在 M3」；补真实 M4 / 踩坑，去掉占位 git log
M4 前端（npm run build 已通过）
列表 / 详情规则与历史 / 评估控制台，走 /api/v1
文案与后端一致：数字越小越先匹配、in 用 JSON 数组、停用恒 false
联调（seed）
后端 go run ./cmd/server，前端 npm run dev
评估控制台 Key=checkout_v2、环境=development
{"country":"CN"} → 命中 true
{"country":"US","plan":"free"} → 默认 false
环境改 production → 停用，恒 false
请人工点一遍后回复：M4 通过。
```

**我的决策与踩坑**
- 决策：规则 `in` 的 expectedValue 在前端用 `JSON.parse` 校验必须为字符串数组，与后端 TEXT 存 `["pro"]` 对齐，不用逗号分隔（避免值内逗号）。
- 409/400 文案与后端锁定句对齐：「该环境下 Key 已存在」「同一 Flag 内优先级不可重复」。

**结果验证**
- `npm run build`（tsc + vite）通过。
- 联调步骤见 README：seed `checkout_v2@development`，CN 命中 true；US+free 走默认 false。
- **人工评审：M4 已通过。**

---

### 阶段 7：M5 提交前收尾

**我的提问**
```text
只做 M5：对照笔试题目做提交前自查、补文档、整理 git、跑全量验收。
禁止新增功能、禁止改后端业务逻辑、禁止可选扩展。
README 补设计取舍 / 已完成 / 未完成 / 已知限制；M3/M4/文档分三次 commit。
```

**AI 输出（节选）**
```text
README 增加「设计取舍」「已知限制」独立章节。
git 拆分：feat(M3) 评估、feat(M4) 前端、docs(M5) 文档。
全量跑 go build/vet/fmt/test 与 npm run build/lint。
```

**我的决策与踩坑**
- 不做百分比灰度等可选扩展，只在「未完成/已知限制」如实列出。
- 需求分析「AI 输出」占位已用可验证的定稿结论摘要替代（不虚构原始全文）。

**结果验证**（2026-09-05 本机真跑，非虚构）
- `gofmt -w` 后 `gofmt -l .` 为空；`go build ./...`、`go vet ./...` 通过。
- `TEST_DATABASE_URL` 已设：`go test ./... -count=1` → `ok db / eval / http / service`。
- 未设 `TEST_DATABASE_URL`：http 集成测试 SKIP，不红。
- `npm run build`、`npm run lint` 通过。
- git 三次提交后见仓库 `git log --oneline`。

---

### 阶段 8：M6 配置版本 + 乐观锁

**我的提问**
```text
只做 M6（可选扩展，唯一一个）：配置版本 + 乐观锁。
禁止其它扩展；禁止改评估逻辑。WHERE version 必须用客户端快照，绝不能用 old.Version。
启停只 bump 不校验。409 VERSION_CONFLICT「数据已被他人修改，请刷新后重试」。
```

**AI 输出（节选）**
```text
0003_add_version 给 flags/rules 加 version DEFAULT 1。
UpdateFlag/UpdateRule：WHERE id AND version=$客户端；0 行 → ErrVersionConflict。
SetFlagEnabled：version=version+1，WHERE 仅 id。
前端编辑打开时记下 row.version，PATCH 带上；409 则 invalidateQueries。
```

**我的决策与踩坑**
- 既有 PATCH 集成测试必须带 version，否则 400；NotFound 的 PATCH 也要带，否则测不到 404。
- 回滚不做：乐观锁 ≠ 历史版本恢复。

**结果验证**（2026-09-05 本机真跑）
- `gofmt -l .` 空；`go build ./...`、`go vet ./...` 通过。
- 已设 `TEST_DATABASE_URL`：`go test ./... -count=1` → `ok db / eval / http / service`（含乐观锁 ①～④）。
- 未设：http 集成 SKIP，不红。
- `npm run build`、`npm run lint` 通过。
- **人工评审：M6 已通过。**

---

## 3. 操作日志（旁证）

### git 提交记录

以仓库内 `git log --oneline` 为准（M5 收尾后应能看到 feat(M3)、feat(M4)、docs(M5)）。**不要使用编造的 hash。**

### 关键命令与测试输出（节选）

```text
# M6 全量（backend，已设 TEST_DATABASE_URL）
ok  	featureflag/internal/db	0.381s
ok  	featureflag/internal/eval	0.358s
ok  	featureflag/internal/http	3.326s
ok  	featureflag/internal/service	0.365s

# 未设 TEST_DATABASE_URL
ok  	featureflag/internal/eval
ok  	featureflag/internal/db
ok  	featureflag/internal/service
ok  	featureflag/internal/http

# 前端
npm run build → tsc -b && vite build  exit 0
npm run lint  → oxlint  exit 0
```

Postgres：Docker Compose，宿主机 **5433**。

---

## 4. 阶段状态

- [x] M0–M6 人工评审通过
- [x] 关键踩坑：Gin `:id`/`:flagId` 冲突；Postgres 未启动导致集成测试失败

## 5. 需人工补齐清单（不代填虚构内容）

以下条目在 `ai-development-log.md` 中仍偏摘要或占位，评卷若要求「原始对话全文」，请人工从 Cursor 聊天记录粘贴，**不要编造**：

- [x] 阶段 1 占位已用定稿结论摘要替代
- [ ] 各阶段完整原始 prompt 可再从聊天导出备份到 `docs/chats/`（当前已有精简版）
- [x] M5 全量验收命令输出已写入本节「操作日志」（2026-09-05 真跑）
