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

// Order представляет собой заказ на консультацию
type Order struct {
	ID             string      `db:"id" json:"id"`               // ID заказа
	ClientID       int64       `db:"client_id" json:"client_id"` // ID клиента в Telegram
	ClientName     string      `db:"client_name" json:"client_name"`
	ClientUser     string      `db:"client_user" json:"client_user"`
	Status         OrderStatus `db:"status" json:"status"`         // Статус заказа
	CreatedAt      time.Time   `db:"created_at" json:"created_at"` // Время создания заказа
	AstrologerID   int64       `db:"astrologer_id" json:"astrologer_id,omitempty"`
	AstrologerName string      `db:"astrologer_name" json:"astrologer_name,omitempty"`
	TakenAt        *time.Time  `db:"taken_at" json:"taken_at,omitempty"` // Время взятия заказа в работу
	ReferrerID     int64       `db:"referrer_id" json:"referrer_id,omitempty"`
	ReferrerName   string      `db:"referrer_name" json:"referrer_name,omitempty"`

	// Добавляем новые поля
	ButtonPressed  bool       `db:"button_pressed" json:"button_pressed"`               // Флаг нажатия кнопки
	ReminderSentAt *time.Time `db:"reminder_sent_at" json:"reminder_sent_at,omitempty"` // Время отправки напоминания
	PDFURL         string     `db:"pdf_url" json:"pdf_url,omitempty"`                   // URL PDF-документа
	PDFSentAt      *time.Time `db:"pdf_sent_at" json:"pdf_sent_at,omitempty"`           // Время отправки PDF
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
