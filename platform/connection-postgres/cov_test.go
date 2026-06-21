package postgresconn

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCov_EscapeDSNValue_Empty(t *testing.T) {
	require.Equal(t, "''", escapeDSNValue(""))
}

func TestCov_NewClient_InvalidSSLModeFailsToOpen(t *testing.T) {
	_, err := NewClient(Config{
		Host:     "127.0.0.1",
		Port:     5432,
		User:     "u",
		Password: "p",
		Database: "d",
		SSLMode:  "definitely-not-a-valid-sslmode",
	})
	require.Error(t, err)
}

// TestCov_NewClient_Success covers the happy path of NewClient, which requires a
// reachable Postgres. It is configurable via the standard PG* env vars and skips
// when no server answers, so the default `go test ./...` stays green everywhere.
func TestCov_NewClient_Success(t *testing.T) {
	cfg := Config{
		Host:     envOr("PGHOST", "127.0.0.1"),
		Port:     5432,
		User:     envOr("PGUSER", "postgres"),
		Password: envOr("PGPASSWORD", "postgres"),
		Database: envOr("PGDATABASE", "postgres"),
		SSLMode:  envOr("PGSSLMODE", "disable"),
	}

	db, err := NewClient(cfg, WithGormConfig(&gorm.Config{}))
	if err != nil {
		t.Skipf("no reachable Postgres for success-path coverage: %v", err)
	}

	require.NotNil(t, db)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, sqlDB.PingContext(ctx))
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
