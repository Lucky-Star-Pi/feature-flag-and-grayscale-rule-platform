# Feature Flag 与灰度规则平台

轻量 Feature Flag 平台（滴滴外包岗笔试题目 A）：React + Go + PostgreSQL。

## 开发门禁

**按阶段开发**：每阶段「自动化评测 + 人工评审」通过后，才能进入下一阶段。

| 阶段 | 内容 | 状态 |
|------|------|------|
| M0 | 脚手架可启动 | **已通过** |
| **M1** | 骨架 + 三表迁移 + DB 工具 + 前端占位 + healthz 测试 | **待人工评审** |
| M2 | Flag/规则 CRUD + 历史同事务（待门禁） | 未开始 |

## 技术选型（M1）

- HTTP：**Gin**（生态成熟、测试用 `httptest` 方便、与国内评卷环境熟悉）
- DB：`sqlx` + `pgx` stdlib；迁移：`golang-migrate`
- 前端：Vite + React + TS + React Router + Ant Design（仅占位页）

## 目录结构

```text
backend/
  cmd/server/main.go
  internal/
    config/          # 环境变量
    db/              # 连接、WithTx、MapUniqueViolation
    http/            # /healthz + 单测
    migrateutil/     # 迁移封装
  migrations/
    0001_init.up.sql / .down.sql
    0002_seed.up.sql / .down.sql
frontend/
  src/pages/         # 列表/详情/评估占位
docker-compose.yml   # Postgres 宿主机端口 5433
docs/
README.md
```

## 数据模型要点（M1）

- `flags`：`UNIQUE (key, environment)` —— **数据库层**保证同环境 Key 唯一
- `rules`：`UNIQUE (flag_id, priority)` —— 重复优先级 **拒绝**（DB 兜底；应用层 400 在 M2）
- `history`：`flag_id` 可空 FK；字段 `operation_type` / `operator` / `summary`

## 启动命令

```powershell
# 0. 环境
$env:PATH = "D:\Tools\go\bin;D:\nodejs;" + $env:PATH
$env:GOROOT = "D:\Tools\go"
$env:GOPATH = "D:\Tools\gopath"
$env:GOPROXY = "https://goproxy.cn,direct"

# 1. Postgres
cd "D:\桌面\陈凯昊项目提交（滴滴）"
$env:COMPOSE_PROJECT_NAME = "featureflag"
docker compose -p featureflag up -d
docker exec featureflag-pg pg_isready -U flaguser -d featureflag

# 2. 后端（自动 migrate + /healthz）
cd backend
$env:DATABASE_URL = "postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable"
$env:MIGRATIONS_PATH = "file://migrations"
$env:HTTP_ADDR = ":8080"
go run ./cmd/server

# 3. 前端
cd ..\frontend
npm install
npm run dev
```

验证：

- `GET http://127.0.0.1:8080/healthz` → `{"status":"ok"}`
- 浏览器打开 Vite 地址，可访问列表/详情/评估占位页

## M1 测试命令

```powershell
cd "D:\桌面\陈凯昊项目提交（滴滴）\backend"
$env:GOPROXY = "https://goproxy.cn,direct"
go test ./internal/http/ ./internal/db/ -count=1
```

可选：检查种子与唯一约束（需 psql 或 docker exec）：

```powershell
docker exec -i featureflag-pg psql -U flaguser -d featureflag -c "\d+ flags"
docker exec -i featureflag-pg psql -U flaguser -d featureflag -c "SELECT key, environment FROM flags;"
docker exec -i featureflag-pg psql -U flaguser -d featureflag -c "INSERT INTO flags(name,key,environment,enabled,default_value) VALUES ('x','checkout_v2','development',true,false);"
# 期望：ERROR unique violation on flags_key_environment_uk
```

## 后续集成测试说明（本阶段不做完整实现）

M2+ 将用**真 PostgreSQL**（`TEST_DATABASE_URL`，可选 Docker）验证：

1. 同环境重复 Key → 23505 → 业务冲突
2. 不同环境同 Key 允许
3. `WithTx`：业务写 + history 任一失败整体回滚

本阶段不引入 testcontainers，避免 Windows/无 Docker 场景评卷失败。

## M1 评测清单（请人工勾选）

- [ ] 迁移后存在 `flags` / `rules` / `history`，且 `flags` 有 `UNIQUE (key, environment)`
- [ ] seed 有 demo 数据；同环境重复 Key 插入失败
- [ ] `go test ./internal/http/ ./internal/db/` 通过
- [ ] `/healthz` 返回 ok；前端三占位页可访问
- [ ] 仓库中**无**完整 Flag CRUD API（留给 M2）

**通过后请回复：`M1 通过，进入 M2`**

## 下一步 M2（先说明，本阶段不实现）

- Flag 列表/新建/编辑/启停 API + 与 history 同事务
- 规则增删改 + 重复 priority 应用层 400
- 把 23505 映射为 HTTP 409
- 集成测试覆盖唯一约束与事务回滚
