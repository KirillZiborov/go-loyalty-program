package models

import "time"

type User struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Order struct {
	OrderNumber string
	Status      string
	Accrual     *float32
	UploadedAt  time.Time
}

type OrderResponse struct {
	OrderNumber string   `json:"order_number"`
	Status      string   `json:"status"`
	Accrual     *float32 `json:"accrual,omitempty"`
	UploadedAt  string   `json:"uploaded_at"`
}
