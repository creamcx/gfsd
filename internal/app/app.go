package app

import (
	"astro-sarafan/internal/bot"
	"astro-sarafan/internal/config"
	"astro-sarafan/internal/database"
	"astro-sarafan/internal/logger"
	"astro-sarafan/internal/telegram"
	"go.uber.org/zap"
	"time"
)

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
