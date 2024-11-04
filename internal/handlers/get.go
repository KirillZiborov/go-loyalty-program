package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/KirillZiborov/go-loyalty-program/internal/auth"
	"github.com/KirillZiborov/go-loyalty-program/internal/database"
	"github.com/KirillZiborov/go-loyalty-program/internal/logging"
	"github.com/KirillZiborov/go-loyalty-program/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

func GetOrders(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID, err := auth.AuthGet(r)
		if err != nil || userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		orders, err := database.GetOrdersByUserID(r.Context(), db, userID)
		if err != nil {
			logging.Sugar.Errorw("Error fetching orders:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if len(orders) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		sort.Slice(orders, func(i, j int) bool {
			return orders[i].UploadedAt.After(orders[j].UploadedAt)
		})

		var response []models.OrderResponse
		for _, order := range orders {
			resp := models.OrderResponse{
				OrderNumber: order.OrderNumber,
				Status:      order.Status,
				UploadedAt:  order.UploadedAt.Format(time.RFC3339),
			}

			if order.Status == "PROCESSED" && order.Accrual != nil {
				resp.Accrual = *order.Accrual
			}
			response = append(response, resp)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

func GetBalance(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID, err := auth.AuthGet(r)
		if err != nil || userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		balance, err := database.GetUserBalance(r.Context(), db, userID)
		if err != nil {
			logging.Sugar.Errorw("Error fetching balance:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		response := models.BalanceResponse{
			Current:   balance.Current,
			Withdrawn: balance.Withdrawn,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

func GetWithdrawals(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID, err := auth.AuthGet(r)
		if err != nil || userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		withdrawals, err := database.GetUserWithdrawals(r.Context(), db, userID)
		if err != nil {
			logging.Sugar.Errorw("Error fetching withdrawals:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if len(withdrawals) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		sort.Slice(withdrawals, func(i, j int) bool {
			return withdrawals[i].ProcessedAt.After(withdrawals[j].ProcessedAt)
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(withdrawals)
	}
}
