package models

import "time"

type OrderStatus string

const (
	OrderStatusNew      OrderStatus = "new"
	OrderStatusInWork   OrderStatus = "in_work"
	OrderStatusComplete OrderStatus = "complete"
)

type User struct {
	ChatID   int64
	Text     string
	Username string
	FullName string
}

type Order struct {
	ID           string      `json:"id"`
	ClientID     int64       `json:"client_id"`   // ID пользователя в Telegram
	ClientName   string      `json:"client_name"` // Имя пользователя
	ClientUser   string      `json:"client_user"` // Username пользователя
	Status       OrderStatus `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
	AstrologerID int64       `json:"astrologer_id,omitempty"` // ID астролога, который взял заказ
	TakenAt      *time.Time  `json:"taken_at,omitempty"`      // Когда заказ был взят в работу
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
