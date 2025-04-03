package bot

import (
	"astro-sarafan/internal/models"
	"net/url"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

const (
	// Текст, который показывается при первом запуске бота
	welcomeMessage = `🎉 Ваш подарок готов!
👉 Просто перешлите это сообщение своему другу и
он получит бесплатную 
мини-консультацию у астролога.`

	// Текст кнопки
	shareButtonText = "Отправить другу"

	// Команда для астрологической консультации
	astroCommand = "/astro_consultation"

	// Имя бота (нужно заменить на имя вашего бота)
	botUsername = "InviteAstroBot"

	// Текст сообщения для друга
	shareMessageText = `«Привет! Я только что получила 
астрологический разбор, и он реально 
классный! 🙌 У меня есть для 
тебя уникальный подарок – 
мини-консультация у астролога!

👉 Просто нажми сюда и 
бот автоматически отправит твой 
промокод астрологу → 💌 Написать астрологу`

	// Ссылка для перехода к боту с командой на консультацию
	botLink = "https://t.me/" + botUsername + "?start=astro"
)

// HandleUpdate - основной обработчик входящих сообщений
func (s *Service) HandleUpdate(update models.User) error {
	// Обработка команды /start
	if strings.HasPrefix(update.Text, "/start") {
		// Проверяем, есть ли параметры в команде start
		parts := strings.Split(update.Text, " ")
		if len(parts) > 1 && parts[1] == "astro" {
			// Это запрос на консультацию по глубокой ссылке
			return s.handleConsultationRequest(update.ChatID, update.FullName, update.Username)
		}
		return s.sendWelcomeMessage(update.ChatID)
	}

	// Обработка нажатия на кнопку "Отправить другу"
	if update.Text == shareButtonText {
		return s.handleShareButton(update.ChatID)
	}

	// Обработка команды для получения консультации
	if update.Text == astroCommand {
		return s.handleConsultationRequest(update.ChatID, update.FullName, update.Username)
	}

	// Обработка клика на текст "Написать астрологу"
	if strings.Contains(update.Text, "Написать астрологу") {
		return s.handleConsultationRequest(update.ChatID, update.FullName, update.Username)
	}

	// В остальных случаях отправляем стандартный ответ
	return s.telegram.SendMessage(update.ChatID, "Здравствуйте! Отправьте /start для начала работы с ботом.")
}

// sendWelcomeMessage - отправляет приветственное сообщение с кнопкой "Отправить другу"
func (s *Service) sendWelcomeMessage(chatID int64) error {
	// Отправляем только приветственное сообщение с текстом для форварда
	if err := s.telegram.SendMessage(chatID, welcomeMessage); err != nil {
		s.logger.Error("ошибка при отправке приветственного сообщения",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
		return err
	}

	// Создаем кнопку "Отправить другу"
	shareButton := tgbotapi.NewKeyboardButton(shareButtonText)
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(shareButton),
	)
	keyboard.OneTimeKeyboard = false
	keyboard.ResizeKeyboard = true

	// Отправляем сообщение с кнопкой
	msg := "Нажмите кнопку ниже, чтобы отправить приглашение другу 👇"
	if err := s.telegram.SendMessageWithKeyboard(chatID, msg, keyboard); err != nil {
		s.logger.Error("ошибка при отправке сообщения с кнопкой",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
		return err
	}

	return nil
}

// handleShareButton - обрабатывает нажатие на кнопку "Отправить другу"
func (s *Service) handleShareButton(chatID int64) error {
	// Создаем закодированную ссылку для шаринга сообщения в Telegram
	// Это специальная ссылка, которая позволяет пользователю выбрать, кому отправить
	// предварительно заполненное сообщение
	messageToShare := shareMessageText + "\n\n" + botLink

	// URL-кодирование сообщения для ссылки
	encodedMessage := url.QueryEscape(messageToShare)

	// Создаем ссылку для шаринга в Telegram
	shareUrl := "https://t.me/share/url?url=" + url.QueryEscape(botLink) + "&text=" + encodedMessage

	// Создаем кнопку с прямой ссылкой для шаринга
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.InlineKeyboardButton{
				Text: "👥 Отправить другу",
				URL:  &shareUrl,
			},
		),
	)

	// Отправляем предварительное сообщение
	msg := "Нажмите на кнопку ниже, чтобы отправить приглашение другу. Откроется окно с выбором получателя:"
	if err := s.telegram.SendMessageWithInlineKeyboard(chatID, msg, keyboard); err != nil {
		s.logger.Error("ошибка при отправке сообщения с кнопкой",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
		return err
	}

	return nil
}

// handleConsultationRequest - обработка запроса на консультацию
func (s *Service) handleConsultationRequest(chatID int64, clientName, clientUser string) error {
	// Отправляем сообщение о получении запроса
	err := s.telegram.SendMessage(chatID, "✨ Спасибо за ваш запрос на астрологическую консультацию! Наш астролог скоро получит уведомление и свяжется с вами.")
	if err != nil {
		s.logger.Error("ошибка при отправке сообщения о получении запроса",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
		return err
	}

	// Если имя не указано, используем значение по умолчанию
	if clientName == "" {
		clientName = "Пользователь"
	}

	// Создаем заказ и отправляем его в канал астрологов
	orderID, err := s.orderService.CreateOrder(chatID, clientName, clientUser)
	if err != nil {
		s.logger.Error("ошибка при создании заказа",
			zap.Error(err),
			zap.Int64("client_id", chatID),
		)
		return err
	}

	s.logger.Info("создан запрос на консультацию",
		zap.Int64("client_id", chatID),
		zap.String("order_id", orderID),
	)

	return nil
}
