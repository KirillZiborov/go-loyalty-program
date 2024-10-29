package database

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/KirillZiborov/go-loyalty-program/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateUsersTable(ctx context.Context, db *pgxpool.Pool) error {
	query := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        login TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		balance NUMERIC (10, 2) DEFAULT 0,
		withdrawn NUMERIC (10, 2) DEFAULT 0
    );
	CREATE UNIQUE INDEX IF NOT EXISTS idx_login ON users (login);
    `
	_, err := db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("unable to create table: %w", err)
	}
	return nil
}

func CreateOrdersTable(ctx context.Context, db *pgxpool.Pool) error {
	query := `
    CREATE TABLE IF NOT EXISTS orders (
    	id SERIAL PRIMARY KEY,
    	order_number TEXT UNIQUE NOT NULL,
    	user_id INT REFERENCES users(id) ON DELETE CASCADE,
    	status TEXT NOT NULL DEFAULT 'NEW',
		accrual NUMERIC(10, 2) DEFAULT NULL,
    	uploaded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("unable to create table: %w", err)
	}
	return nil
}

func CreateWithdrawalsTable(ctx context.Context, db *pgxpool.Pool) error {
	query := `
    CREATE TABLE IF NOT EXISTS withdrawals (
		id SERIAL PRIMARY KEY,
		user_id INT REFERENCES users(id) ON DELETE CASCADE,
		order_number TEXT NOT NULL,
		amount NUMERIC(10, 2) NOT NULL,
		withdrawn_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("unable to create table: %w", err)
	}
	return nil
}

var ErrorDuplicate = errors.New("duplicate entry: URL already exists")
var ErrorInsufficientFunds = errors.New("insufficient funds")

func CreateUser(ctx context.Context, db *pgxpool.Pool, user *models.User) (int, error) {
	query := `INSERT INTO users (login, password) 
			  VALUES ($1, $2)
			  ON CONFLICT (login) DO NOTHING
			  RETURNING id
			  `

	var userID int
	err := db.QueryRow(ctx, query, user.Login, user.Password).Scan(&userID)

	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, ErrorDuplicate
		}
		return 0, err
	}

	return userID, nil
}

func GetUserByLogin(ctx context.Context, db *pgxpool.Pool, login string) (*models.User, error) {
	var user models.User
	query := `SELECT id, login, password FROM USERS WHERE login=$1`

	err := db.QueryRow(ctx, query, login).Scan(&user.ID, &user.Login, &user.Password)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func AddOrder(ctx context.Context, db *pgxpool.Pool, userID int, orderNumber string) error {
	query := `INSERT INTO orders (order_number, user_id, status)
			  VALUES ($1, $2, 'NEW')`

	_, err := db.Exec(ctx, query, orderNumber, userID)
	return err
}

func OrderExists(ctx context.Context, db *pgxpool.Pool, orderNumber string) (bool, int, error) {
	query := `SELECT user_id FROM orders WHERE order_number = $1`

	var userID int
	err := db.QueryRow(ctx, query, orderNumber).Scan(&userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, 0, nil
		}
		return false, 0, err
	}
	return true, userID, nil
}

func GetOrdersByUserID(ctx context.Context, db *pgxpool.Pool, userID int) ([]models.Order, error) {
	query := `SELECT order_number, status, accrual, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC`
	rows, err := db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.OrderNumber, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func GetUserBalance(ctx context.Context, db *pgxpool.Pool, userID int) (*models.Balance, error) {
	query := `SELECT balance, withdrawn FROM users WHERE id = $1`

	var balance models.Balance
	err := db.QueryRow(ctx, query, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		return nil, err
	}
	return &balance, nil
}

func WithdrawBalance(ctx context.Context, db *pgxpool.Pool, userID int, amount float32, orderNumber string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var currentBalance float32
	queryBalance := `SELECT balance FROM users WHERE id = $1`
	err = tx.QueryRow(ctx, queryBalance, userID).Scan(&currentBalance)
	if err != nil {
		return err
	}

	if currentBalance < amount {
		return ErrorInsufficientFunds
	}

	queryUpdBalance := `UPDATE users 
						SET balance = balance - $1, withdrawn = withdrawn + $1
						WHERE id = $2`
	_, err = tx.Exec(ctx, queryUpdBalance, amount, userID)
	if err != nil {
		return err
	}

	queryInsWithdraw := `INSERT INTO withdrawals (user_id, order_number, amount) 
						 VALUES ($1, $2, $3)`
	_, err = tx.Exec(ctx, queryInsWithdraw, userID, orderNumber, amount)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func GetUserWithdrawals(ctx context.Context, db *pgxpool.Pool, userID int) ([]models.Withdrawal, error) {
	query := `
        SELECT order_number, amount, withdrawn_at 
        FROM withdrawals 
        WHERE user_id = $1 
        ORDER BY withdrawn_at DESC
    `
	rows, err := db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []models.Withdrawal
	for rows.Next() {
		var withdrawal models.Withdrawal
		err := rows.Scan(&withdrawal.OrderNumber, &withdrawal.Sum, &withdrawal.ProcessedAt)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	return withdrawals, nil
}

func GetPendingOrders(ctx context.Context, db *pgxpool.Pool) ([]models.Order, error) {
	query := `
        SELECT user_id, order_number, status 
        FROM orders 
        WHERE status = 'NEW' OR status = 'PROCESSING'`

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.UserID, &order.OrderNumber, &order.Status)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func UpdateOrder(ctx context.Context, db *pgxpool.Pool, orderNumber, status string, accrual float32, userID int) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	log.Printf("Updating order %s with status: %s, accrual: %.2f", orderNumber, status, accrual)
	queryOrders := `UPDATE orders
					SET status = $1, accrual = $2
					WHERE order_number = $3`

	_, err = tx.Exec(ctx, queryOrders, status, accrual, orderNumber)
	if err != nil {
		return fmt.Errorf("failed to update orders: %w", err)
	}

	if status == "PROCESSED" && accrual > 0 {
		log.Printf("Updating balance for user %d with accrual: %.2f", userID, accrual)
		queryUpdBalance := `UPDATE users 
							SET balance = balance + $1
							WHERE id = $2`
		_, err = tx.Exec(ctx, queryUpdBalance, accrual, userID)
		if err != nil {
			return fmt.Errorf("failed to update user balance: %w", err)
		}
	}

	return tx.Commit(ctx)

}
