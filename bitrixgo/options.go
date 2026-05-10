package bitrixgo

// Option настраивает Client.
type Option func(*Config)

// Config хранит параметры подключения к БД.
type Config struct {
	DSN          string
	TablePrefix  string
	MaxOpenConns int
	MaxIdleConns int
	BatchSize    int
}

// WithTablePrefix задаёт опциональный префикс имён таблиц (например, "b_").
func WithTablePrefix(prefix string) Option {
	return func(c *Config) {
		c.TablePrefix = prefix
	}
}

// WithMaxOpenConns задаёт максимальное число открытых соединений.
func WithMaxOpenConns(n int) Option {
	return func(c *Config) {
		c.MaxOpenConns = n
	}
}

// WithMaxIdleConns задаёт максимальное число простаивающих соединений.
func WithMaxIdleConns(n int) Option {
	return func(c *Config) {
		c.MaxIdleConns = n
	}
}

// WithBatchSize задаёт максимальное число строк в одном batch INSERT.
func WithBatchSize(n int) Option {
	return func(c *Config) {
		c.BatchSize = n
	}
}
