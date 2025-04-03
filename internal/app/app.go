package app

import (
	"astro-sarafan/internal/bot"
	"astro-sarafan/internal/config"
	"astro-sarafan/internal/logger"
	"astro-sarafan/internal/telegram"
	"go.uber.org/zap"
)

func Run() error {
	cfg, err := config.NewConfig("config.yaml")
	if err != nil {
		return err
	}

	logger, err := logger.New(cfg.Logger)
	if err != nil {
		zap.L().Error("не удалось создать логгер", zap.Error(err))
		return err
	}
	tgClient := telegram.NewTelegramClient(cfg.Telegram.Token)
	botService := bot.NewService(tgClient, logger, cfg.Telegram.AstrologerChannel)

	if err := botService.Start(); err != nil {
		logger.Error("failed to start bot: %v", zap.Error(err))
		return err
	}

	return nil
}
