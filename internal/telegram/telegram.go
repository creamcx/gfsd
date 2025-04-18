package telegram

import (
	"astro-sarafan/internal/models"
	"astro-sarafan/internal/utils"
	"bytes"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type TelegramClient struct {
	bot    *tgbotapi.BotAPI
	client *http.Client
}

func NewTelegramClient(token string) *TelegramClient {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Printf("error creating telegram client: %v", err)
	}

	return &TelegramClient{
		bot:    bot,
		client: &http.Client{},
	}
}

func (t *TelegramClient) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := t.bot.Send(msg)
	return err
}

func (t *TelegramClient) SendMarkdownMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = false // включаем предпросмотр URL, чтобы ссылки работали

	_, err := t.bot.Send(msg)
	return err
}

func (t *TelegramClient) SendMarkdownMessageAndGetID(chatID int64, text string) (string, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = false // включаем предпросмотр URL, чтобы ссылки работали

	sentMsg, err := t.bot.Send(msg)
	if err != nil {
		return "", err
	}

	return strconv.Itoa(sentMsg.MessageID), nil
}

func (t *TelegramClient) SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.ReplyKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err := t.bot.Send(msg)
	return err
}

func (t *TelegramClient) SendMessageWithInlineKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err := t.bot.Send(msg)
	return err
}

func (t *TelegramClient) UpdateMessageReplyMarkup(chatID int64, messageID string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msgID, err := strconv.Atoi(messageID)
	if err != nil {
		return fmt.Errorf("invalid message ID: %v", err)
	}

	editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, msgID, keyboard)
	_, err = t.bot.Send(editMsg)
	return err
}

func (t *TelegramClient) SendCustomMessage(params map[string]interface{}) error {
	// Формируем URL для Telegram API
	url := "https://api.telegram.org/bot" + t.bot.Token + "/sendMessage"

	// Преобразуем параметры в JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		return err
	}

	// Создаем HTTP запрос
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// Устанавливаем заголовок Content-Type
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}

// Отправка сообщения в канал
func (t *TelegramClient) SendMessageToChannel(channelID string, text string) error {
	// Если channelID не содержит "-100" в начале, добавим
	if !strings.HasPrefix(channelID, "-100") {
		channelID = "-100" + channelID
	}

	chatID, err := strconv.ParseInt(channelID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid channel ID: %v", err)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	_, err = t.bot.Send(msg)
	return err
}

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
	textBuilder.WriteString(fmt.Sprintf("*Клиент:* %s\n", clientName))
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
		return "", fmt.Errorf("invalid channel ID: %v", err)
	}

	// Отправляем сообщение
	msg := tgbotapi.NewMessage(chatID, textBuilder.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	// Отправляем сообщение и получаем его ID
	sentMsg, err := t.bot.Send(msg)
	if err != nil {
		return "", fmt.Errorf("error sending message to astrologers: %v", err)
	}

	// Возвращаем ID отправленного сообщения для дальнейшего обновления
	return strconv.Itoa(sentMsg.MessageID), nil
}

// Обновление сообщения о заказе (например, когда заказ взят в работу)
func (t *TelegramClient) UpdateOrderMessage(channelID string, messageID string, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	// Если channelID не содержит "-100" в начале, добавим
	if !strings.HasPrefix(channelID, "-100") {
		channelID = "-100" + channelID
	}

	chatID, err := strconv.ParseInt(channelID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid channel ID: %v", err)
	}

	msgID, err := strconv.Atoi(messageID)
	if err != nil {
		return fmt.Errorf("invalid message ID: %v", err)
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

	// Логируем попытку обновления сообщения
	log.Printf("Обновление сообщения: chat_id=%d, message_id=%d, text=%s",
		chatID, msgID, text,
	)

	// Отправляем запрос на обновление
	_, err = t.bot.Send(editMsg)
	if err != nil {
		log.Printf("Ошибка при обновлении сообщения: %v", err)
		return err
	}

	return nil
}

// Единый метод обработки обновлений
func (t *TelegramClient) StartBot() (chan models.User, chan models.CallbackQuery, error) {
	// Удаляем вебхук перед запуском Long Polling
	_, err := t.bot.Request(tgbotapi.DeleteWebhookConfig{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete webhook: %v", err)
	}

	// Пауза для стабилизации соединения
	time.Sleep(1 * time.Second)

	// Создаем каналы для обычных сообщений и callback-запросов
	userMessages := make(chan models.User)
	callbackQueries := make(chan models.CallbackQuery)

	// Настраиваем получение обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Важно! Теперь используем только один вызов GetUpdatesChan
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

				// Отправляем ответ на callback, чтобы убрать индикатор загрузки у кнопки
				callbackCfg := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
				t.bot.Send(callbackCfg)
			}
		}
	}()

	return userMessages, callbackQueries, nil
}

// Эти методы теперь устарели, но мы оставляем их для обратной совместимости
// Они должны быть удалены в дальнейшем

func (t *TelegramClient) ListenUpdates() (<-chan models.User, error) {
	log.Println("ВНИМАНИЕ: Метод ListenUpdates устарел. Используйте StartBot вместо него")
	return nil, fmt.Errorf("метод устарел, используйте StartBot")
}

func (t *TelegramClient) ListenCallbackQueries() (<-chan models.CallbackQuery, error) {
	log.Println("ВНИМАНИЕ: Метод ListenCallbackQueries устарел. Используйте StartBot вместо него")
	return nil, fmt.Errorf("метод устарел, используйте StartBot")
}
