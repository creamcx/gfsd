package models

import (
	"database/sql"
	"time"
)

type OrderStatus string

const (
	OrderStatusNew      OrderStatus = "new"
	OrderStatusInWork   OrderStatus = "in_work"
	OrderStatusComplete OrderStatus = "complete"
)

type User struct {
	ChatID       int64          `db:"chat_id"`
	Username     string         `db:"username"`
	FullName     string         `db:"full_name"`
	ReferralCode sql.NullString `db:"referral_code"`
	Text         string         `db:"-"`
	Contact      *Contact       `db:"-"`
}

type Contact struct {
	PhoneNumber string
	FirstName   string
	LastName    string
}

// В файле internal/models/user.go

type Order struct {
	ID                    string      `db:"id" json:"id"`
	ClientID              int64       `db:"client_id" json:"client_id"`
	ClientName            string      `db:"client_name" json:"-"`
	ClientUser            string      `db:"client_user" json:"-"`
	Status                OrderStatus `db:"status" json:"status"`
	CreatedAt             time.Time   `db:"created_at" json:"created_at"`
	AstrologerID          int64       `db:"astrologer_id"`
	AstrologerName        string      `db:"astrologer_name" json:"astrologer_name,omitempty"`
	TakenAt               *time.Time  `db:"taken_at" json:"taken_at,omitempty"`
	ReferrerID            int64       `db:"referrer_id" json:"referrer_id,omitempty"`
	ReferrerName          string      `db:"referrer_name" json:"referrer_name,omitempty"`
	ConsultationStartedAt time.Time   `db:"consultation_started_at"`
	ConsultationStatus    string      `db:"consultation_status"`
	NotificationSent      bool        `db:"notification_sent" json:"notification_sent"`
}

type CallbackQuery struct {
	ID          string // ID callback запроса
	UserID      int64  // ID пользователя, который нажал на кнопку
	UserName    string // Имя пользователя
	UserLogin   string // Логин пользователя в Telegram
	MessageID   string // ID сообщения, в котором была нажата кнопка
	ChatID      string // ID чата или канала, где был нажат callback
	Data        string // Данные callback запроса (например, "take_order:123")
	MessageText string // Текст сообщения
}
