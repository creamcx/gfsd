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
	// ErrConsultationExists ошибка, когда у пользователя уже есть консультация
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

func (r *OrderRepository) CreateOrder(order models.Order) error {
	// Начинаем транзакцию
	tx, err := r.db.Beginx()
	if err != nil {
		r.logger.Error("Ошибка при начале транзакции", zap.Error(err))
		return err
	}
	defer tx.Rollback() // Откатываем транзакцию в случае ошибки

	// Проверяем, есть ли у пользователя уже заказ
	var count int
	err = tx.Get(&count, "SELECT COUNT(*) FROM orders WHERE client_id = $1", order.ClientID)
	if err != nil {
		r.logger.Error("Ошибка при проверке существующих заказов",
			zap.Error(err),
			zap.Int64("client_id", order.ClientID),
		)
		return err
	}

	// Если у пользователя уже есть заказ, возвращаем ошибку
	if count > 0 {
		r.logger.Info("Попытка создать второй заказ",
			zap.Int64("client_id", order.ClientID),
		)
		return ErrConsultationExists
	}

	// Создаем заказ с учетом реферера
	query := `
		INSERT INTO orders (id, client_id, status, created_at, referrer_id, referrer_name)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = tx.Exec(query, order.ID, order.ClientID, order.Status, order.CreatedAt,
		order.ReferrerID, order.ReferrerName)
	if err != nil {
		r.logger.Error("Ошибка при создании заказа",
			zap.Error(err),
			zap.String("order_id", order.ID),
			zap.Int64("client_id", order.ClientID),
		)
		return err
	}

	// Фиксируем транзакцию
	if err = tx.Commit(); err != nil {
		r.logger.Error("Ошибка при фиксации транзакции", zap.Error(err))
		return err
	}

	return nil
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
            astrologer_id = $3, 
            astrologer_name = $4
        WHERE id = $5 AND status = $6
    `

	result, err := tx.Exec(
		query,
		status,
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

// GetOrdersForReminder получает заказы, которым нужно отправить напоминание
func (r *OrderRepository) GetOrdersForReminder(reminderThreshold time.Duration) ([]models.Order, error) {
	var orders []models.Order
	thresholdTime := time.Now().Add(-reminderThreshold)

	query := `
        SELECT 
            o.id, o.client_id, o.status, o.created_at, o.taken_at, 
            COALESCE(o.astrologer_id, 0) as astrologer_id, 
            COALESCE(o.astrologer_name, '') as astrologer_name,
            COALESCE(u.username, '') as client_user, 
            COALESCE(u.full_name, '') as client_name,
            o.button_pressed, o.reminder_sent_at, 
            COALESCE(o.pdf_url, '') as pdf_url,
            o.pdf_sent_at
        FROM orders o
        LEFT JOIN users u ON o.client_id = u.chat_id
        WHERE o.status = $1 
          AND o.taken_at <= $2 
          AND o.button_pressed = FALSE 
          AND (o.reminder_sent_at IS NULL OR o.reminder_sent_at <= $3)
          AND o.pdf_url IS NOT NULL 
          AND o.pdf_sent_at IS NOT NULL
    `

	// Считаем, что повторное напоминание можно отправить через 24 часа после предыдущего
	repeatReminderTime := time.Now().Add(-24 * time.Hour)

	err := r.db.Select(&orders, query, models.OrderStatusInWork, thresholdTime, repeatReminderTime)
	if err != nil {
		r.logger.Error("Ошибка при получении заказов для напоминания",
			zap.Error(err),
			zap.Duration("threshold", reminderThreshold))
		return nil, err
	}

	return orders, nil
}

// UpdateReminderSent обновляет время отправки напоминания
func (r *OrderRepository) UpdateReminderSent(orderID string) error {
	now := time.Now()
	query := `UPDATE orders SET reminder_sent_at = $1 WHERE id = $2`

	_, err := r.db.Exec(query, now, orderID)
	if err != nil {
		r.logger.Error("Ошибка при обновлении времени отправки напоминания",
			zap.Error(err),
			zap.String("order_id", orderID))
		return err
	}

	return nil
}

// UpdatePDFInfo обновляет информацию о PDF консультации
func (r *OrderRepository) UpdatePDFInfo(orderID string, pdfURL string) error {
	now := time.Now()
	query := `UPDATE orders SET pdf_url = $1, pdf_sent_at = $2 WHERE id = $3`

	_, err := r.db.Exec(query, pdfURL, now, orderID)
	if err != nil {
		r.logger.Error("Ошибка при обновлении информации о PDF",
			zap.Error(err),
			zap.String("order_id", orderID),
			zap.String("pdf_url", pdfURL))
		return err
	}

	return nil
}

// UpdateButtonPressed обновляет статус нажатия кнопки
func (r *OrderRepository) UpdateButtonPressed(orderID string) error {
	query := `UPDATE orders SET button_pressed = TRUE WHERE id = $1`

	_, err := r.db.Exec(query, orderID)
	if err != nil {
		r.logger.Error("Ошибка при обновлении статуса нажатия кнопки",
			zap.Error(err),
			zap.String("order_id", orderID))
		return err
	}

	r.logger.Info("Обновлен статус нажатия кнопки",
		zap.String("order_id", orderID),
		zap.Bool("button_pressed", true))

	return nil
}

// GetOrdersInWorkWithoutPDF получает заказы в статусе "in_work" без URL PDF
func (r *OrderRepository) GetOrdersInWorkWithoutPDF() ([]models.Order, error) {
	var orders []models.Order
	query := `
        SELECT 
            o.id, o.client_id, o.status, o.created_at, o.taken_at, 
            COALESCE(o.astrologer_id, 0) as astrologer_id, 
            COALESCE(o.astrologer_name, '') as astrologer_name,
            COALESCE(u.username, '') as client_user, 
            COALESCE(u.full_name, '') as client_name,
            o.button_pressed, o.reminder_sent_at, 
            COALESCE(o.pdf_url, '') as pdf_url,
            o.pdf_sent_at
        FROM orders o
        LEFT JOIN users u ON o.client_id = u.chat_id
        WHERE o.status = $1 AND (o.pdf_url IS NULL OR o.pdf_url = '')
        ORDER BY o.taken_at
    `

	err := r.db.Select(&orders, query, models.OrderStatusInWork)
	if err != nil {
		r.logger.Error("Ошибка при получении заказов без PDF", zap.Error(err))
		return nil, err
	}

	return orders, nil
}
