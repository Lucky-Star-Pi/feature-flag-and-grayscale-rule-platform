# AI 辅助开发过程记录

## 门禁说明

按用户要求：**分阶段开发，每阶段评测 + 人工评审通过后才能进入下一阶段**。

---

## M0 脚手架（进行中 / 待评审）

### 我问了 AI 什么

- 按需求分析从零搭建 Feature Flag 平台
- 本机无 Go、Docker 守护进程曾未启动、5432 被其他容器占用时如何落地

### AI / 环境做了什么

- 将 Go 1.22 安装到 `D:\Tools\go`
- 启动 Docker Desktop；Postgres 映射到 **5433**（避开本机已有 `agent-postgres` 占用的 5432）
- 创建 `docker-compose.yml`、`backend/`、`frontend/`、迁移 SQL、README 启动说明

### 我（流程）做了什么决策 / 修正

- **新增强制规则**：每阶段必须评测 + 人工评审通过才能进下一阶段
- 提前写出的后续阶段草稿**不视为已完成**；当前只提交 M0 验收
- 默认 `DATABASE_URL` 端口改为 5433

### 验证（M0）

- `featureflag-pg` healthy，`pg_isready` 成功
- `GET http://127.0.0.1:8080/health` → `{"status":"ok"}`
- 前端 `npm run build`（修完 type-only import 后）应通过

### 待人工确认

请评审 M0 后回复：`M0 通过，进入 M1`
