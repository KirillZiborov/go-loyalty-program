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
	Accrual     *int
	UploadedAt  time.Time
}

type OrderRequest struct {
	OrderNumber string `json:"order"`
}

type OrderResponse struct {
	OrderNumber string `json:"order_number"`
	Status      string `json:"status"`
	Accrual     *int   `json:"accrual,omitempty"`
	UploadedAt  string `json:"uploaded_at"`
}
