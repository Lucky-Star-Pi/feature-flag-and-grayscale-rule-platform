# Feature Flag 与灰度规则平台

轻量 Feature Flag 平台（滴滴外包岗笔试题目 A）：React + Go + PostgreSQL。

## 开发门禁（强制）

**按阶段开发**：每个里程碑结束后必须完成「自动化评测 + 人工评审」通过后，才能进入下一阶段。未获评审通过不得继续实现后续功能。

| 阶段 | 内容 | 本阶段评测 | 状态 |
|------|------|------------|------|
| **M0** | 脚手架：Postgres、Go 服务可启动、Vite 前端可启动、README 启动命令 | 见下方「M0 评测清单」 | **待人工评审** |
| M1 | flags 表 UNIQUE + CRUD/409 | 集成测试 | 未开始（待 M0 通过） |
| M2 | eval 纯函数单测 | `go test ./internal/eval` | 未开始（待门禁） |
| M3 | POST /evaluate | 接口错误码 | 未开始 |
| M4 | 规则 CRUD + history 同事务 | 集成测试 | 未开始 |
| M5 | 启停与编辑 | 停用恒 false | 未开始 |
| M6 | 四页前端调真实 API | 手测主路径 | 未开始 |
| M7 | README 定稿 + AI 过程记录 | 文档齐套 | 未开始 |

> 说明：仓库内可能已有后续阶段草稿文件；**在 M0 人工评审通过前，不以这些草稿作为「已完成」交付**，下一阶段以评审结论为准再继续。

## 已锁定的关键业务决策（全文一致）

- 优先级：数字越小越先匹配（`ORDER BY priority ASC, id ASC`）
- 重复优先级：拒绝（400 / DB UNIQUE）
- 停用：`enabled=false` 时恒返回 `false`，不评估规则
- 固定操作者：`local-admin`

## 技术栈

- 后端：Go、Gin、sqlx、pgx、golang-migrate、testify
- 前端：Vite、React、TypeScript、Ant Design、TanStack Query、React Router
- 数据库：PostgreSQL 16（Docker Compose，宿主机端口 **5433**，避免与本机已有 5432 冲突）

## 启动方式（M0）

### 0. 环境变量建议

```powershell
$env:PATH = "D:\Tools\go\bin;D:\nodejs;" + $env:PATH
$env:GOROOT = "D:\Tools\go"
$env:GOPATH = "D:\Tools\gopath"
$env:GOPROXY = "https://goproxy.cn,direct"
```

### 1. 启动 PostgreSQL

```powershell
cd "D:\桌面\陈凯昊项目提交（滴滴）"
$env:COMPOSE_PROJECT_NAME = "featureflag"
docker compose -p featureflag up -d
docker exec featureflag-pg pg_isready -U flaguser -d featureflag
```

连接串：

```text
postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable
```

### 2. 启动后端

```powershell
cd "D:\桌面\陈凯昊项目提交（滴滴）\backend"
$env:DATABASE_URL = "postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable"
$env:MIGRATIONS_PATH = "file://migrations"
$env:HTTP_ADDR = ":8080"
go run ./cmd/server
```

健康检查：`GET http://127.0.0.1:8080/health` → `{"status":"ok"}`

### 3. 启动前端

```powershell
cd "D:\桌面\陈凯昊项目提交（滴滴）\frontend"
npm install
npm run dev
```

浏览器打开 Vite 提示的地址（默认 `http://127.0.0.1:5173`）。开发代理：`/api` → `:8080`。

## M0 评测清单（请人工勾选）

- [ ] `docker compose up -d` 后容器 `featureflag-pg` 为 healthy，`pg_isready` 成功
- [ ] `go run ./cmd/server` 能启动，迁移成功打印 version
- [ ] `GET /health` 返回 ok
- [ ] `npm run dev` 前端可打开，无构建错误
- [ ] README 启动命令可按本机路径复现

**请确认 M0 通过后，回复「M0 通过，进入 M1」**，再开始 M1 开发与评测。

## 目录结构

```text
backend/          Go API
frontend/         React 应用
docker-compose.yml
docs/             过程记录与阶段说明
README.md
```

## 已完成 / 未完成（相对整题）

- **本阶段（M0）目标**：脚手架可运行（进行中，待评审）
- **整题未完成**：Flag 管理验收、规则、评估控制台、历史事务、完整 README 语义定稿、AI 过程记录定稿等均待各阶段门禁通过后交付
