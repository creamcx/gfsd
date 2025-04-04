package bot

import (
	"astro-sarafan/internal/database"
	"astro-sarafan/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// TelegramClient - интерфейс для взаимодействия с Telegram API
type TelegramClient interface {
	SendMessage(chatID int64, text string) error
	SendMarkdownMessage(chatID int64, text string) error
	SendMarkdownMessageAndGetID(chatID int64, text string) (string, error)
	SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.ReplyKeyboardMarkup) error
	SendMessageWithInlineKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error
	SendMessageToChannel(channelID string, text string) error
	SendOrderToAstrologers(channelID string, order models.Order) (string, error)
	UpdateMessageReplyMarkup(chatID int64, messageID string, keyboard tgbotapi.InlineKeyboardMarkup) error
	UpdateOrderMessage(channelID string, messageID string, text string, keyboard tgbotapi.InlineKeyboardMarkup) error
	SendCustomMessage(params map[string]interface{}) error

	// Метод объединенного обработчика обновлений
	StartBot() (chan models.User, chan models.CallbackQuery, error)

	// Устаревшие методы, оставлены для обратной совместимости
	ListenUpdates() (<-chan models.User, error)
	ListenCallbackQueries() (<-chan models.CallbackQuery, error)
}

// Service - основной сервис бота
type Service struct {
	telegram     TelegramClient
	logger       *zap.Logger
	orderService *OrderService
}

// OrderService - структура сервиса заказов
type OrderService struct {
	orders        map[string]models.Order
	orderMessages map[string]string
	telegram      TelegramClient
	logger        *zap.Logger
	channelID     string
	orderRepo     *database.OrderRepository
	userRepo      *database.UserRepository
}
