package bitrixgo

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

const defaultBatchSize = 500

// Client предоставляет доступ к MySQL-базе Bitrix.
type Client struct {
	db          *sql.DB
	tablePrefix string
	batchSize   int
}

// New открывает MySQL-соединение и проверяет его через PingContext.
func New(ctx context.Context, dsn string, opts ...Option) (*Client, error) {
	cfg := Config{
		DSN:       dsn,
		BatchSize: defaultBatchSize,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaultBatchSize
	}

	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("bitrixgo: open db: %w", err)
	}
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("bitrixgo: ping db: %w", err)
	}

	return &Client{
		db:          db,
		tablePrefix: cfg.TablePrefix,
		batchSize:   cfg.BatchSize,
	}, nil
}

// DB возвращает базовый *sql.DB.
func (c *Client) DB() *sql.DB {
	return c.db
}

// TablePrefix возвращает настроенный префикс имён таблиц.
func (c *Client) TablePrefix() string {
	return c.tablePrefix
}

// BatchSize возвращает настроенный размер пакетной вставки.
func (c *Client) BatchSize() int {
	return c.batchSize
}

// FullTableName склеивает префикс и логическое имя таблицы.
func (c *Client) FullTableName(logical string) string {
	return c.tablePrefix + logical
}

// Close закрывает соединение с БД.
func (c *Client) Close() error {
	return c.db.Close()
}
