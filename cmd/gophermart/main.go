package main

import (
	"context"
	"net/http"
	"os"
	"time"

	accrualclient "github.com/KirillZiborov/go-loyalty-program/internal/accrualClient"
	"github.com/KirillZiborov/go-loyalty-program/internal/config"
	"github.com/KirillZiborov/go-loyalty-program/internal/database"
	"github.com/KirillZiborov/go-loyalty-program/internal/gzip"
	"github.com/KirillZiborov/go-loyalty-program/internal/handlers"
	"github.com/KirillZiborov/go-loyalty-program/internal/logging"
	"github.com/go-chi/chi"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	db *pgxpool.Pool
)

func main() {

	err := logging.Initialize()
	if err != nil {
		logging.Sugar.Fatalw("Internal logging error", err)
	}

	cfg := config.NewConfig()

	if cfg.DBPath != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		db, err = pgxpool.New(ctx, cfg.DBPath)
		if err != nil {
			logging.Sugar.Fatalw("Unable to connect to database", "error", err)
			os.Exit(1)
		}

		err = database.CreateUsersTable(ctx, db)
		if err != nil {
			logging.Sugar.Fatalw("Failed to create users table", "error", err)
			os.Exit(1)
		}
		err = database.CreateOrdersTable(ctx, db)
		if err != nil {
			logging.Sugar.Fatalw("Failed to create orders table", "error", err)
			os.Exit(1)
		}
		err = database.CreateWithdrawalsTable(ctx, db)
		if err != nil {
			logging.Sugar.Fatalw("Failed to create withdrawals table", "error", err)
			os.Exit(1)
		}
		defer db.Close()
	} else {
		logging.Sugar.Fatalw("No database address")
	}

	accrualclient.StartAccrual(cfg, context.Background(), db)

	r := chi.NewRouter()

	r.Use(logging.LoggingMiddleware())

	r.Post("/api/user/register", gzip.Middleware(handlers.RegisterUser(db)))
	r.Post("/api/user/login", gzip.Middleware(handlers.LoginUser(db)))
	r.Post("/api/user/orders", gzip.Middleware(handlers.SubmitOrder(db)))
	r.Post("/api/user/balance/withdraw", gzip.Middleware(handlers.Withdraw(db)))

	r.Get("/api/user/orders", gzip.Middleware(handlers.GetOrders(db)))
	r.Get("/api/user/balance", gzip.Middleware(handlers.GetBalance(db)))
	r.Get("/api/user/withdrawals", gzip.Middleware(handlers.GetWithdrawals(db)))

	logging.Sugar.Infow(
		"Starting server at",
		"addr", cfg.Address,
	)

	err = http.ListenAndServe(cfg.Address, r)
	if err != nil {
		logging.Sugar.Fatalw(err.Error(), "event", "start server")
	}
}
