package bitrixgo

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

const (
	envDSN          = "BITRIX_DSN"
	envDBHost       = "BITRIX_DB_HOST"
	envDBPort       = "BITRIX_DB_PORT"
	envDBUser       = "BITRIX_DB_USER"
	envDBPassword   = "BITRIX_DB_PASSWORD"
	envDBName       = "BITRIX_DB_NAME"
	envDBParams     = "BITRIX_DB_PARAMS"
	envTablePrefix  = "BITRIX_TABLE_PREFIX"
	defaultDBPort   = "3306"
	defaultDBParams = "parseTime=true&charset=utf8mb4"
)

// LoadEnv загружает переменные из .env-файлов в окружение процесса.
// Отсутствующие файлы игнорируются; ошибки разбора возвращаются вызывающему.
func LoadEnv(paths ...string) error {
	if len(paths) == 0 {
		paths = []string{".env"}
	}
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		if err := godotenv.Load(path); err != nil {
			return fmt.Errorf("bitrixgo: load env %s: %w", path, err)
		}
	}
	return nil
}

// DSNFromEnv формирует MySQL DSN из переменных окружения.
//
// Если задан BITRIX_DSN, он возвращается без изменений.
// Иначе DSN собирается из BITRIX_DB_HOST, BITRIX_DB_PORT, BITRIX_DB_USER,
// BITRIX_DB_PASSWORD, BITRIX_DB_NAME и опционального BITRIX_DB_PARAMS.
func DSNFromEnv() (string, error) {
	if dsn := os.Getenv(envDSN); dsn != "" {
		return dsn, nil
	}

	host := os.Getenv(envDBHost)
	user := os.Getenv(envDBUser)
	dbName := os.Getenv(envDBName)
	if host == "" || user == "" || dbName == "" {
		return "", fmt.Errorf("bitrixgo: set %s or %s, %s, %s", envDSN, envDBHost, envDBUser, envDBName)
	}

	port := os.Getenv(envDBPort)
	if port == "" {
		port = defaultDBPort
	}
	if _, err := strconv.Atoi(port); err != nil {
		return "", fmt.Errorf("bitrixgo: invalid %s: %q", envDBPort, port)
	}

	params := os.Getenv(envDBParams)
	if params == "" {
		params = defaultDBParams
	}

	cfg := mysql.Config{
		User:                 user,
		Passwd:               os.Getenv(envDBPassword),
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%s", host, port),
		DBName:               dbName,
		AllowNativePasswords: true,
	}

	parseTime := false
	for _, part := range splitParams(params) {
		key, val, ok := splitParam(part)
		if !ok {
			continue
		}
		switch key {
		case "parseTime":
			parseTime = val == "true"
		case "charset":
			if cfg.Params == nil {
				cfg.Params = make(map[string]string)
			}
			cfg.Params["charset"] = val
		default:
			if cfg.Params == nil {
				cfg.Params = make(map[string]string)
			}
			cfg.Params[key] = val
		}
	}
	cfg.ParseTime = parseTime

	return cfg.FormatDSN(), nil
}

// TablePrefixFromEnv возвращает BITRIX_TABLE_PREFIX или пустую строку.
func TablePrefixFromEnv() string {
	return os.Getenv(envTablePrefix)
}

// NewFromEnv загружает .env (если есть), читает настройки подключения из окружения
// и открывает Client. Опции, переданные в NewFromEnv, переопределяют значения из env.
func NewFromEnv(ctx context.Context, envFiles ...string) (*Client, error) {
	if err := LoadEnv(envFiles...); err != nil {
		return nil, err
	}

	dsn, err := DSNFromEnv()
	if err != nil {
		return nil, err
	}

	opts := envOptions()
	return New(ctx, dsn, opts...)
}

func envOptions() []Option {
	var opts []Option
	if prefix := TablePrefixFromEnv(); prefix != "" {
		opts = append(opts, WithTablePrefix(prefix))
	}
	return opts
}

func splitParams(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '&' {
			if i > start {
				parts = append(parts, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func splitParam(part string) (key, val string, ok bool) {
	for i := 0; i < len(part); i++ {
		if part[i] == '=' {
			return part[:i], part[i+1:], true
		}
	}
	return "", "", false
}
