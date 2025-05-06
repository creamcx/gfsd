package bot

import (
	"astro-sarafan/internal/database"
	"astro-sarafan/internal/models"
	"go.uber.org/zap"
	"strings"
	"time"
)

// NewService - создает новый экземпляр основного сервиса бота
func NewService(telegram TelegramClient, logger *zap.Logger, orderService *OrderService, userRepo *database.UserRepository) *Service {
	return &Service{
		telegram:     telegram,
		logger:       logger,
		orderService: orderService,
		userRepo:     userRepo,
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
			} else if callback.Data == "consultation_continue" {
				s.logger.Info("получен запрос на полную консультацию",
					zap.Int64("user_id", callback.UserID),
				)

				// Получаем информацию о пользователе
				user, err := s.userRepo.GetUserByID(callback.UserID)
				if err != nil {
					s.logger.Error("ошибка при получении пользователя",
						zap.Error(err),
						zap.Int64("user_id", callback.UserID),
					)
					continue
				}

				// Создаем или обновляем заказ пользователя
				orderID, err := s.orderService.ProcessFullConsultationRequest(
					callback.UserID,
					user.FullName,
					user.Username,
				)

				if err != nil {
					s.logger.Error("ошибка при обработке запроса на полную консультацию",
						zap.Error(err),
						zap.Int64("user_id", callback.UserID),
					)

					// Информируем пользователя об ошибке
					s.telegram.SendMessage(callback.UserID, "Произошла ошибка при обработке запроса на полную консультацию. Пожалуйста, попробуйте позже.")
					continue
				}

				// Отправляем подтверждение пользователю
				s.telegram.SendMessage(callback.UserID, "✨ Благодарим за ваш запрос на полную астрологическую консультацию! Наш астролог получит уведомление и свяжется с вами в ближайшее время.")

				s.logger.Info("заказ на полную консультацию обработан",
					zap.Int64("user_id", callback.UserID),
					zap.String("order_id", orderID),
				)
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

// Добавьте в internal/bot/order_service.go
func (s *OrderService) CreateFullConsultationOrder(clientID int64, clientName, clientUser string) (string, error) {
	// Логируем запрос
	s.logger.Info("Создание заказа на полную консультацию",
		zap.Int64("client_id", clientID),
		zap.String("client_name", clientName),
		zap.String("client_user", clientUser),
	)

	// Сохраняем пользователя (если ещё не существует)
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

	// Создаем заказ с маркером полной консультации
	order := models.Order{
		ID:                 orderID,
		ClientID:           clientID,
		Status:             models.OrderStatusNew,
		CreatedAt:          time.Now(),
		ConsultationStatus: "full", // Отмечаем, что это полная консультация
	}

	// Проверяем, есть ли у пользователя уже заказ
	hasConsultation, err := s.userRepo.HasActiveConsultation(clientID)
	if err != nil {
		s.logger.Error("ошибка при проверке наличия консультации",
			zap.Error(err),
			zap.Int64("client_id", clientID),
		)
		return "", err
	}

	// Если у пользователя уже есть консультация, обновляем её до полной
	if hasConsultation {
		err = s.orderRepo.UpdateConsultationToFull(clientID)
		if err != nil {
			s.logger.Error("ошибка при обновлении консультации до полной",
				zap.Error(err),
				zap.Int64("client_id", clientID),
			)
			return "", err
		}

		// Получаем ID существующего заказа для отправки уведомления
		existingOrder, err := s.orderRepo.GetOrderByClientID(clientID)
		if err != nil {
			s.logger.Error("ошибка при получении существующего заказа",
				zap.Error(err),
				zap.Int64("client_id", clientID),
			)
			return "", err
		}

		orderID = existingOrder.ID
	} else {
		// Создаем новый заказ
		err = s.orderRepo.CreateOrder(order)
		if err != nil {
			s.logger.Error("ошибка при создании заказа",
				zap.Error(err),
				zap.String("order_id", orderID),
			)
			return "", err
		}
	}

	// Обновляем данные для отправки в канал
	orderWithClient := models.Order{
		ID:                 orderID,
		ClientID:           clientID,
		ClientName:         clientName,
		ClientUser:         clientUser,
		Status:             models.OrderStatusNew,
		CreatedAt:          time.Now(),
		ConsultationStatus: "full",
	}

	// Отправляем уведомление в канал астрологов
	messageID, err := s.telegram.SendFullConsultationToAstrologers(s.channelID, orderWithClient)
	if err != nil {
		s.logger.Error("ошибка при отправке заказа на полную консультацию в канал астрологов",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return orderID, err
	}

	// Сохраняем ID сообщения для возможности обновления
	s.orderMessages[orderID] = messageID

	s.logger.Info("создан заказ на полную консультацию",
		zap.String("order_id", orderID),
		zap.Int64("client_id", clientID),
		zap.String("message_id", messageID),
	)

	return orderID, nil
}
