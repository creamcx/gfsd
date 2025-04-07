package bot

import (
	"astro-sarafan/internal/database"
	"time"

	"go.uber.org/zap"
)

// ReminderService отвечает за отправку напоминаний пользователям,
// которые не нажали кнопку "Купить консультацию" в течение определенного времени
type ReminderService struct {
	orderRepo    *database.OrderRepository
	telegram     TelegramClient
	logger       *zap.Logger
	checkPeriod  time.Duration // Период проверки заказов (например, каждый час)
	reminderTime time.Duration // Время, после которого отправляется напоминание (24 часа)
}

// NewReminderService создает новый сервис напоминаний
func NewReminderService(
	orderRepo *database.OrderRepository,
	telegram TelegramClient,
	logger *zap.Logger,
) *ReminderService {
	return &ReminderService{
		orderRepo:    orderRepo,
		telegram:     telegram,
		logger:       logger,
		checkPeriod:  30 * time.Minute, // Проверка каждые 30 минут
		reminderTime: 24 * time.Hour,   // Напоминание через 24 часа
	}
}

// Start запускает сервис напоминаний
func (s *ReminderService) Start() {
	s.logger.Info("Запуск сервиса напоминаний")
	go s.reminderLoop()
}

// reminderLoop запускает цикл проверки заказов и отправки напоминаний
func (s *ReminderService) reminderLoop() {
	ticker := time.NewTicker(s.checkPeriod)
	defer ticker.Stop()

	for range ticker.C {
		s.checkAndSendReminders()
	}
}

// checkAndSendReminders проверяет заказы и отправляет напоминания
func (s *ReminderService) checkAndSendReminders() {
	s.logger.Debug("Проверка заказов для отправки напоминаний")

	// Получаем заказы, которые нуждаются в напоминании
	orders, err := s.orderRepo.GetOrdersForReminder(s.reminderTime)
	if err != nil {
		s.logger.Error("Ошибка при получении заказов для напоминаний",
			zap.Error(err))
		return
	}

	if len(orders) == 0 {
		s.logger.Debug("Нет заказов, требующих напоминания")
		return
	}

	s.logger.Info("Найдены заказы для отправки напоминаний",
		zap.Int("count", len(orders)))

	// Отправляем напоминания
	for _, order := range orders {
		s.sendReminder(order.ID, order.ClientID)
	}
}

// sendReminder отправляет напоминание клиенту
func (s *ReminderService) sendReminder(orderID string, clientID int64) {
	// Текст напоминания
	reminderText := `⏰ Здравствуйте!

Напоминаем, что для вас готова астрологическая консультация.
Для продолжения сотрудничества и получения полного разбора нажмите на кнопку "КУПИТЬ КОНСУЛЬТАЦИЮ" в вашем PDF-документе.

Ваш астролог будет рад помочь вам раскрыть весь потенциал гороскопа!`

	// Отправляем напоминание
	err := s.telegram.SendMessage(clientID, reminderText)
	if err != nil {
		s.logger.Error("Ошибка при отправке напоминания",
			zap.Error(err),
			zap.Int64("client_id", clientID),
			zap.String("order_id", orderID))
		return
	}

	// Обновляем время отправки напоминания
	err = s.orderRepo.UpdateReminderSent(orderID)
	if err != nil {
		s.logger.Error("Ошибка при обновлении времени отправки напоминания",
			zap.Error(err),
			zap.String("order_id", orderID))
		return
	}

	s.logger.Info("Напоминание успешно отправлено",
		zap.Int64("client_id", clientID),
		zap.String("order_id", orderID))
}
