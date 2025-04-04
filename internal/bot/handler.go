package bot

import (
	"astro-sarafan/internal/database"
	"astro-sarafan/internal/models"
	"errors"
	"fmt"
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
		"🚀 Нажми сюда: [💌 Написать астрологу](https://t.me/InviteAstroBot?start=ref_%s)"

	// Ссылка для перехода к боту с командой на консультацию
	botLinkWithRefFormat = "https://t.me/" + botUsername + "?start=ref_%s"

	// Сообщение о том, что консультация уже запрошена
	consultationExistsMessage = "⚠️ Вы уже оставили заявку на астрологическую консультацию. Каждый пользователь может получить только одну бесплатную консультацию."
)

func (s *Service) HandleUpdate(update models.User) error {
	// Логируем входящие данные
	s.logger.Info("Получено обновление",
		zap.Int64("chat_id", update.ChatID),
		zap.String("text", update.Text),
		zap.String("username", update.Username),
		zap.String("full_name", update.FullName),
	)

	// Обработка команды /start
	if strings.HasPrefix(update.Text, "/start") {
		parts := strings.Split(update.Text, " ")

		// Обрабатываем реферальный код
		if len(parts) > 1 && strings.HasPrefix(parts[1], "ref_") {
			// Извлекаем реферальный код
			referralCode := strings.TrimPrefix(parts[1], "ref_")

			// Корректируем пустые значения
			username := update.Username
			if username == "" {
				username = "unnamed_user"
			}
			fullName := update.FullName
			if fullName == "" {
				fullName = "Unnamed User"
			}

			return s.handleReferralConsultationRequest(update.ChatID, fullName, username, referralCode)
		}

		// Обработка обычного запроса на консультацию
		if len(parts) > 1 && parts[1] == "astro" {
			// Корректируем пустые значения
			username := update.Username
			if username == "" {
				username = "unnamed_user"
			}
			fullName := update.FullName
			if fullName == "" {
				fullName = "Unnamed User"
			}

			return s.handleConsultationRequest(update.ChatID, fullName, username, "")
		}

		return s.sendWelcomeMessage(update.ChatID)
	}

	// Обработка нажатия на кнопку "Отправить другу"
	if update.Text == shareButtonText {
		// Корректируем пустые значения
		username := update.Username
		if username == "" {
			username = "unnamed_user"
		}
		fullName := update.FullName
		if fullName == "" {
			fullName = "Unnamed User"
		}

		return s.handleShareButton(update.ChatID, username, fullName)
	}

	// Обработка команды для получения консультации
	if update.Text == astroCommand {
		// Корректируем пустые значения
		username := update.Username
		if username == "" {
			username = "unnamed_user"
		}
		fullName := update.FullName
		if fullName == "" {
			fullName = "Unnamed User"
		}

		return s.handleConsultationRequest(update.ChatID, fullName, username, "")
	}

	// Обработка клика на текст "Написать астрологу"
	if strings.Contains(update.Text, "Написать астрологу") {
		// Корректируем пустые значения
		username := update.Username
		if username == "" {
			username = "unnamed_user"
		}
		fullName := update.FullName
		if fullName == "" {
			fullName = "Unnamed User"
		}

		return s.handleConsultationRequest(update.ChatID, fullName, username, "")
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

func (s *Service) handleShareButton(chatID int64, username string, fullName string) error {
	// Логируем входящие данные
	s.logger.Info("Обработка share button",
		zap.Int64("chat_id", chatID),
		zap.String("username", username),
		zap.String("full_name", fullName),
	)

	// Получаем пользователя из базы данных
	user, err := s.userRepo.GetUserByID(chatID)
	if err != nil {
		s.logger.Error("ошибка при получении пользователя",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
		return err
	}

	var referralCode string
	if user.ChatID == 0 { // Если пользователь не найден
		// Создаём нового пользователя
		newUser := models.User{
			ChatID:   chatID,
			Username: username,
			FullName: fullName,
		}
		if err := s.userRepo.CreateUser(newUser); err != nil {
			s.logger.Error("ошибка при создании пользователя",
				zap.Error(err),
				zap.Int64("chat_id", chatID),
			)
			return err
		}
		// Генерируем реферальный код для нового пользователя
		referralCode, err = s.userRepo.GenerateReferralCode(chatID)
		if err != nil {
			s.logger.Error("ошибка при генерации реферального кода",
				zap.Error(err),
				zap.Int64("chat_id", chatID),
			)
			return err
		}
	} else if !user.ReferralCode.Valid {
		// Если пользователь существует, но у него нет реферального кода
		referralCode, err = s.userRepo.GenerateReferralCode(chatID)
		if err != nil {
			s.logger.Error("ошибка при генерации реферального кода",
				zap.Error(err),
				zap.Int64("chat_id", chatID),
			)
			return err
		}
	} else {
		referralCode = user.ReferralCode.String
	}

	// Логируем сгенерированный реферальный код
	s.logger.Info("Реферальный код сгенерирован",
		zap.Int64("chat_id", chatID),
		zap.String("referral_code", referralCode),
	)

	// Формируем текст сообщения с персональным реферальным кодом
	personalShareMessage := fmt.Sprintf(
		"Привет! Я только что получила астрологический разбор, и он реально классный! 🙌\n\n"+
			"У меня есть для тебя уникальный подарок – мини-консультация у астролога!\n\n"+
			"🚀 Нажми сюда: [💌 Написать астрологу](https://t.me/InviteAstroBot?start=ref_%s)",
		referralCode,
	)

	// Отправляем Markdown сообщение с информацией о подарке
	if err := s.telegram.SendMarkdownMessage(chatID, personalShareMessage); err != nil {
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

func (s *Service) handleReferralConsultationRequest(clientID int64, clientName, clientUser, referralCode string) error {
	// Получаем пользователя-реферера по коду
	referrer, err := s.userRepo.GetUserByReferralCode(referralCode)
	if err != nil {
		s.logger.Error("ошибка при получении реферера",
			zap.Error(err),
			zap.String("referral_code", referralCode),
		)
		// Если не удалось получить реферера, создаем заказ без него
		return s.handleConsultationRequest(clientID, clientName, clientUser, "")
	}

	if referrer.ChatID == 0 {
		// Если реферер не найден, создаем заказ без него
		s.logger.Warn("реферер не найден по коду",
			zap.String("referral_code", referralCode),
		)
		return s.handleConsultationRequest(clientID, clientName, clientUser, "")
	}

	// Создаем заказ с информацией о реферере
	return s.handleConsultationRequest(clientID, clientName, clientUser, referralCode)
}

func (s *Service) handleConsultationRequest(clientID int64, clientName, clientUser, referralCode string) error {
	var referrerID int64
	var referrerName string

	// Если есть реферальный код, получаем информацию о реферере
	if referralCode != "" {
		referrer, err := s.userRepo.GetUserByReferralCode(referralCode)
		if err == nil && referrer.ChatID != 0 {
			referrerID = referrer.ChatID
			referrerName = referrer.FullName
		}
	}

	// Создаем заказ
	orderID, err := s.orderService.CreateOrder(clientID, clientName, clientUser, referrerID, referrerName)

	if err != nil {
		if errors.Is(err, database.ErrConsultationExists) {
			return s.telegram.SendMessage(clientID, consultationExistsMessage)
		}

		s.logger.Error("ошибка при создании заказа",
			zap.Error(err),
			zap.Int64("client_id", clientID),
		)
		return s.telegram.SendMessage(clientID, "Произошла ошибка при обработке запроса. Пожалуйста, попробуйте позже.")
	}

	// Отправляем сообщение о получении запроса
	err = s.telegram.SendMessage(clientID, "✨ Спасибо за ваш запрос на астрологическую консультацию! Наш астролог скоро получит уведомление и свяжется с вами.")
	if err != nil {
		s.logger.Error("ошибка при отправке сообщения о получении запроса",
			zap.Error(err),
			zap.Int64("chat_id", clientID),
		)
		return err
	}

	// Если есть реферер, отправляем ему уведомление
	if referrerID != 0 {
		referralNotification := fmt.Sprintf(
			"🎉 Отличные новости! Ваш друг %s воспользовался вашей рекомендацией и запросил астрологическую консультацию.",
			clientName)

		err = s.telegram.SendMessage(referrerID, referralNotification)
		if err != nil {
			s.logger.Error("ошибка при отправке уведомления рефереру",
				zap.Error(err),
				zap.Int64("referrer_id", referrerID),
			)
			// Не прерываем выполнение, если не удалось отправить уведомление
		}
	}

	s.logger.Info("создан запрос на консультацию",
		zap.Int64("client_id", clientID),
		zap.String("order_id", orderID),
		zap.Int64("referrer_id", referrerID),
	)

	return nil
}
