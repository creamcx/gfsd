package bot

import (
	"astro-sarafan/internal/database"
	"astro-sarafan/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// TelegramClient - интерфейс для взаимодействия с Telegram API
type TelegramClient interface {
	// Базовые методы отправки сообщений
	SendMessage(chatID int64, text string) error
	SendMarkdownMessage(chatID int64, text string) error
	SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.ReplyKeyboardMarkup) error
	SendMessageWithInlineKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error

	// Методы для работы с заказами
	SendOrderToAstrologers(channelID string, order models.Order) (string, error)
	UpdateOrderMessage(channelID string, messageID string, text string, keyboard tgbotapi.InlineKeyboardMarkup) error

	// Метод для получения обновлений
	StartBot() (chan models.User, chan models.CallbackQuery, error)
}

// Service - основной сервис бота
type Service struct {
	telegram     TelegramClient
	logger       *zap.Logger
	orderService *OrderService
	userRepo     *database.UserRepository
}

// OrderService - структура сервиса заказов
type OrderService struct {
	orderMessages map[string]string
	telegram      TelegramClient
	logger        *zap.Logger
	channelID     string
	orderRepo     *database.OrderRepository
	userRepo      *database.UserRepository
}
