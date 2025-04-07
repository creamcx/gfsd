package app

import (
	"astro-sarafan/internal/api"
	"astro-sarafan/internal/bot"
	"astro-sarafan/internal/config"
	"astro-sarafan/internal/database"
	"astro-sarafan/internal/logger"
	"astro-sarafan/internal/telegram"
	"os"
	_ "time"

	"go.uber.org/zap"
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

	// Создаем директорию для хранения PDF, если её нет
	if err := os.MkdirAll(cfg.API.PDFStoragePath, 0755); err != nil {
		logger.Error("ошибка создания директории для PDF", zap.Error(err))
		return err
	}

	// Инициализируем HTTP-сервер для обработки нажатий на кнопку
	buttonServer := api.NewButtonServer(
		logger,
		orderRepo,
		cfg.API.ButtonServerAddr,
		cfg.API.PDFStoragePath,
	)
	buttonServer.Start()

	// Инициализируем сервис напоминаний
	reminderService := bot.NewReminderService(orderRepo, tgClient, logger)
	reminderService.Start()

	// Инициализируем основной сервис бота
	botService := bot.NewService(tgClient, logger, orderService, userRepo)

	// Запускаем бота
	if err := botService.Start(); err != nil {
		logger.Error("ошибка запуска бота", zap.Error(err))
		return err
	}

	return nil
}
