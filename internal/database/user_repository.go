package database

import (
	"astro-sarafan/internal/models"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// UserRepository представляет репозиторий для работы с пользователями
type UserRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewUserRepository создает новый репозиторий пользователей
func NewUserRepository(db *sqlx.DB, logger *zap.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

// CreateUser создает нового пользователя в базе данных
func (r *UserRepository) CreateUser(user models.User) error {
	query := `
		INSERT INTO users (chat_id, username, full_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (chat_id) DO UPDATE SET
			username = EXCLUDED.username,
			full_name = EXCLUDED.full_name
	`

	_, err := r.db.Exec(query, user.ChatID, user.Username, user.FullName)
	if err != nil {
		r.logger.Error("Ошибка при создании/обновлении пользователя",
			zap.Error(err),
			zap.Int64("chat_id", user.ChatID),
		)
		return err
	}

	return nil
}

// GetUserByID получает пользователя по chat_id
func (r *UserRepository) GetUserByID(chatID int64) (models.User, error) {
	var user models.User
	query := `SELECT chat_id, username, full_name FROM users WHERE chat_id = $1`

	err := r.db.Get(&user, query, chatID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, nil // Пользователь не найден
		}
		r.logger.Error("Ошибка при получении пользователя",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
		return models.User{}, err
	}

	return user, nil
}

// HasActiveConsultation проверяет, есть ли у пользователя активная консультация
func (r *UserRepository) HasActiveConsultation(chatID int64) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM orders WHERE client_id = $1`

	err := r.db.Get(&count, query, chatID)
	if err != nil {
		r.logger.Error("Ошибка при проверке наличия консультации",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
		return false, err
	}

	return count > 0, nil
}
