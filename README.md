# Feature Flag 与灰度规则平台

轻量 Feature Flag 平台（滴滴外包岗笔试题目 A）：React + Go + PostgreSQL。

## 开发门禁

**按阶段开发**：每阶段「自动化评测 + 人工评审」通过后，才能进入下一阶段。

| 阶段 | 内容 | 状态 |
|------|------|------|
| M0 | 脚手架可启动 | **已通过** |
| M1 | 骨架 + 三表迁移 + DB 工具 + 前端占位 + healthz | **已通过** |
| **M2** | Flag/规则 CRUD + 历史同事务 + 真库集成测试 | **待人工评审** |
| M3 | 规则评估纯函数 + `/evaluate` | 未开始（待门禁） |
| M4 | 前端对接真实 API | 未开始 |

## 已锁定语义（前后端 / README / 代码三处一致）

- **优先级方向**：数字越小优先级越高；规则按 `priority ASC, id ASC` 返回。M3 评估将沿用同一顺序。
- **重复优先级**：拒绝。应用层先查 → `400 PRIORITY_CONFLICT`；DB `UNIQUE(flag_id, priority)` 的 23505 同样映射为 400。
- **同环境 Key 唯一**：只信任 DB `UNIQUE(key, environment)`。直接 INSERT，捕获 23505 → `409 KEY_CONFLICT`。不同环境允许相同 Key。
- **历史原子性**：创建/编辑/启停/规则增删改均在 `db.WithTx` 内同时写入业务行与 history；任一失败整体回滚。
- **操作者**：固定 `local-admin`，不信任请求体。
- **本阶段不做**：`/evaluate`、`internal/eval`、前端页面改造。

## 技术选型

- HTTP：**Gin**；DB：`sqlx` + `pgx`；迁移：`golang-migrate`
- 前端：Vite + React + TS（M2 **不改页面**，仍为占位）

## 目录结构

```text
backend/
  cmd/server/main.go          # 注入 service，挂 /healthz + /api/v1
  internal/
    config/                   # HTTP_ADDR / DATABASE_URL / MIGRATIONS_PATH
    db/                       # Open / WithTx / MapUniqueViolation
    model/                    # Flag / Rule / History
    store/                    # sqlx 数据访问（写操作走 *sqlx.Tx）
    service/                  # 校验 + WithTx 编排
    http/                     # 路由、writeError、集成测试
    migrateutil/
    eval/                     # 留给 M3
  migrations/                 # 0001 表结构 + 0002 seed（M2 未改）
frontend/                     # M1 占位页（M2 未改）
```

## API 契约（M2）

Base：`/api/v1`。统一错误体：

```json
{"error":{"code":"KEY_CONFLICT","message":"该环境下 Key 已存在"}}
```

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/flags` | 列表。查询：`key` 模糊、`environment` 精确、`enabled=true/false` |
| POST | `/api/v1/flags` | 新建。201。body: name, key, environment, enabled?, defaultValue |
| GET | `/api/v1/flags/:id` | 详情，含按 priority 升序的 rules |
| PATCH | `/api/v1/flags/:id` | 编辑 name / defaultValue（不可改 key/environment） |
| POST | `/api/v1/flags/:id/enable` | 启用 |
| POST | `/api/v1/flags/:id/disable` | 停用 |
| POST | `/api/v1/flags/:id/rules` | 新增规则 |
| PATCH | `/api/v1/flags/:id/rules/:ruleId` | 编辑规则 |
| DELETE | `/api/v1/flags/:id/rules/:ruleId` | 删除，204 |
| GET | `/api/v1/flags/:id/history` | 历史，created_at 降序 |
| GET | `/healthz` | 健康检查 |

错误码：

- `KEY_CONFLICT` → 409（同环境 Key 冲突，来自 23505）
- `PRIORITY_CONFLICT` → 400（重复优先级，应用层或 23505）
- `NOT_FOUND` → 404
- `INVALID_INPUT` → 400（非法 environment / operator / priority<0 / 缺字段）

规则 body 示例：

```json
{"attribute":"country","operator":"equals","expectedValue":"CN","returnValue":true,"priority":0}
```

`in` 的 `expectedValue` 本阶段存 TEXT（如 `["pro","enterprise"]`）；匹配语义在 M3。

## 启动命令

```powershell
$env:PATH = "D:\Tools\go\bin;D:\nodejs;" + $env:PATH
$env:GOROOT = "D:\Tools\go"
$env:GOPATH = "D:\Tools\gopath"
$env:GOPROXY = "https://goproxy.cn,direct"

cd "D:\桌面\陈凯昊项目提交（滴滴）"
$env:COMPOSE_PROJECT_NAME = "featureflag"
docker compose -p featureflag up -d

cd backend
$env:DATABASE_URL = "postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable"
$env:MIGRATIONS_PATH = "file://migrations"
$env:HTTP_ADDR = ":8080"
go run ./cmd/server
```

验证：`GET http://127.0.0.1:8080/healthz`；`GET http://127.0.0.1:8080/api/v1/flags`

## 测试命令

未设置 `TEST_DATABASE_URL` 时，集成测试会 Skip；单测仍会跑。

```powershell
cd "D:\桌面\陈凯昊项目提交（滴滴）\backend"
$env:PATH = "D:\Tools\go\bin;" + $env:PATH
$env:GOROOT = "D:\Tools\go"
$env:GOPATH = "D:\Tools\gopath"
$env:GOPROXY = "https://goproxy.cn,direct"
$env:TEST_DATABASE_URL = "postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable"
go test ./... -count=1
```

仅单测（不连库）：

```powershell
go test ./internal/db/ ./internal/service/ ./internal/http/ -count=1 -short
```

（集成测试在 `internal/http` 的 `api_int_test.go`，不看 `-short`，只看环境变量。）

## M2 评测清单（请人工勾选）

- [ ] `go test ./...` 在设置 `TEST_DATABASE_URL` 后全部通过
- [ ] 同环境重复 Key → 409，且无多余 history
- [ ] 不同环境同 Key → 201
- [ ] 创建成功后 flags + history(CREATE_FLAG) 都有
- [ ] 重复 priority → 400，无 history 残留
- [ ] 编辑/启停 history 的 summary 可读
- [ ] 不存在的 flag 详情/规则操作 → 404
- [ ] 无 `/evaluate`、前端仍为占位页

**通过后请回复：`M2 通过，进入 M3`**

## 下一步 M3（本阶段不实现）

- `internal/eval` 纯函数：停用短路、priority 升序首条命中、默认值兜底、equals/in、属性缺失/类型
- `POST /api/v1/evaluate`
- 表驱动单测覆盖核心场景 ①～⑤⑪
