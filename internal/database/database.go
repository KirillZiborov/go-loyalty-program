package database

import (
	"context"
	"errors"
	"fmt"

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
		balance NUMERIC (10, 2) DEFAULT 0;
		withdrawn NUMERIC (10, 2) DEFAULT 0;
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

var ErrorDuplicate = errors.New("duplicate entry: URL already exists")

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
