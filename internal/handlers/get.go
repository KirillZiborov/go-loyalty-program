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

			if order.Status == "PROCESSED" {
				resp.Accrual = order.Accrual
			}
			response = append(response, resp)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
