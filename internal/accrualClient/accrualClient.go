package accrualclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/KirillZiborov/go-loyalty-program/internal/config"
	"github.com/KirillZiborov/go-loyalty-program/internal/database"
	"github.com/KirillZiborov/go-loyalty-program/internal/logging"
	"github.com/KirillZiborov/go-loyalty-program/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccrualResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual,omitempty"`
}

func StartAccrual(cfg *config.Config, ctx context.Context, db *pgxpool.Pool) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ProccessPendingOrders(cfg, ctx, db, 5)
		case <-ctx.Done():
			log.Println("Accrual process finished")
			return
		}

	}
}

func GetAccrual(cfg *config.Config, orderNumber string) (*AccrualResponse, error) {
	url := fmt.Sprintf("%s%s%s", cfg.SysAdress, "/api/orders/", orderNumber)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get request to acrual failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var accrual AccrualResponse
		if err := json.NewDecoder(resp.Body).Decode(&accrual); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &accrual, nil
	case http.StatusNoContent:
		return nil, nil
	case http.StatusTooManyRequests:
		retryAfter := resp.Header.Get("Retry-After")
		retryDuration, _ := time.ParseDuration(retryAfter + "s")
		time.Sleep(retryDuration)
		return GetAccrual(cfg, orderNumber)
	default:
		return nil, fmt.Errorf("accrual server error: %s", resp.Status)
	}
}

func ProccessPendingOrders(cfg *config.Config, ctx context.Context, db *pgxpool.Pool, workerCount int) {
	orders, err := database.GetPendingOrders(ctx, db)
	if err != nil {
		logging.Sugar.Errorw("error fetching orders", err)
		return
	}

	var wg sync.WaitGroup
	jobs := make(chan models.Order, len(orders))

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for order := range jobs {
				accrualResponse, err := GetAccrual(cfg, order.OrderNumber)
				if err != nil {
					log.Printf("Error getting order status %s: %v", order.OrderNumber, err)
					continue
				}

				log.Printf("Accrual response for order %s: %+v", order.OrderNumber, accrualResponse)

				var status string
				var accrual float32
				if accrualResponse != nil {
					status = accrualResponse.Status
					accrual = accrualResponse.Accrual
				}

				err = database.UpdateOrder(ctx, db, order.OrderNumber, status, accrual)
				if err != nil {
					log.Printf("Error updating order status %s: %v", order.OrderNumber, err)
					continue
				}
			}
		}()
	}

	for _, order := range orders {
		jobs <- order
	}

	close(jobs)
	wg.Wait()
}