package bot

import (
	"astro-sarafan/internal/database"
	"astro-sarafan/internal/models"
	"errors"
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

	// Текст сообщения для пересылки с Markdown
	shareMessageText = "Привет! Я только что получила " +
		"астрологический разбор, и он реально " +
		"классный! 🙌 У меня есть для " +
		"тебя уникальный подарок – " +
		"мини-консультация у астролога!\n\n" +
		"🚀 Нажми сюда: [💌 Написать астрологу](https://t.me/InviteAstroBot?start=astro)"

	// Ссылка для перехода к боту с командой на консультацию
	botLink = "https://t.me/" + botUsername + "?start=astro"

	// Сообщение о том, что консультация уже запрошена
	consultationExistsMessage = "⚠️ Вы уже оставили заявку на астрологическую консультацию. Каждый пользователь может получить только одну бесплатную консультацию."
)

// HandleUpdate - основной обработчик входящих сообщений
func (s *Service) HandleUpdate(update models.User) error {
	// Обработка команды /start
	if strings.HasPrefix(update.Text, "/start") {
		parts := strings.Split(update.Text, " ")
		if len(parts) > 1 && parts[1] == "astro" {
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

	// Обработка клика на текст "Написать астрологу" (если кто-то вручную напишет)
	if strings.Contains(update.Text, "Написать астрологу") {
		return s.handleConsultationRequest(update.ChatID, update.FullName, update.Username)
	}

	// В остальных случаях отправляем стандартный ответ
	return s.telegram.SendMessage(update.ChatID, "Здравствуйте! Отправьте /start для начала работы с ботом.")
}

// sendWelcomeMessage - отправляет приветственное сообщение с кнопкой "Отправить другу"
func (s *Service) sendWelcomeMessage(chatID int64) error {
	// Отправляем приветственное сообщение
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
	msg := "Нажмите кнопку ниже, чтобы подготовить приглашение для друга 👇"
	if err := s.telegram.SendMessageWithKeyboard(chatID, msg, keyboard); err != nil {
		s.logger.Error("ошибка при отправке сообщения с кнопкой",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
		return err
	}

	return nil
}

func (s *Service) handleShareButton(chatID int64) error {
	if err := s.telegram.SendMarkdownMessage(chatID, shareMessageText); err != nil {
		s.logger.Error("ошибка при отправке сообщения с Markdown",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
		return err
	}

	// Отправляем инструкцию для пересылки
	instruction := "👆 Перешлите это сообщение другу, чтобы он получил подарок!"
	if err := s.telegram.SendMessage(chatID, instruction); err != nil {
		s.logger.Error("ошибка при отправке инструкции",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
		return err
	}

	return nil
}

// handleConsultationRequest - обработка запроса на консультацию
func (s *Service) handleConsultationRequest(chatID int64, clientName, clientUser string) error {
	// Создаем заказ
	orderID, err := s.orderService.CreateOrder(chatID, clientName, clientUser)

	if err != nil {
		if errors.Is(err, database.ErrConsultationExists) {
			return s.telegram.SendMessage(chatID, consultationExistsMessage)
		}

		s.logger.Error("ошибка при создании заказа",
			zap.Error(err),
			zap.Int64("client_id", chatID),
		)
		return s.telegram.SendMessage(chatID, "Произошла ошибка при обработке запроса. Пожалуйста, попробуйте позже.")
	}

	// Отправляем сообщение о получении запроса
	err = s.telegram.SendMessage(chatID, "✨ Спасибо за ваш запрос на астрологическую консультацию! Наш астролог скоро получит уведомление и свяжется с вами.")
	if err != nil {
		s.logger.Error("ошибка при отправке сообщения о получении запроса",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
		return err
	}

	s.logger.Info("создан запрос на консультацию",
		zap.Int64("client_id", chatID),
		zap.String("order_id", orderID),
	)

	return nil
}
