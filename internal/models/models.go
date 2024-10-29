package models

import "time"

type User struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Order struct {
	UserID      int
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

type Balance struct {
	Current   float32
	Withdrawn float32
}

type BalanceResponse struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

type WithdrawRequest struct {
	OrderNumber string  `json:"order"`
	Sum         float32 `json:"sum"`
}

type Withdrawal struct {
	OrderNumber string    `json:"order"`
	Sum         float32   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}
