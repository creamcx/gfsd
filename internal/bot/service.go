package bot

import (
	"go.uber.org/zap"
	"strings"
)

// NewService - создает новый экземпляр основного сервиса бота
func NewService(telegram TelegramClient, logger *zap.Logger, orderService *OrderService) *Service {
	return &Service{
		telegram:     telegram,
		logger:       logger,
		orderService: orderService,
	}
}

// Start - запускает обработку сообщений и callback-запросов
func (s *Service) Start() error {
	// Запускаем бота с единым обработчиком обновлений
	messagesChan, callbacksChan, err := s.telegram.StartBot()
	if err != nil {
		s.logger.Error("ошибка при запуске бота",
			zap.Error(err),
		)
		return err
	}

	// Обрабатываем callback-запросы (нажатия на кнопки) в отдельной горутине
	go func() {
		for callback := range callbacksChan {
			s.logger.Info("получен callback-запрос",
				zap.String("data", callback.Data),
				zap.Int64("user_id", callback.UserID),
			)

			// Обрабатываем запрос на взятие заказа в работу
			if strings.HasPrefix(callback.Data, "take_order:") {
				parts := strings.Split(callback.Data, ":")
				if len(parts) == 2 {
					orderID := parts[1]
					err := s.orderService.TakeOrder(orderID, callback.UserID, callback.UserName)
					if err != nil {
						s.logger.Error("ошибка при взятии заказа в работу",
							zap.Error(err),
							zap.String("order_id", orderID),
							zap.Int64("user_id", callback.UserID),
						)
					}
				}
			}
		}
	}()

	// Обрабатываем сообщения от пользователей
	for message := range messagesChan {
		s.logger.Info("получено сообщение",
			zap.Int64("chatid", message.ChatID),
			zap.String("text", message.Text),
		)

		// Обрабатываем обновление через обработчик команд
		if err := s.HandleUpdate(message); err != nil {
			s.logger.Error("ошибка при обработке сообщения",
				zap.Error(err),
				zap.Int64("chatid", message.ChatID),
			)
		}
	}

	return nil
}
