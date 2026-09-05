# AI 辅助开发过程记录

## 门禁

分阶段开发；每阶段评测 + 人工评审通过后才能进入下一阶段。

---

## M0 脚手架 — 已通过

- Go 装到 `D:\Tools\go`；Postgres Compose 映射 **5433**
- 健康检查路径定为 `/healthz`

---

## M1 骨架 + 数据模型 — 已通过

- 三表迁移 + `WithTx` + `MapUniqueViolation` + 前端占位
- 按评审要求删除提前写的完整 CRUD

---

## M2 Flag/规则 CRUD + 历史同事务 — 已通过

- 复用 `db.WithTx` / `MapUniqueViolation`；23505 → Key 409、priority 400
- 写操作与 history 同事务；真库集成测试 ①～⑥
- 规则路由统一 `:id`（Gin 不允许混用 `:flagId`）

---

## M3 规则评估 — 已通过

- `internal/eval` 纯函数：停用短路 `disabled`、命中 `matched`、默认 `default`
- `POST /api/v1/evaluate`；Flag 不存在 404 message「Flag 不存在」
- `go test ./internal/eval/` 与 http 集成 ⑨～⑫ 通过

---

## M4 前端对接真实 API — 已通过

- 只改前端；相对路径 `/api/v1`；列表 / 详情 / 评估
- 文案与后端一致：数字越小越先匹配；`in` 为 JSON 字符串数组；停用恒 false
- `npm run build` 通过

---

## M5 收尾验收

### 提示词要求

- 不改业务逻辑、不改迁移、不新增 API、不做可选扩展
- README 补齐：启动方式、设计取舍、已完成、未完成、已知限制
- 更新过程文档；把未提交的 M3/M4/文档拆成清晰 git 历史
- 真跑：`go build/vet/fmt/test`、`npm run build/lint`

### 做了什么

- README 增加「设计取舍」「已知限制」独立章节
- `PHASE_GATES` / `ai-log` 将 M4 标为已通过，M5 为收尾
- 核对 `ai-development-log.md`：列出仍需人工补齐的项（不虚构）
- 按路径拆分提交 M3 / M4 / M5

### 验收结果（2026-09-05 本机真跑）

- Postgres：`featureflag-pg` 已在跑，`pg_isready` 通过（宿主机 5433）。
- `gofmt -l .`：初检有 `config.go` / `api_int_test.go` / `service.go` 未对齐；已 `gofmt -w`（仅空白，无业务改动），再检为空。
- `go build ./...`：通过
- `go vet ./...`：通过（无警告）
- 设置 `TEST_DATABASE_URL` 后 `go test ./... -count=1`：`db` / `eval` / `http` / `service` 全绿
- 不设 `TEST_DATABASE_URL`：`eval`/`db`/`service` 单测 PASS；http 集成测试全部 SKIP（`TEST_DATABASE_URL 未设置`），`TestHealthz` PASS，不红
- `npm run build`：`tsc -b && vite build` 通过（vite 提示主包 >500kB，无害）
- `npm run lint`：`oxlint` 通过

### 待人工确认

回复：`M5 通过，可提交`

---

## M6 配置版本 + 乐观锁 — 已通过

### 提示词要求

- 唯一可选扩展：配置版本 + 乐观锁；禁止百分比灰度/登录/拖拽/前端 E2E
- 不改评估逻辑与既有业务语义
- `flags`/`rules` 加 `version`；编辑必须带客户端快照；启停只 bump 不校验
- `WHERE version=?` 绝不能用服务端刚读的 `old.Version`

### 实现要点

- 迁移 `0003_add_version`（启动 `migrate.Up` 自动 apply）
- store：UpdateFlag/UpdateRule 用客户端 version；0 行 → `db.ErrVersionConflict` → 409 `VERSION_CONFLICT`「数据已被他人修改，请刷新后重试」
- SetFlagEnabled：`version=version+1`，WHERE 仅 `id`
- 前端编辑 Modal 打开时记录 `row.version`，PATCH 带上；409 则提示并 invalidateQueries

### 验证命令

```powershell
cd backend
$env:TEST_DATABASE_URL = "postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable"
go test ./... -count=1
# 不连库：不要设置 TEST_DATABASE_URL
go test ./internal/eval/ ./internal/db/ ./internal/service/ ./internal/http/ -count=1
cd ..\frontend
npm run build
npm run lint
```

### 踩坑记录

- 既有 `TestUpdateEnableHistorySummary`、`TestRulesCRUD_AndDetailOrder` 的 PATCH 必须带 `version`，否则 400。
- `TestNotFound` 的 PATCH 也要带 `version`，否则在 GetFlag 之前就因「version 必填」变成 400，测不到 404。
- 若把 `old.Version` 写入 WHERE，陈旧客户端 version 永远对得上，乐观锁形同虚设。

### 人工评审

**已通过**（口令：`M6 通过`）
