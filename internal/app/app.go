package app

import (
	"astro-sarafan/internal/bot"
	"astro-sarafan/internal/config"
	"astro-sarafan/internal/database"
	"astro-sarafan/internal/logger"
	"astro-sarafan/internal/telegram"
	"database/sql"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
	"time"
)

func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("не удалось создать драйвер миграций: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("не удалось создать экземпляр миграций: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("ошибка выполнения миграций: %w", err)
	}

	return nil
}

func Run() error {
	// Загружаем конфигурацию
	cfg, err := config.NewConfig("config.yaml")
	if err != nil {
		return err
	}

	// Инициализируем логгер
	logger, err := logger.New(cfg.Logger)
	if err != nil {
		zap.L().Error("не удалось создать логгер", zap.Error(err))
		return err
	}

	// Подключаемся к базе данных
	db, err := database.NewConnection(cfg.Database, logger)
	if err != nil {
		logger.Error("не удалось подключиться к базе данных", zap.Error(err))
		return err
	}
	go func() {
		if err := runMigrations(db.DB); err != nil {
			logger.Error("не удалось выполнить миграции", zap.Error(err))
		}
	}()

	// Инициализируем репозитории
	userRepo := database.NewUserRepository(db, logger)
	orderRepo := database.NewOrderRepository(db, logger)

	// Инициализируем Telegram клиент
	tgClient := telegram.NewTelegramClient(cfg.Telegram.Token)

	// Инициализируем сервис заказов
	orderService := bot.NewOrderService(tgClient, logger, cfg.Telegram.AstrologerChannel, orderRepo, userRepo)

	// Инициализируем основной сервис бота
	botService := bot.NewService(tgClient, logger, orderService, userRepo)

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			orderService.CheckConsultationTimeouts()
		}
	}()

	// Запускаем бота
	if err := botService.Start(); err != nil {
		logger.Error("failed to start bot", zap.Error(err))
		return err
	}

	return nil
}
