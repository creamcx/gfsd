package telegram

import (
	"astro-sarafan/internal/models"
	"astro-sarafan/internal/utils"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// TelegramClient реализует взаимодействие с Telegram API
type TelegramClient struct {
	bot    *tgbotapi.BotAPI
	logger *zap.Logger
}

// NewTelegramClient создает новый экземпляр клиента Telegram
func NewTelegramClient(token string) *TelegramClient {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		// Используем стандартный логгер, так как наш логгер еще не инициализирован
		// Это не критично, так как при ошибке здесь приложение все равно не запустится
		return &TelegramClient{bot: nil}
	}

	return &TelegramClient{
		bot: bot,
	}
}

// SetLogger устанавливает логгер для клиента
func (t *TelegramClient) SetLogger(logger *zap.Logger) {
	t.logger = logger
}

// SendMessage отправляет простое текстовое сообщение
func (t *TelegramClient) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := t.bot.Send(msg)
	if err != nil && t.logger != nil {
		t.logger.Error("ошибка отправки сообщения",
			zap.Int64("chat_id", chatID),
			zap.Error(err),
		)
	}
	return err
}

// SendMarkdownMessage отправляет сообщение с разметкой Markdown
func (t *TelegramClient) SendMarkdownMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = false // включаем предпросмотр URL, чтобы ссылки работали

	_, err := t.bot.Send(msg)
	if err != nil && t.logger != nil {
		t.logger.Error("ошибка отправки markdown сообщения",
			zap.Int64("chat_id", chatID),
			zap.Error(err),
		)
	}
	return err
}

// SendMessageWithKeyboard отправляет сообщение с клавиатурой
func (t *TelegramClient) SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.ReplyKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err := t.bot.Send(msg)
	if err != nil && t.logger != nil {
		t.logger.Error("ошибка отправки сообщения с клавиатурой",
			zap.Int64("chat_id", chatID),
			zap.Error(err),
		)
	}
	return err
}

// SendMessageWithInlineKeyboard отправляет сообщение с встроенной клавиатурой
func (t *TelegramClient) SendMessageWithInlineKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err := t.bot.Send(msg)
	if err != nil && t.logger != nil {
		t.logger.Error("ошибка отправки сообщения с inline клавиатурой",
			zap.Int64("chat_id", chatID),
			zap.Error(err),
		)
	}
	return err
}

// SendOrderToAstrologers отправляет заказ в канал астрологов
func (t *TelegramClient) SendOrderToAstrologers(channelID string, order models.Order) (string, error) {
	// Корректируем имя пользователя и клиента
	clientUser := order.ClientUser
	if clientUser == "" {
		clientUser = "unnamed_user"
	}
	clientName := order.ClientName
	if clientName == "" {
		clientName = "Unnamed User"
	}

	// Составляем текст сообщения для астрологов
	textBuilder := strings.Builder{}
	textBuilder.WriteString("🌟 *НОВЫЙ ЗАКАЗ НА КОНСУЛЬТАЦИЮ* 🌟\n\n")
	textBuilder.WriteString(fmt.Sprintf("*ID заказа:* `%s`\n", order.ID))
	textBuilder.WriteString(fmt.Sprintf("*Клиент:* %s\n", utils.EscapeMarkdownV2(clientName)))
	textBuilder.WriteString(fmt.Sprintf("*Username:* @%s\n", utils.EscapeMarkdownV2(clientUser)))
	textBuilder.WriteString(fmt.Sprintf("*Дата заказа:* %s\n", order.CreatedAt.Format("02.01.2006 15:04")))

	// Добавляем информацию о реферере, если есть
	if order.ReferrerID != 0 {
		textBuilder.WriteString(fmt.Sprintf("\n*Приглашен пользователем:* %s\n",
			utils.EscapeMarkdownV2(order.ReferrerName)))
	}

	textBuilder.WriteString("\n*Нажмите кнопку ниже, чтобы взять заказ в работу.*")

	// Создаем клавиатуру с кнопкой "Взять в работу"
	takeOrderButton := tgbotapi.NewInlineKeyboardButtonData("🔮 Взять в работу", "take_order:"+order.ID)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(takeOrderButton),
	)

	// Если channelID не содержит "-100" в начале, добавим
	if !strings.HasPrefix(channelID, "-100") {
		channelID = "-100" + channelID
	}

	chatID, err := strconv.ParseInt(channelID, 10, 64)
	if err != nil {
		if t.logger != nil {
			t.logger.Error("некорректный ID канала при отправке заказа",
				zap.String("channel_id", channelID),
				zap.Error(err),
			)
		}
		return "", fmt.Errorf("некорректный ID канала: %v", err)
	}

	// Отправляем сообщение
	msg := tgbotapi.NewMessage(chatID, textBuilder.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	// Отправляем сообщение и получаем его ID
	sentMsg, err := t.bot.Send(msg)
	if err != nil {
		if t.logger != nil {
			t.logger.Error("ошибка отправки заказа астрологам",
				zap.String("channel_id", channelID),
				zap.String("order_id", order.ID),
				zap.Error(err),
			)
		}
		return "", fmt.Errorf("ошибка отправки сообщения астрологам: %v", err)
	}

	return strconv.Itoa(sentMsg.MessageID), nil
}

// UpdateOrderMessage обновляет сообщение о заказе
func (t *TelegramClient) UpdateOrderMessage(channelID string, messageID string, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	// Если channelID не содержит "-100" в начале, добавим
	if !strings.HasPrefix(channelID, "-100") {
		channelID = "-100" + channelID
	}

	chatID, err := strconv.ParseInt(channelID, 10, 64)
	if err != nil {
		if t.logger != nil {
			t.logger.Error("некорректный ID канала при обновлении сообщения",
				zap.String("channel_id", channelID),
				zap.Error(err),
			)
		}
		return fmt.Errorf("некорректный ID канала: %v", err)
	}

	msgID, err := strconv.Atoi(messageID)
	if err != nil {
		if t.logger != nil {
			t.logger.Error("некорректный ID сообщения при обновлении заказа",
				zap.String("message_id", messageID),
				zap.Error(err),
			)
		}
		return fmt.Errorf("некорректный ID сообщения: %v", err)
	}

	// Создаем конфигурацию для редактирования сообщения
	editMsg := tgbotapi.NewEditMessageText(chatID, msgID, text)
	editMsg.ParseMode = "Markdown"

	// Если клавиатура пустая, удаляем ее
	if len(keyboard.InlineKeyboard) == 0 {
		editMsg.ReplyMarkup = nil
	} else {
		editMsg.ReplyMarkup = &keyboard
	}

	// Отправляем запрос на обновление
	_, err = t.bot.Send(editMsg)
	if err != nil && t.logger != nil {
		t.logger.Error("ошибка при обновлении сообщения заказа",
			zap.String("channel_id", channelID),
			zap.String("message_id", messageID),
			zap.Error(err),
		)
	}

	return err
}

// StartBot запускает получение обновлений от Telegram API
func (t *TelegramClient) StartBot() (chan models.User, chan models.CallbackQuery, error) {
	// Удаляем вебхук перед запуском Long Polling
	_, err := t.bot.Request(tgbotapi.DeleteWebhookConfig{})
	if err != nil {
		if t.logger != nil {
			t.logger.Error("ошибка удаления вебхука", zap.Error(err))
		}
		return nil, nil, fmt.Errorf("ошибка удаления вебхука: %v", err)
	}

	// Пауза для стабилизации соединения
	time.Sleep(1 * time.Second)

	// Создаем каналы для обычных сообщений и callback-запросов
	userMessages := make(chan models.User)
	callbackQueries := make(chan models.CallbackQuery)

	// Настраиваем получение обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Получаем канал обновлений
	updates := t.bot.GetUpdatesChan(u)

	// Запускаем горутину для обработки обновлений
	go func() {
		for update := range updates {
			// Обработка обычных сообщений
			if update.Message != nil {
				// Получаем информацию о пользователе
				username := update.Message.From.UserName
				fullName := update.Message.From.FirstName
				if update.Message.From.LastName != "" {
					fullName += " " + update.Message.From.LastName
				}

				// Отправляем сообщение в канал обычных сообщений
				userMessages <- models.User{
					ChatID:   update.Message.Chat.ID,
					Text:     update.Message.Text,
					Username: username,
					FullName: fullName,
				}

				if t.logger != nil {
					t.logger.Debug("получено сообщение",
						zap.Int64("chat_id", update.Message.Chat.ID),
						zap.String("text", update.Message.Text),
						zap.String("username", username),
					)
				}
			}

			// Обработка callback-запросов (нажатий на инлайн-кнопки)
			if update.CallbackQuery != nil {
				userName := update.CallbackQuery.From.FirstName
				if update.CallbackQuery.From.LastName != "" {
					userName += " " + update.CallbackQuery.From.LastName
				}

				// Отправляем callback-запрос в соответствующий канал
				callbackQueries <- models.CallbackQuery{
					ID:          update.CallbackQuery.ID,
					UserID:      update.CallbackQuery.From.ID,
					UserName:    userName,
					UserLogin:   update.CallbackQuery.From.UserName,
					MessageID:   strconv.Itoa(update.CallbackQuery.Message.MessageID),
					ChatID:      strconv.FormatInt(update.CallbackQuery.Message.Chat.ID, 10),
					Data:        update.CallbackQuery.Data,
					MessageText: update.CallbackQuery.Message.Text,
				}

				if t.logger != nil {
					t.logger.Debug("получен callback",
						zap.String("callback_id", update.CallbackQuery.ID),
						zap.Int64("user_id", update.CallbackQuery.From.ID),
						zap.String("data", update.CallbackQuery.Data),
					)
				}

				// Отправляем ответ на callback, чтобы убрать индикатор загрузки у кнопки
				callbackCfg := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
				t.bot.Send(callbackCfg)
			}
		}
	}()

	return userMessages, callbackQueries, nil
}
