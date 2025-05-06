package database

import (
	"astro-sarafan/internal/config"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // драйвер для PostgreSQL
	"go.uber.org/zap"
)

const (
	maxRetries    = 10              // Максимальное количество попыток
	retryInterval = 3 * time.Second // Интервал между попытками
)

// NewConnection создает новое подключение к базе данных с retry-логикой
func NewConnection(cfg config.Database, logger *zap.Logger) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	var db *sqlx.DB
	var err error

	// Retry-логика подключения
	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err = sqlx.Connect("postgres", dsn)
		if err == nil {
			break
		}

		logger.Warn(
			"Не удалось подключиться к базе данных, повторная попытка",
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", maxRetries),
			zap.Error(err),
		)

		if attempt == maxRetries {
			logger.Error("Достигнуто максимальное количество попыток подключения", zap.Error(err))
			return nil, fmt.Errorf("не удалось подключиться к базе данных после %d попыток: %w", maxRetries, err)
		}

		time.Sleep(retryInterval)
	}

	// Установка настроек пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// Проверка подключения
	if err := db.Ping(); err != nil {
		logger.Error("Ошибка проверки подключения к базе данных", zap.Error(err))
		return nil, fmt.Errorf("не удалось проверить подключение к базе данных: %w", err)
	}

	logger.Info("Успешное подключение к базе данных")
	return db, nil
}
