package database

import (
	"astro-sarafan/internal/models"
	"database/sql"
	"errors"
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

// GetOrderByID получает заказ по ID
func (r *OrderRepository) GetOrderByID(orderID string) (models.Order, error) {
	var order models.Order
	query := `
		SELECT id, client_id, status, created_at, taken_at, astrologer_id
		FROM orders
		WHERE id = $1
	`

	err := r.db.Get(&order, query, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Order{}, nil // Заказ не найден
		}
		r.logger.Error("Ошибка при получении заказа",
			zap.Error(err),
			zap.String("order_id", orderID),
		)
		return models.Order{}, err
	}

	return order, nil
}

// UpdateOrderStatus обновляет статус заказа
func (r *OrderRepository) UpdateOrderStatus(orderID string, status models.OrderStatus, astrologerID int64, astrologerName string) error {
	var query string
	var args []interface{}

	if status == models.OrderStatusInWork {
		now := time.Now()
		query = `
			UPDATE orders
			SET status = $1, taken_at = $2, astrologer_id = $3, astrologer_name = $4
			WHERE id = $5
		`
		args = []interface{}{status, now, astrologerID, astrologerName, orderID}
	} else {
		query = `
			UPDATE orders
			SET status = $1
			WHERE id = $2
		`
		args = []interface{}{status, orderID}
	}

	_, err := r.db.Exec(query, args...)
	if err != nil {
		r.logger.Error("Ошибка при обновлении статуса заказа",
			zap.Error(err),
			zap.String("order_id", orderID),
			zap.String("status", string(status)),
		)
		return err
	}

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
