package database

import (
	"astro-sarafan/internal/models"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

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

func (r *UserRepository) GenerateReferralCode(chatID int64) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const codeLength = 8

	// Генерируем новый код
	referralCode := generateUniqueReferralCode(r.db)

	// Сохраняем код
	query := `
        UPDATE users 
        SET referral_code = $1 
        WHERE chat_id = $2
        RETURNING referral_code
    `

	var savedCode string
	err := r.db.QueryRow(query, referralCode, chatID).Scan(&savedCode)
	if err != nil {
		r.logger.Error("Ошибка при сохранении реферального кода",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
			zap.String("code", referralCode),
		)
		return "", err
	}

	r.logger.Info("Сгенерирован новый реферальный код",
		zap.Int64("chat_id", chatID),
		zap.String("code", savedCode),
	)

	return savedCode, nil
}

// Функция для генерации уникального кода
func generateUniqueReferralCode(db *sqlx.DB) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const codeLength = 8

	for attempts := 0; attempts < 10; attempts++ {
		// Генерируем код
		result := strings.Builder{}
		for i := 0; i < codeLength; i++ {
			result.WriteByte(charset[rand.Intn(len(charset))])
		}
		code := result.String()

		// Проверяем уникальность кода
		var count int
		err := db.Get(&count, "SELECT COUNT(*) FROM users WHERE referral_code = $1", code)
		if err != nil {
			// Если ошибка - логируем и продолжаем
			log.Printf("Ошибка проверки уникальности кода: %v", err)
			continue
		}

		// Если код уникален - возвращаем его
		if count == 0 {
			return code
		}
	}

	// Если не удалось сгенерировать уникальный код
	return fmt.Sprintf("%s%d",
		generateRandomString(6),
		time.Now().UnixNano()%10000,
	)
}

func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func (r *UserRepository) GetUserByReferralCode(referralCode string) (models.User, error) {
	var user models.User
	query := `SELECT chat_id, username, full_name, referral_code FROM users WHERE referral_code = $1`

	err := r.db.Get(&user, query, referralCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, nil // Пользователь не найден
		}
		r.logger.Error("Ошибка при получении пользователя по реферальному коду",
			zap.Error(err),
			zap.String("referral_code", referralCode),
		)
		return models.User{}, err
	}

	return user, nil
}

// Вспомогательный метод для логирования существующих реферальных кодов
func (r *UserRepository) logExistingReferralCodes(requestedCode string) {
	query := `SELECT referral_code FROM users WHERE referral_code IS NOT NULL LIMIT 10`
	var existingCodes []string

	err := r.db.Select(&existingCodes, query)
	if err != nil {
		r.logger.Error("Ошибка при получении существующих реферальных кодов",
			zap.Error(err),
		)
		return
	}

	r.logger.Warn("Код не найден. Существующие коды:",
		zap.Strings("existing_codes", existingCodes),
		zap.String("requested_code", requestedCode),
	)
}

func (r *UserRepository) GetUserByID(chatID int64) (models.User, error) {
	var user models.User
	query := `SELECT chat_id, username, full_name, referral_code FROM users WHERE chat_id = $1`

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
