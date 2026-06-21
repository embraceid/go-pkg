package postgresconn

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	PingTimeout     time.Duration
}

type Option func(*options)

type options struct {
	gormConfig *gorm.Config
}

func WithGormConfig(cfg *gorm.Config) Option {
	return func(opts *options) {
		opts.gormConfig = cloneGormConfig(cfg)
	}
}

func NewClient(cfg Config, opts ...Option) (*gorm.DB, error) {
	cfg = normalizeConfig(cfg)
	gormConfig := applyOptions(nil, opts...)
	sqlDB, err := sql.Open("pgx", buildDSN(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), gormConfig)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.PingTimeout)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func buildDSN(cfg Config) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		escapeDSNValue(cfg.Host),
		cfg.Port,
		escapeDSNValue(cfg.User),
		escapeDSNValue(cfg.Password),
		escapeDSNValue(cfg.Database),
		escapeDSNValue(cfg.SSLMode),
	)
}

func escapeDSNValue(value string) string {
	if value == "" {
		return "''"
	}

	needsQuotes := strings.IndexFunc(value, func(r rune) bool {
		return r == ' ' || r == '\'' || r == '\\'
	}) >= 0
	if !needsQuotes {
		return value
	}

	replacer := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return "'" + replacer.Replace(value) + "'"
}

func normalizeConfig(cfg Config) Config {
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 5
	}
	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = 25
	}
	if cfg.ConnMaxLifetime <= 0 {
		cfg.ConnMaxLifetime = 5 * time.Minute
	}
	if cfg.PingTimeout <= 0 {
		cfg.PingTimeout = 5 * time.Second
	}
	return cfg
}

func applyOptions(base *gorm.Config, opts ...Option) *gorm.Config {
	settings := &options{gormConfig: cloneGormConfig(base)}
	for _, opt := range opts {
		if opt != nil {
			opt(settings)
		}
	}
	if settings.gormConfig == nil {
		settings.gormConfig = &gorm.Config{}
	}
	settings.gormConfig.DisableAutomaticPing = true
	if settings.gormConfig.NowFunc == nil {
		settings.gormConfig.NowFunc = func() time.Time {
			return time.Now().UTC()
		}
	}
	return settings.gormConfig
}

func cloneGormConfig(cfg *gorm.Config) *gorm.Config {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	return &clone
}
