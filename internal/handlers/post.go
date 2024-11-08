package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/KirillZiborov/go-loyalty-program/internal/auth"
	"github.com/KirillZiborov/go-loyalty-program/internal/database"
	"github.com/KirillZiborov/go-loyalty-program/internal/logging"
	"github.com/KirillZiborov/go-loyalty-program/internal/models"
	"github.com/KirillZiborov/go-loyalty-program/internal/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func RegisterUser(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var user models.User

		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		if user.Login == "" || user.Password == "" {
			http.Error(w, "Login and password are required", http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			logging.Sugar.Errorw("Error hashing password", "error", err)
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}
		user.Password = string(hashedPassword)

		userID, err := database.CreateUser(r.Context(), db, &user)
		if err != nil {
			if err == database.ErrorDuplicate {
				http.Error(w, "User with this login already exists", http.StatusConflict)
			} else {
				logging.Sugar.Errorw("Error creating user", "error", err)
				http.Error(w, "Error creating user", http.StatusInternalServerError)
			}
			return
		}

		err = auth.AuthPost(w, r, userID)
		if err != nil {
			http.Error(w, "Error setting authentication cookie", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
	}
}

func LoginUser(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var user models.User

		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		if user.Login == "" || user.Password == "" {
			http.Error(w, "Login and password are required", http.StatusBadRequest)
			return
		}

		storedUser, err := database.GetUserByLogin(context.Background(), db, user.Login)
		if err != nil {
			logging.Sugar.Errorw("Error to find user", "error", err)
			http.Error(w, "Error to find user", http.StatusInternalServerError)
			return
		}
		if storedUser == nil {
			http.Error(w, "Invalid login or password", http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(storedUser.Password), []byte(user.Password))
		if err != nil {
			http.Error(w, "Invalid login or password", http.StatusUnauthorized)
			return
		}

		err = auth.AuthPost(w, r, storedUser.ID)
		if err != nil {
			http.Error(w, "Error getting token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	}
}

func SubmitOrder(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID, err := auth.AuthGet(r)
		if err != nil || userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		orderNumber := strings.TrimSpace(string(body))
		if !utils.CheckLuhn(orderNumber) {
			http.Error(w, "Invalid order number format", http.StatusUnprocessableEntity)
			return
		}

		ctx := r.Context()

		exists, ownerID, err := database.OrderExists(ctx, db, orderNumber)
		if err != nil {
			logging.Sugar.Errorw("Error to find order", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if exists {
			if ownerID == userID {
				http.Error(w, "Order already submitted by this user", http.StatusOK)
			} else {
				http.Error(w, "Order already submitted by another user", http.StatusConflict)
			}
			return
		}

		err = database.AddOrder(ctx, db, userID, orderNumber)
		if err != nil {
			logging.Sugar.Errorw("Error adding user", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func Withdraw(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID, err := auth.AuthGet(r)
		if err != nil || userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req models.WithdrawRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		if !utils.CheckLuhn(req.OrderNumber) {
			http.Error(w, "Invalid order number format", http.StatusUnprocessableEntity)
			return
		}

		if req.Sum <= 0 {
			http.Error(w, "Invalid amount", http.StatusBadRequest)
			return
		}

		err = database.WithdrawBalance(r.Context(), db, userID, req.Sum, req.OrderNumber)
		if err != nil {
			if err == database.ErrorInsufficientFunds {
				http.Error(w, "Insufficient funds", http.StatusPaymentRequired)
				return
			}

			logging.Sugar.Errorw("Error to withdraw", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
