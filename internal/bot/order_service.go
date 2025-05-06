package bot

import (
	"astro-sarafan/internal/database"
	"astro-sarafan/internal/models"
	"astro-sarafan/internal/utils"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"math/rand"
	"strings"
	"time"

	"go.uber.org/zap"
)

// NewOrderService - создает новый сервис заказов
func NewOrderService(telegram TelegramClient, logger *zap.Logger, channelID string, orderRepo *database.OrderRepository, userRepo *database.UserRepository) *OrderService {
	return &OrderService{
		telegram:      telegram,
		logger:        logger,
		channelID:     channelID,
		orderRepo:     orderRepo,
		userRepo:      userRepo,
		orderMessages: make(map[string]string),
	}
}

func (s *OrderService) CreateOrder(clientID int64, clientName, clientUser string, referrerID int64, referrerName string) (string, error) {
	// Логируем входящие данные
	s.logger.Info("Создание заказа",
		zap.Int64("client_id", clientID),
		zap.String("client_name", clientName),
		zap.String("client_user", clientUser),
		zap.Int64("referrer_id", referrerID),
		zap.String("referrer_name", referrerName),
	)

	// Сохраняем пользователя (или обновляем информацию)
	user := models.User{
		ChatID:   clientID,
		Username: clientUser,
		FullName: clientName,
	}

	err := s.userRepo.CreateUser(user)
	if err != nil {
		s.logger.Error("ошибка при сохранении пользователя",
			zap.Error(err),
			zap.Int64("client_id", clientID),
		)
		return "", err
	}

	// Генерируем уникальный ID заказа
	orderID := generateOrderID()

	// Создаем заказ с информацией о реферере
	order := models.Order{
		ID:           orderID,
		ClientID:     clientID,
		Status:       models.OrderStatusNew,
		CreatedAt:    time.Now(),
		ReferrerID:   referrerID,
		ReferrerName: referrerName,
	}

	// Сохраняем заказ в репозитории
	err = s.orderRepo.CreateOrder(order)
	if err != nil {
		s.logger.Error("ошибка при создании заказа",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return "", err
	}

	// Обновляем клиента для заказа перед отправкой в канал
	orderWithClient := models.Order{
		ID:           orderID,
		ClientID:     clientID,
		ClientName:   clientName,
		ClientUser:   clientUser,
		Status:       models.OrderStatusNew,
		CreatedAt:    time.Now(),
		ReferrerID:   referrerID,
		ReferrerName: referrerName,
	}

	// Отправляем уведомление в канал астрологов
	messageID, err := s.telegram.SendOrderToAstrologers(s.channelID, orderWithClient)
	if err != nil {
		s.logger.Error("ошибка при отправке заказа в канал астрологов",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return orderID, err
	}

	// Сохраняем ID сообщения для возможности обновления
	s.orderMessages[orderID] = messageID

	s.logger.Info("создан новый заказ",
		zap.String("order_id", orderID),
		zap.Int64("client_id", clientID),
		zap.String("message_id", messageID),
		zap.Int64("referrer_id", referrerID),
	)

	return orderID, nil
}

func (s *OrderService) TakeOrder(orderID string, astrologerID int64, astrologerName string) error {
	// Получаем заказ из репозитория
	order, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil {
		s.logger.Error("Ошибка при получении заказа",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}

	// Корректируем данные
	clientName := order.ClientName
	if clientName == "" {
		clientName = "Unnamed User"
	}
	clientUser := order.ClientUser
	if clientUser == "" {
		clientUser = "unnamed_user"
	}
	astrologerNameSafe := utils.EscapeMarkdownV2(astrologerName)
	clientNameSafe := clientName
	clientUserSafe := utils.EscapeMarkdownV2(clientUser)

	// Логируем информацию о заказе перед обновлением
	s.logger.Info("Информация о заказе перед обновлением",
		zap.String("order_id", orderID),
		zap.String("current_status", string(order.Status)),
		zap.Int64("client_id", order.ClientID),
	)

	if order.ID == "" {
		return fmt.Errorf("заказ не найден: %s", orderID)
	}

	// Проверяем, не занят ли заказ текущим астрологом
	if order.Status == models.OrderStatusInWork && order.AstrologerID == astrologerID {
		s.logger.Info("Заказ уже взят текущим астрологом",
			zap.String("order_id", orderID),
			zap.Int64("astrologer_id", astrologerID),
		)
		// Получаем ID сообщения в канале астрологов
		messageID, exists := s.orderMessages[orderID]
		if exists {
			// Обновляем сообщение в канале астрологов
			text := fmt.Sprintf(
				"🌟 *ЗАКАЗ В РАБОТЕ* 🌟\n\n"+
					"*ID заказа:* `%s`\n"+
					"*Клиент:* %s\n"+
					"*Username:* @%s\n"+
					"*Дата заказа:* %s\n\n"+
					"*Взят в работу:* %s\n"+
					"*Астролог:* %s",
				orderID,
				clientNameSafe,
				clientUserSafe,
				order.CreatedAt.Format("02.01.2006 15:04"),
				order.TakenAt.Format("02.01.2006 15:04"),
				astrologerNameSafe,
			)

			// Пустая клавиатура для удаления кнопки
			keyboard := tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
			}

			err = s.telegram.UpdateOrderMessage(s.channelID, messageID, text, keyboard)
			if err != nil {
				s.logger.Error("Ошибка при обновлении сообщения о заказе",
					zap.Error(err),
					zap.String("order_id", orderID),
				)
			}
		}

		return nil
	}

	// Если заказ уже не в статусе new, возвращаем ошибку
	if order.Status != models.OrderStatusNew {
		s.logger.Warn("Попытка взять заказ, который уже не в статусе 'new'",
			zap.String("order_id", orderID),
			zap.String("current_status", string(order.Status)),
		)
		return fmt.Errorf("заказ уже взят в работу или завершен: %s", orderID)
	}

	// Обновляем статус заказа в репозитории
	err = s.orderRepo.UpdateOrderStatus(orderID, models.OrderStatusInWork, astrologerID, astrologerName)
	if err != nil {
		s.logger.Error("Ошибка при обновлении статуса заказа",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}

	// Получаем обновленный заказ
	updatedOrder, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil {
		s.logger.Error("Ошибка при получении обновленного заказа",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}

	// Логируем информацию об обновленном заказе
	s.logger.Info("Информация об обновленном заказе",
		zap.String("order_id", updatedOrder.ID),
		zap.String("new_status", string(updatedOrder.Status)),
		zap.Int64("client_id", updatedOrder.ClientID),
	)

	// Получаем ID сообщения в канале астрологов
	messageID, exists := s.orderMessages[orderID]
	if exists {
		// Обновляем сообщение в канале астрологов
		text := fmt.Sprintf(
			"🌟 *ЗАКАЗ В РАБОТЕ* 🌟\n\n"+
				"*ID заказа:* `%s`\n"+
				"*Клиент:* %s\n"+
				"*Username:* @%s\n"+
				"*Дата заказа:* %s\n\n"+
				"*Взят в работу:* %s\n"+
				"*Астролог:* %s",
			updatedOrder.ID,
			clientNameSafe,
			clientUserSafe,
			updatedOrder.CreatedAt.Format("02.01.2006 15:04"),
			updatedOrder.TakenAt.Format("02.01.2006 15:04"),
			astrologerNameSafe,
		)

		// Пустая клавиатура для удаления кнопки
		keyboard := tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
		}

		err = s.telegram.UpdateOrderMessage(s.channelID, messageID, text, keyboard)
		if err != nil {
			s.logger.Error("Ошибка при обновлении сообщения о заказе",
				zap.Error(err),
				zap.String("order_id", orderID),
			)
		}
	}

	// Отправляем уведомление клиенту о том, что его запрос взят в работу
	err = s.telegram.SendMessage(updatedOrder.ClientID, fmt.Sprintf(
		"✨ Ваш запрос на астрологическую консультацию принят в работу! Астролог скоро свяжется с вами.",
	))
	if err != nil {
		s.logger.Error("Ошибка при отправке уведомления клиенту",
			zap.Error(err),
			zap.Int64("client_id", updatedOrder.ClientID),
		)
	}

	s.logger.Info("Заказ взят в работу",
		zap.String("order_id", orderID),
		zap.Int64("astrologer_id", astrologerID),
	)

	return nil
}

func (s *OrderService) CheckConsultationTimeouts() {
	// Получаем список активных заказов
	activeOrders, err := s.orderRepo.GetActiveOrdersOver24Hours()
	if err != nil {
		s.logger.Error("Ошибка получения активных заказов", zap.Error(err))
		return
	}

	for _, order := range activeOrders {
		// Отправляем push-уведомление
		err := s.sendConsultationReminderMessage(order)
		if err == nil {
			// Помечаем, что напоминание отправлено, только если отправка успешна
			s.orderRepo.MarkReminderSent(order.ID)
		}
	}
}

func (s *OrderService) sendConsultationReminderMessage(order models.Order) error {
	message := "⭐ Хотите узнать больше? ⭐ Только сегодня скидка на полный анализ вашей карты!"

	// Создаем inline-кнопки
	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Записаться на консультацию", "consultation_continue"),
		),
	)

	err := s.telegram.SendMessageWithInlineKeyboard(order.ClientID, message, buttons)
	if err != nil {
		s.logger.Error("Ошибка отправки напоминания", zap.Error(err))
		return err
	}

	return nil
}

// generateOrderID - генерирует уникальный ID заказа
func generateOrderID() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const idLength = 8

	result := strings.Builder{}
	for i := 0; i < idLength; i++ {
		result.WriteByte(charset[rand.Intn(len(charset))])
	}

	return result.String()
}
