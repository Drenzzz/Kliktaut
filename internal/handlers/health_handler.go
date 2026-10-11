package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthHandler struct {
	DB    *gorm.DB
	Redis *redis.Client
	Start time.Time
}

func NewHealthHandler(db *gorm.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{DB: db, Redis: rdb, Start: time.Now()}
}

// GET /health — public; performs real pg & redis checks on every request
func (h *HealthHandler) Get(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "ok"
	if err := h.DB.WithContext(ctx).Raw("SELECT 1").Scan(&[]int{}).Error; err != nil {
		dbStatus = "error"
	}

	redisStatus := "ok"
	if err := h.Redis.Ping(ctx).Err(); err != nil {
		redisStatus = "error"
	}

	status := fiber.StatusOK
	if dbStatus != "ok" || redisStatus != "ok" {
		status = fiber.StatusServiceUnavailable
	}

	return c.Status(status).JSON(fiber.Map{
		"status":   map[bool]string{true: "healthy", false: "degraded"}[status == fiber.StatusOK],
		"database": dbStatus,
		"redis":    redisStatus,
		"uptime":   time.Since(h.Start).Round(time.Second).String(),
	})
}
