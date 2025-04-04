package bot

import (
	"astro-sarafan/internal/database"
	"astro-sarafan/internal/models"
	"fmt"
	"math/rand"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

// CreateOrder - создает новый заказ на консультацию
func (s *OrderService) CreateOrder(clientID int64, clientName, clientUser string, referrerID int64, referrerName string) (string, error) {
	// Проверяем, есть ли у пользователя уже активная консультация
	hasConsultation, err := s.userRepo.HasActiveConsultation(clientID)
	if err != nil {
		s.logger.Error("ошибка при проверке наличия консультации",
			zap.Error(err),
			zap.Int64("client_id", clientID),
		)
		return "", err
	}

	if hasConsultation {
		s.logger.Info("у пользователя уже есть активная консультация",
			zap.Int64("client_id", clientID),
		)
		return "", database.ErrConsultationExists
	}

	// Сохраняем пользователя (или обновляем информацию)
	user := models.User{
		ChatID:   clientID,
		Username: clientUser,
		FullName: clientName,
	}

	err = s.userRepo.CreateUser(user)
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

// TakeOrder - взятие заказа в работу астрологом
func (s *OrderService) TakeOrder(orderID string, astrologerID int64, astrologerName string) error {
	// Получаем заказ из репозитория
	order, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil {
		s.logger.Error("ошибка при получении заказа",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}

	if order.ID == "" {
		return fmt.Errorf("заказ не найден: %s", orderID)
	}

	// Проверяем, не взят ли заказ уже в работу
	if order.Status != models.OrderStatusNew {
		return fmt.Errorf("заказ уже взят в работу или завершен: %s", orderID)
	}

	// Обновляем статус заказа в репозитории
	err = s.orderRepo.UpdateOrderStatus(orderID, models.OrderStatusInWork, astrologerID, astrologerName)
	if err != nil {
		s.logger.Error("ошибка при обновлении статуса заказа",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}

	// Получаем ID сообщения в канале астрологов
	messageID, exists := s.orderMessages[orderID]
	if !exists {
		s.logger.Error("не найдено сообщение для заказа",
			zap.String("order_id", orderID),
		)
		return fmt.Errorf("не найдено сообщение для заказа: %s", orderID)
	}

	// Получаем обновленный заказ
	updatedOrder, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil {
		s.logger.Error("ошибка при получении обновленного заказа",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}

	// Обновляем сообщение в канале астрологов
	text := fmt.Sprintf(
		"🌟 *ЗАКАЗ ВЗЯТ В РАБОТУ* 🌟\n\n"+
			"*ID заказа:* `%s`\n"+
			"*Клиент:* %s\n"+
			"*Username:* @%s\n"+
			"*Дата заказа:* %s\n\n"+
			"*Взят в работу:* %s\n"+
			"*Астролог:* %s",
		updatedOrder.ID,
		updatedOrder.ClientName,
		updatedOrder.ClientUser,
		updatedOrder.CreatedAt.Format("02.01.2006 15:04"),
		updatedOrder.TakenAt.Format("02.01.2006 15:04"),
		astrologerName,
	)

	keyboard := tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
	}

	err = s.telegram.UpdateOrderMessage(s.channelID, messageID, text, keyboard)
	if err != nil {
		s.logger.Error("ошибка при обновлении сообщения о заказе",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}

	// Отправляем уведомление клиенту о том, что его запрос взят в работу
	err = s.telegram.SendMessage(updatedOrder.ClientID, fmt.Sprintf(
		"✨ Ваш запрос на астрологическую консультацию принят в работу! Астролог скоро свяжется с вами.",
	))
	if err != nil {
		s.logger.Error("ошибка при отправке уведомления клиенту",
			zap.Error(err),
			zap.Int64("client_id", updatedOrder.ClientID),
		)
	}

	s.logger.Info("заказ взят в работу",
		zap.String("order_id", orderID),
		zap.Int64("astrologer_id", astrologerID),
	)

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
