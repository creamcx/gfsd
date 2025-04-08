package database

import (
	"astro-sarafan/internal/config"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // драйвер для PostgreSQL
	"go.uber.org/zap"
)

// NewConnection создает новое подключение к базе данных
func NewConnection(cfg config.Database, logger *zap.Logger) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		logger.Error("Ошибка подключения к базе данных", zap.Error(err))
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
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
