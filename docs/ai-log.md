# AI 辅助开发过程记录

## 门禁

分阶段开发；每阶段评测 + 人工评审通过后才能进入下一阶段。

---

## M0 脚手架 — 已通过

- Go 装到 `D:\Tools\go`；Postgres Compose 映射 **5433**
- `/healthz` 冒烟通过（M1 将健康检查路径定为 `/healthz`）

---

## M1 骨架 + 数据模型 — 已通过

- 三表迁移 + `WithTx` + `MapUniqueViolation` + 前端占位
- 按评审要求删除提前写的完整 CRUD

---

## M2 Flag/规则 CRUD + 历史同事务 — 待评审

### 提示词要求

- 复用 M1 的 `db.WithTx` / `MapUniqueViolation`，禁止重造
- 同环境 Key 唯一只信任 DB 23505 → 409
- 重复 priority → 400（应用层 + DB 兜底）
- 所有写操作与 history 同一事务
- 真 PostgreSQL 集成测试；未设 `TEST_DATABASE_URL` 则 Skip
- 不实现评估、不改前端

### 实现要点

- `NewRouter(svc)` 注入 service；`cmd/server/main.go` 已更新
- 规则路由使用 `:id`（Gin 不允许 `/flags/:id` 与 `/flags/:flagId/rules` 并存）
- 修正注释：`ErrRulePriorityConflict` 映射 **400** 而非 409
- 集成测试覆盖题目要求的 ①～⑥，另补非法 environment、规则 CRUD 排序

### 验证

```text
$env:TEST_DATABASE_URL = "postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable"
go test ./... -count=1
```

### 待人工确认

回复：`M2 通过，进入 M3`
