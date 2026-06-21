package postgresconn

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestBuildDSN(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "builds postgres dsn from config",
			cfg: Config{
				Host:     "db.internal",
				Port:     5433,
				User:     "cakapp",
				Password: "secret",
				Database: "cakapp_dev",
				SSLMode:  "disable",
			},
			want: "host=db.internal port=5433 user=cakapp password=secret dbname=cakapp_dev sslmode=disable",
		},
		{
			name: "quotes password containing spaces",
			cfg: Config{
				Host:     "db.internal",
				Port:     5432,
				User:     "cakapp",
				Password: "my secret",
				Database: "cakapp_dev",
				SSLMode:  "disable",
			},
			want: "host=db.internal port=5432 user=cakapp password='my secret' dbname=cakapp_dev sslmode=disable",
		},
		{
			name: "escapes password containing quotes and backslashes",
			cfg: Config{
				Host:     "db.internal",
				Port:     5432,
				User:     "cakapp",
				Password: "pa'ss\\word",
				Database: "cakapp_dev",
				SSLMode:  "disable",
			},
			want: "host=db.internal port=5432 user=cakapp password='pa\\'ss\\\\word' dbname=cakapp_dev sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, buildDSN(tt.cfg))
		})
	}
}

func TestNormalizeConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want Config
	}{
		{
			name: "applies defaults when pool and ping settings are unset",
			cfg:  Config{},
			want: Config{
				MaxIdleConns:    5,
				MaxOpenConns:    25,
				ConnMaxLifetime: 5 * time.Minute,
				PingTimeout:     5 * time.Second,
			},
		},
		{
			name: "preserves explicit values",
			cfg: Config{
				Host:            "db.internal",
				Port:            5432,
				User:            "postgres",
				Password:        "secret",
				Database:        "cakapp",
				SSLMode:         "require",
				MaxIdleConns:    7,
				MaxOpenConns:    13,
				ConnMaxLifetime: 7 * time.Minute,
				PingTimeout:     2 * time.Second,
			},
			want: Config{
				Host:            "db.internal",
				Port:            5432,
				User:            "postgres",
				Password:        "secret",
				Database:        "cakapp",
				SSLMode:         "require",
				MaxIdleConns:    7,
				MaxOpenConns:    13,
				ConnMaxLifetime: 7 * time.Minute,
				PingTimeout:     2 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeConfig(tt.cfg))
		})
	}
}

func TestWithGormConfig(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *gorm.Config
		assert func(t *testing.T, got *gorm.Config)
	}{
		{
			name: "keeps provided gorm config and does not override now func",
			cfg: &gorm.Config{
				NowFunc: func() time.Time {
					return time.Unix(1700000000, 0)
				},
				Logger: gormlogger.Default.LogMode(gormlogger.Warn),
			},
			assert: func(t *testing.T, got *gorm.Config) {
				require.NotNil(t, got)
				require.NotNil(t, got.Logger)
				assert.Equal(t, time.Unix(1700000000, 0), got.NowFunc())
			},
		},
		{
			name: "sets utc now func when caller did not provide one",
			cfg:  &gorm.Config{},
			assert: func(t *testing.T, got *gorm.Config) {
				require.NotNil(t, got)
				require.NotNil(t, got.NowFunc)
				assert.Equal(t, time.UTC, got.NowFunc().Location())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applied := applyOptions(tt.cfg, WithGormConfig(tt.cfg))
			tt.assert(t, applied)
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		assertError func(t *testing.T, err error)
	}{
		{
			name: "returns error when postgres is unreachable",
			cfg: Config{
				Host:        "127.0.0.1",
				Port:        1,
				User:        "postgres",
				Password:    "postgres",
				Database:    "postgres",
				SSLMode:     "disable",
				PingTimeout: time.Second,
			},
			assertError: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to ping database")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewClient(tt.cfg)

			tt.assertError(t, err)
			assert.Nil(t, db)
		})
	}
}
