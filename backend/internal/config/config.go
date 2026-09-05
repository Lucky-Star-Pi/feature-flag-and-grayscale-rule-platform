package config

import "os"

// Config 由环境变量加载，便于本地与测试复用。
type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	MigrationsPath  string
}

func Load() Config {
	return Config{
		HTTPAddr:       envOr("HTTP_ADDR", ":8080"),
		DatabaseURL:    envOr("DATABASE_URL", "postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable"),
		MigrationsPath: envOr("MIGRATIONS_PATH", "file://migrations"),
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
