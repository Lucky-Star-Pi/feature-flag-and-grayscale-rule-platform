# Feature Flag 与灰度规则平台

轻量 Feature Flag 平台（滴滴外包岗笔试题目 A）：React + Go + PostgreSQL。用于按环境控制功能开关，并按用户属性做有序规则匹配。

## 开发门禁

| 阶段 | 内容 | 状态 |
|------|------|------|
| M0 | 脚手架可启动 | **已通过** |
| M1 | 骨架 + 三表迁移 + DB 工具 + 前端占位 + healthz | **已通过** |
| M2 | Flag/规则 CRUD + 历史同事务 + 真库集成测试 | **已通过** |
| M3 | 规则评估纯函数 + `POST /api/v1/evaluate` | **已通过** |
| M4 | 前端对接真实 Go API | **已通过** |
| M5 | 提交前自查、文档与 git 整理、全量验收 | **已通过** |
| **M6** | 配置版本 + 乐观锁（唯一可选扩展） | **已通过** |

## 启动方式

少量分步命令即可本地复现（题面不强制 Docker 一键）。

下面 PATH / GOROOT / GOPATH 是作者 Windows 本机示例；若 Go、Node 已加入 PATH 可直接省略这几行，请按本机实际安装路径替换。`GOPROXY` 仅大陆网络加速用，非必需。宿主机端口 **5433** 是因为本机 5432 已被占用；若改端口，须同步改 `docker-compose.yml` 的 `ports` 以及 `DATABASE_URL` / `TEST_DATABASE_URL`。

```powershell
$env:PATH = "D:\Tools\go\bin;D:\nodejs;" + $env:PATH
$env:GOROOT = "D:\Tools\go"
$env:GOPATH = "D:\Tools\gopath"
$env:GOPROXY = "https://goproxy.cn,direct"

# 1. Postgres（宿主机端口 5433，避开本机已占用的 5432）
cd "D:\桌面\陈凯昊项目提交（滴滴）"
$env:COMPOSE_PROJECT_NAME = "featureflag"
docker compose -p featureflag up -d

# 2. 后端（启动时自动 migrate，含 0003 version 列）
cd backend
$env:DATABASE_URL = "postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable"
$env:MIGRATIONS_PATH = "file://migrations"
$env:HTTP_ADDR = ":8080"
go run ./cmd/server

# 3. 前端（Vite 把 /api 代理到 :8080）
cd ..\frontend
npm install
npm run dev
```

浏览器打开 Vite 地址（默认 `http://127.0.0.1:5173`）。健康检查：`GET http://127.0.0.1:8080/healthz`。

用 seed **`checkout_v2` + `development`** 在评估控制台复现：

1. **命中 true**：`{"country":"CN"}`（或 `"plan":"pro"`）→ `value=true`，`reason=matched`
2. **走默认值**：`{"country":"US","plan":"free"}` → `value=false`，`reason=default`
3. **停用短路**：同一 Key 换 `production`（seed 停用）→ `value=false`，`reason=disabled`

后端测试：

```powershell
cd backend
$env:TEST_DATABASE_URL = "postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable"
go test ./... -count=1
```

不连库（集成测试 Skip）：不要设置 `TEST_DATABASE_URL`，然后  
`go test ./internal/eval/ ./internal/db/ ./internal/service/ ./internal/http/ -count=1`

## 已锁定语义（代码 / README / 界面三处一致）

- **优先级**：数字越小越高；`priority ASC, id ASC`。
- **重复优先级**：拒绝 → `400 PRIORITY_CONFLICT`。
- **同环境 Key 唯一**：DB `UNIQUE(key, environment)`，23505 → `409 KEY_CONFLICT`。
- **历史**：写操作与 history 同一 `WithTx`；操作者固定 `local-admin`。
- **评估**：停用 → `reason=disabled` 恒 false；首条命中 → `matched`；否则 `default`。equals 精确字符串化比较；`in` 解析 JSON 字符串数组，失败或 `[]` 跳过；属性缺失 / null / 对象 / 数组跳过。
- **乐观锁**：编辑 Flag/规则必须带客户端看到的 `version`；`WHERE version=?` 用该快照，**禁止**用服务端刚读到的值。启停只 bump、不校验。冲突 → `409 VERSION_CONFLICT`「数据已被他人修改，请刷新后重试」。

## 设计取舍

- **Key 唯一只信任 DB**：`UNIQUE(key, environment)`，直接 INSERT，捕获 PostgreSQL `23505`，而不是「先查后插」。两个并发 POST 都会通过应用层检查，只有库约束能拦住。
- **重复优先级用拒绝，不用二级排序**：策略简单、结果确定；应用层先 400，DB UNIQUE 兜底。列表与评估仍按 `id ASC` 做同分防御排序。
- **评估做成 `internal/eval` 纯函数**：无 IO，表驱动单测不依赖数据库；service 只负责查 Flag/规则再调用。
- **`history.flag_id` 为 `ON DELETE SET NULL`**：本题不做删 Flag，但即便误删，历史摘要仍可留存。
- **`in` 的 expectedValue 存 TEXT JSON 数组**：不单独建成员表，写入/展示简单；前端用 `JSON.parse` 校验必须是字符串数组。
- **固定操作者 `local-admin`**：题面不要求登录，避免把时间花在无分值鉴权上。
- **集成测试不用 testcontainers**：`TEST_DATABASE_URL` + 未设置则 `t.Skip`，避免 Windows/评卷机无 Docker 时整套变红。
- **HTTP 用 Gin、DB 用 sqlx 显式 SQL**：事务边界和 UNIQUE 错误对评卷人可见；不用 GORM 隐藏 SQL。
- **前端只走相对路径 `/api/v1`**：开发靠 Vite proxy，不写死后端地址。
- **乐观锁只锁编辑，启停不校验版本**：UpdateFlag / UpdateRule 必须带客户端快照 `version`，UPDATE 用 `WHERE ... AND version=$客户端version`，影响 0 行 → `409 VERSION_CONFLICT`。启用/停用是单布尔幂等切换，只 `version=version+1`，last-write-wins。铁律：`WHERE version=?` **绝不能**用服务端 `GetFlag/GetRule` 刚读到的 `old.Version`（那是「读后立刻写」，锁永远匹配）；`old` 只用于历史摘要。
- **回滚 ≠ 乐观锁**：本扩展只防并发覆盖并递增版本号，不做历史版本恢复/一键回滚。

## 数据模型（摘要）

- `flags`：`UNIQUE(key, environment)`；`version BIGINT NOT NULL DEFAULT 1`
- `rules`：`UNIQUE(flag_id, priority)`；`flag_id` CASCADE；同样有 `version`
- `history`：`flag_id` 可空 SET NULL；`operation_type` / `operator` / `summary`

## API 契约（摘要）

Base `/api/v1`，错误体 `{"error":{"code","message"}}`。

- Flag：`GET/POST /flags`，`GET/PATCH /flags/:id`，`POST .../enable|disable`
- 规则：`POST/PATCH/DELETE /flags/:id/rules[/:ruleId]`
- 历史：`GET /flags/:id/history`
- 评估：`POST /evaluate` → `{value, matched, matchedRule, reason}`，`reason` ∈ `disabled|matched|default`

错误码：`KEY_CONFLICT` 409、`VERSION_CONFLICT` 409（「数据已被他人修改，请刷新后重试」）、`PRIORITY_CONFLICT` 400、`NOT_FOUND` 404（评估文案「Flag 不存在」）、`INVALID_INPUT` 400。

## 前端页面

- `/` 列表：筛选、新建/编辑、启停、进详情
- `/flags/:id` 详情：规则 CRUD、操作历史
- `/evaluate` 评估控制台

## 已完成

- Flag 列表 / 搜索筛选 / 新建 / 编辑 / 启停 / 详情
- 有序规则增删改；优先级拒绝重复；数字越小越先匹配
- 在线评估控制台（真实 Go API）；停用短路、命中、默认值、JSON 错误、Flag 不存在反馈
- 操作历史与业务变更同事务；固定操作者 `local-admin`
- 数据库迁移 + seed；同环境 Key 唯一落在 DB
- 评估核心逻辑自动化测试 + 真库集成测试（唯一约束、事务、评估接口）
- 前端对接真实 API（loading / 空状态 / 错误反馈）
- 配置版本 + 乐观锁：编辑带客户端 `version`；启停只 bump；冲突 409 并刷新列表/详情

## 未完成

题面可选扩展，**无独立分值，本项目不做**：

- 登录与角色权限
- 稳定百分比灰度
- 配置回滚（回滚 ≠ 乐观锁：乐观锁只防并发覆盖，不恢复历史版本）
- 规则拖拽排序
- 环境间配置复制或发布
- 前端自动化测试或在线部署

## 已知限制

- 无登录/角色权限（题面明确不要求）
- 列表无分页（演示数据量可接受）
- 无配置回滚、无百分比灰度、无环境间复制（见未完成）
- 前端无自动化测试（可选扩展未做）
- 评估每次实时查库、无缓存（量小可接受）
- 启动需分步命令 + 本机 Docker 起 Postgres（未做一键 Compose 含前后端；题面不强制 Docker）
- Postgres 映射 **5433**，因本机 5432 可能已被其它容器占用

## 目录结构

```text
backend/           Go API、migrations、测试
frontend/          Vite React，相对路径调 /api/v1
docker-compose.yml 仅 Postgres
docs/              AI 过程记录与阶段门禁
README.md
```
