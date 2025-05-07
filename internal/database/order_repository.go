package database

import (
	"astro-sarafan/internal/models"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

var (
	ErrConsultationExists = errors.New("у пользователя уже есть консультация")
)

// OrderRepository представляет репозиторий для работы с заказами
type OrderRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewOrderRepository создает новый репозиторий заказов
func NewOrderRepository(db *sqlx.DB, logger *zap.Logger) *OrderRepository {
	return &OrderRepository{
		db:     db,
		logger: logger,
	}
}

func (r *OrderRepository) GetActiveOrdersByClientID(clientID int64) ([]models.Order, error) {
	query := `
        SELECT 
            o.id, 
            o.client_id, 
            o.status, 
            o.created_at, 
            o.taken_at, 
            COALESCE(o.astrologer_id, 0) as astrologer_id,
            COALESCE(o.astrologer_name, '') as astrologer_name,
            COALESCE(u.username, '') as client_user, 
            COALESCE(u.full_name, '') as client_name,
            o.referrer_id,
            o.referrer_name,
            o.consultation_started_at,
            COALESCE(o.consultation_status, 'demo') as consultation_status,
            COALESCE(o.notification_sent, false) as notification_sent
        FROM orders o
        LEFT JOIN users u ON o.client_id = u.chat_id
        WHERE o.client_id = $1
        AND o.status IN ('new', 'in_work')
        AND (o.consultation_status = 'demo' OR o.consultation_status IS NULL)
        ORDER BY o.created_at DESC
    `

	var orders []models.Order
	err := r.db.Select(&orders, query, clientID)
	if err != nil {
		r.logger.Error("Ошибка при получении активных заказов по ID клиента",
			zap.Error(err),
			zap.Int64("client_id", clientID),
		)
		return nil, err
	}

	return orders, nil
}

func (r *OrderRepository) CreateOrder(order models.Order) error {
	// Начинаем транзакцию
	tx, err := r.db.Beginx()
	if err != nil {
		r.logger.Error("Ошибка при начале транзакции", zap.Error(err))
		return err
	}
	defer tx.Rollback() // Откатываем транзакцию в случае ошибки

	// Если это демо-консультация, проверяем флаг demo_used у пользователя
	if order.ConsultationStatus == "demo" || order.ConsultationStatus == "" {
		var demoUsed bool
		err = tx.Get(&demoUsed, "SELECT demo_used FROM users WHERE chat_id = $1", order.ClientID)
		if err != nil {
			r.logger.Error("Ошибка при проверке использования демо-консультации",
				zap.Error(err),
				zap.Int64("client_id", order.ClientID),
			)
			return err
		}

		if demoUsed {
			r.logger.Info("Пользователь уже использовал демо-консультацию",
				zap.Int64("client_id", order.ClientID),
			)
			return ErrConsultationExists
		}

		// Если пользователь еще не использовал демо-консультацию,
		// устанавливаем флаг demo_used = true
		_, err = tx.Exec("UPDATE users SET demo_used = true WHERE chat_id = $1", order.ClientID)
		if err != nil {
			r.logger.Error("Ошибка при установке флага demo_used",
				zap.Error(err),
				zap.Int64("client_id", order.ClientID),
			)
			return err
		}
	}

	// Создаем заказ
	insertQuery := `
        INSERT INTO orders (
            id, client_id, status, created_at, referrer_id, 
            referrer_name, consultation_status
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

	_, err = tx.Exec(
		insertQuery,
		order.ID,
		order.ClientID,
		order.Status,
		order.CreatedAt,
		order.ReferrerID,
		order.ReferrerName,
		order.ConsultationStatus,
	)

	if err != nil {
		r.logger.Error("Ошибка при создании заказа",
			zap.Error(err),
			zap.String("order_id", order.ID))
		return err
	}

	// Фиксируем транзакцию
	if err = tx.Commit(); err != nil {
		r.logger.Error("Ошибка при фиксации транзакции", zap.Error(err))
		return err
	}

	return nil
}

func (r *OrderRepository) UpdateOrderToFullConsultation(orderID string) error {
	query := `
        UPDATE orders 
        SET consultation_status = 'full',
            status = 'new',  -- Сбрасываем статус до "new", чтобы астролог мог взять заказ
            astrologer_id = NULL,  -- Сбрасываем ID астролога
            astrologer_name = NULL  -- Сбрасываем имя астролога
        WHERE id = $1
    `

	_, err := r.db.Exec(query, orderID)
	if err != nil {
		r.logger.Error("Ошибка при обновлении заказа до полной консультации",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}

	return nil
}

// Добавьте в internal/database/order_repository.go
func (r *OrderRepository) GetOrdersByClientID(clientID int64) ([]models.Order, error) {
	query := `
        SELECT 
            o.id, 
            o.client_id, 
            o.status, 
            o.created_at, 
            o.taken_at, 
            COALESCE(o.astrologer_id, 0) as astrologer_id,
            COALESCE(o.astrologer_name, '') as astrologer_name,
            COALESCE(u.username, '') as client_user, 
            COALESCE(u.full_name, '') as client_name
        FROM orders o
        LEFT JOIN users u ON o.client_id = u.chat_id
        WHERE o.client_id = $1
        ORDER BY o.created_at DESC
    `

	var orders []models.Order
	err := r.db.Select(&orders, query, clientID)
	if err != nil {
		r.logger.Error("Ошибка при получении заказов по ID клиента",
			zap.Error(err),
			zap.Int64("client_id", clientID),
		)
		return nil, err
	}

	return orders, nil
}

func (r *OrderRepository) GetOrderByID(orderID string) (models.Order, error) {
	var order models.Order
	query := `
        SELECT 
            o.id, 
            o.client_id, 
            o.status, 
            o.created_at, 
            o.taken_at, 
            COALESCE(o.astrologer_id, 0) as astrologer_id,
            COALESCE(o.astrologer_name, '') as astrologer_name,
            COALESCE(u.username, '') as client_user, 
            COALESCE(u.full_name, '') as client_name
        FROM orders o
        LEFT JOIN users u ON o.client_id = u.chat_id
        WHERE o.id = $1
    `

	err := r.db.Get(&order, query, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Warn("Заказ не найден",
				zap.String("order_id", orderID),
			)
			return models.Order{}, nil
		}
		r.logger.Error("Ошибка при получении заказа",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return models.Order{}, err
	}

	// Логируем полученную информацию о заказе
	r.logger.Info("Получен заказ",
		zap.String("order_id", order.ID),
		zap.Int64("client_id", order.ClientID),
		zap.String("client_name", order.ClientName),
		zap.String("client_user", order.ClientUser),
	)

	return order, nil
}

func (r *OrderRepository) UpdateOrderStatus(orderID string, status models.OrderStatus, astrologerID int64, astrologerName string) error {
	// Начинаем транзакцию
	tx, err := r.db.Beginx()
	if err != nil {
		r.logger.Error("Ошибка при начале транзакции",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}
	defer tx.Rollback() // Откатываем транзакцию в случае ошибки

	// Проверяем текущий статус заказа
	var currentStatus string
	err = tx.Get(&currentStatus, "SELECT status FROM orders WHERE id = $1", orderID)
	if err != nil {
		r.logger.Error("Ошибка при проверке текущего статуса заказа",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}

	// Логируем текущий статус
	r.logger.Info("Текущий статус заказа",
		zap.String("order_id", orderID),
		zap.String("current_status", currentStatus),
	)

	// Проверяем, можно ли изменить статус
	if currentStatus != string(models.OrderStatusNew) {
		r.logger.Warn("Попытка изменить статус неактивного заказа",
			zap.String("order_id", orderID),
			zap.String("current_status", currentStatus),
			zap.String("new_status", string(status)),
		)
		return fmt.Errorf("заказ уже изменен: %s", orderID)
	}

	// Обновляем статус заказа
	now := time.Now()
	query := `
        UPDATE orders
        SET 
            status = $1, 
            taken_at = $2, 
            consultation_started_at = $3,
            astrologer_id = $4, 
            astrologer_name = $5
        WHERE id = $6 AND status = $7
    `

	result, err := tx.Exec(
		query,
		status,
		now,
		now,
		astrologerID,
		astrologerName,
		orderID,
		models.OrderStatusNew,
	)
	if err != nil {
		r.logger.Error("Ошибка при обновлении статуса заказа",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}

	// Проверяем, что строка была обновлена
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Ошибка при проверке количества обновленных строк",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}

	if rowsAffected == 0 {
		r.logger.Warn("Не удалось обновить статус заказа",
			zap.String("order_id", orderID),
		)
		return fmt.Errorf("не удалось обновить статус заказа: %s", orderID)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		r.logger.Error("Ошибка при фиксации транзакции",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return err
	}

	r.logger.Info("Статус заказа успешно обновлен",
		zap.String("order_id", orderID),
		zap.String("new_status", string(status)),
	)

	return nil
}

// GetAllOrders получает все заказы
func (r *OrderRepository) GetAllOrders() ([]models.Order, error) {
	var orders []models.Order
	query := `
		SELECT o.id, o.client_id, o.status, o.created_at, o.taken_at, o.astrologer_id, o.astrologer_name,
			   u.username as client_user, u.full_name as client_name
		FROM orders o
		JOIN users u ON o.client_id = u.chat_id
		ORDER BY o.created_at DESC
	`

	err := r.db.Select(&orders, query)
	if err != nil {
		r.logger.Error("Ошибка при получении всех заказов", zap.Error(err))
		return nil, err
	}

	return orders, nil
}

func (r *OrderRepository) GetActiveOrdersForNotification() ([]models.Order, error) {
	query := `
        SELECT o.id, o.client_id, o.status, o.created_at, o.taken_at, 
               COALESCE(o.astrologer_id, 0) as astrologer_id,
               COALESCE(o.astrologer_name, '') as astrologer_name,
               u.username as client_user, u.full_name as client_name,
               o.referrer_id, o.referrer_name, o.consultation_started_at,
               o.consultation_status, o.notification_sent
        FROM orders o
        JOIN users u ON o.client_id = u.chat_id
        WHERE o.consultation_started_at < NOW() - INTERVAL '1 minutes'
          AND o.status = 'in_work'
          AND (o.consultation_status = 'demo' OR o.consultation_status IS NULL)
          AND (o.notification_sent = false OR o.notification_sent IS NULL)
    `
	var orders []models.Order
	err := r.db.Select(&orders, query)
	if err != nil {
		r.logger.Error("Ошибка при получении заказов для уведомлений",
			zap.Error(err),
		)
	}
	return orders, err
}

func (r *OrderRepository) MarkNotificationSent(orderID string) error {
	query := `
        UPDATE orders 
        SET notification_sent = true 
        WHERE id = $1
    `
	_, err := r.db.Exec(query, orderID)
	if err != nil {
		r.logger.Error("Ошибка при отметке уведомления как отправленного",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
	}
	return err
}
