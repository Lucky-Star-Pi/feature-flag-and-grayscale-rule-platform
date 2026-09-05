-- M6: 配置版本。初始/存量行 DEFAULT 1。
-- 编辑（PATCH Flag/Rule）用客户端快照做乐观锁；启停只 bump、不校验。

ALTER TABLE flags ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE rules ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
