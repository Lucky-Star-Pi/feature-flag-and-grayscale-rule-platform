# AI 辅助开发过程记录

## 门禁

分阶段开发；每阶段评测 + 人工评审通过后才能进入下一阶段。

---

## M0 脚手架 — 已通过

- Go 装到 `D:\Tools\go`；Postgres Compose 映射 **5433**（避开本机 5432）
- `/health` 冒烟通过

---

## M1 骨架 + 数据模型 — 待评审

### 我问了 AI / 提示词要求

- 只做 M1：骨架、迁移三表、DB 工具、前端占位、healthz 测试
- **不要**提前实现完整 CRUD
- `UNIQUE(key, environment)` 必须落在 DB；事务与 23505 映射先封装给 M2

### 做了什么

- 按 M1 提示词**收敛范围**：删除提前写的 service/store/eval/完整 handlers 与前端 API 页
- 迁移对齐表名：`flags` / `rules` / `history`（`flag_id` 可空）
- 新增 `config`、`db.WithTx`、`db.MapUniqueViolation`、`/healthz` + 单测
- 前端改为三占位页

### 验证命令

```text
go test ./internal/http/ ./internal/db/ -count=1
GET /healthz
docker exec ... psql 检查 UNIQUE 与 seed
```

### 待人工确认

回复：`M1 通过，进入 M2`
