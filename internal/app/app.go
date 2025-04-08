package app

import (
	"astro-sarafan/internal/bot"
	"astro-sarafan/internal/config"
	"astro-sarafan/internal/database"
	"astro-sarafan/internal/logger"
	"astro-sarafan/internal/telegram"
	"fmt"
	"time"

	"go.uber.org/zap"
)

func Run(configPath string, runMigrations, rollbackMigrations, verbose bool) error {
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		return err
	}

	// Инициализируем логгер с учетом verbose-режима
	logConfig := cfg.Logger
	if verbose && logConfig.Level != "debug" {
		logConfig.Level = "debug"
		fmt.Println("Включен режим подробного логирования, уровень логирования установлен в 'debug'")
	}

	l, err := logger.New(logConfig)
	if err != nil {
		zap.L().Error("не удалось создать логгер", zap.Error(err))
		return err
	}

	// Подключаемся к базе данных
	db, err := database.NewConnection(cfg.Database, l)
	if err != nil {
		l.Error("не удалось подключиться к базе данных", zap.Error(err))
		return err
	}

	// Если указан флаг миграции, выполняем миграции и завершаем работу
	if runMigrations {
		l.Info("Запуск миграций базы данных",
			zap.String("config_path", configPath),
			zap.Bool("verbose", verbose),
		)
		if err := database.MigrateUp(cfg.Database, l, verbose); err != nil {
			l.Error("ошибка при выполнении миграций", zap.Error(err))
			return err
		}
		l.Info("Миграции успешно выполнены")
		return nil
	}

	// Если указан флаг отката миграций, выполняем откат и завершаем работу
	if rollbackMigrations {
		l.Info("Запуск отката миграций базы данных",
			zap.String("config_path", configPath),
			zap.Bool("verbose", verbose),
		)
		if err := database.MigrateDown(cfg.Database, l, verbose); err != nil {
			l.Error("ошибка при откате миграций", zap.Error(err))
			return err
		}
		l.Info("Откат миграций успешно выполнен")
		return nil
	}

	// Инициализируем репозитории
	userRepo := database.NewUserRepository(db, l)
	orderRepo := database.NewOrderRepository(db, l)

	// Инициализируем Telegram клиент
	tgClient := telegram.NewTelegramClient(cfg.Telegram.Token)
	tgClient.SetLogger(l)

	// Инициализируем сервис заказов
	orderService := bot.NewOrderService(tgClient, l, cfg.Telegram.AstrologerChannel, orderRepo, userRepo)

	// Инициализируем основной сервис бота
	botService := bot.NewService(tgClient, l, orderService, userRepo)

	// Запускаем проверку таймаутов консультаций в отдельной горутине
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			orderService.CheckConsultationTimeouts()
		}
	}()

	// Запускаем бота
	if err := botService.Start(); err != nil {
		l.Error("ошибка при запуске бота", zap.Error(err))
		return err
	}

	return nil
}
