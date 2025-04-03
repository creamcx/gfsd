package bot

import (
	"astro-sarafan/internal/models"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// NewOrderService - создает новый сервис заказов
func NewOrderService(telegram TelegramClient, logger *zap.Logger, channelID string) *OrderService {
	return &OrderService{
		orders:        make(map[string]models.Order),
		orderMessages: make(map[string]string),
		telegram:      telegram,
		logger:        logger,
		channelID:     channelID,
	}
}

// Mutex для синхронизации доступа к данным заказов
var orderMutex sync.RWMutex

// CreateOrder - создает новый заказ на консультацию
func (s *OrderService) CreateOrder(clientID int64, clientName, clientUser string) (string, error) {
	orderMutex.Lock()
	defer orderMutex.Unlock()

	// Генерируем уникальный ID заказа
	orderID := generateOrderID()

	// Создаем заказ
	order := models.Order{
		ID:         orderID,
		ClientID:   clientID,
		ClientName: clientName,
		ClientUser: clientUser,
		Status:     models.OrderStatusNew,
		CreatedAt:  time.Now(),
	}

	// Сохраняем заказ в хранилище
	s.orders[orderID] = order

	// Отправляем уведомление в канал астрологов
	messageID, err := s.telegram.SendOrderToAstrologers(s.channelID, order)
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
	)

	return orderID, nil
}

// TakeOrder - взятие заказа в работу астрологом
func (s *OrderService) TakeOrder(orderID string, astrologerID int64, astrologerName string) error {
	orderMutex.Lock()
	defer orderMutex.Unlock()

	// Проверяем, существует ли заказ
	order, exists := s.orders[orderID]
	if !exists {
		return fmt.Errorf("заказ не найден: %s", orderID)
	}

	// Проверяем, не взят ли заказ уже в работу
	if order.Status != models.OrderStatusNew {
		return fmt.Errorf("заказ уже взят в работу или завершен: %s", orderID)
	}

	// Обновляем статус заказа
	now := time.Now()
	order.Status = models.OrderStatusInWork
	order.AstrologerID = astrologerID
	order.TakenAt = &now

	// Сохраняем обновленный заказ
	s.orders[orderID] = order

	// Получаем ID сообщения в канале астрологов
	messageID, exists := s.orderMessages[orderID]
	if !exists {
		s.logger.Error("не найдено сообщение для заказа",
			zap.String("order_id", orderID),
		)
		return fmt.Errorf("не найдено сообщение для заказа: %s", orderID)
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
		order.ID,
		order.ClientName,
		order.ClientUser,
		order.CreatedAt.Format("02.01.2006 15:04"),
		now.Format("02.01.2006 15:04"),
		astrologerName,
	)

	keyboard := tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
	}

	err := s.telegram.UpdateOrderMessage(s.channelID, messageID, text, keyboard)
	if err != nil {
		s.logger.Error("ошибка при обновлении сообщения о заказе",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}

	// Отправляем уведомление клиенту о том, что его запрос взят в работу
	err = s.telegram.SendMessage(order.ClientID, fmt.Sprintf(
		"✨ Ваш запрос на астрологическую консультацию принят в работу! Астролог скоро свяжется с вами.",
	))
	if err != nil {
		s.logger.Error("ошибка при отправке уведомления клиенту",
			zap.Error(err),
			zap.Int64("client_id", order.ClientID),
		)
	}

	s.logger.Info("заказ взят в работу",
		zap.String("order_id", orderID),
		zap.Int64("astrologer_id", astrologerID),
	)

	return nil
}

// GetOrder - получение заказа по ID
func (s *OrderService) GetOrder(orderID string) (models.Order, bool) {
	orderMutex.RLock()
	defer orderMutex.RUnlock()

	order, exists := s.orders[orderID]
	return order, exists
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
