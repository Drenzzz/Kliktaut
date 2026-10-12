package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"

	"kliktaut-api/internal/config"
	"kliktaut-api/internal/database"
	"kliktaut-api/internal/handlers"
	"kliktaut-api/internal/middleware"
	"kliktaut-api/internal/repositories"
	"kliktaut-api/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}

	rdb, err := database.NewRedis(cfg)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}

	app := fiber.New(fiber.Config{
		AppName: cfg.AppName,
	})

	app.Use(logger.New(logger.Config{
		// redact X-API-Key so it never lands in logs (REQ-028)
		Format: "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${error}\n",
	}))

	// order matters: static routes first, /:shortCode added in Phase 2
	health := handlers.NewHealthHandler(db, rdb)
	app.Get("/health", health.Get)

	// /api/v1/* — API key middleware arrives in Phase 3
	linkRepo := repositories.NewLinkRepository(db)
	shortener := services.NewShortener(cfg, services.NewRandomSource(), linkRepo.ExistsShortCode)
	linkSvc := services.NewLinkService(cfg, linkRepo, shortener)
	clickRepo := repositories.NewClickRepository(db)
	linkHandler := handlers.NewLinkHandler(linkSvc)
	redirectHandler := handlers.NewRedirectHandler(linkSvc, clickRepo)
	api := app.Group("/api/v1", middleware.APIKey(cfg.APIKey), middleware.RateLimit(rdb, "api", cfg.RateLimitMax, cfg.RateLimitWindowSeconds))
	api.Post("/links", linkHandler.Create)
	api.Get("/links", linkHandler.List)
	api.Get("/links/:id", linkHandler.GetByID)
	api.Patch("/links/:id", linkHandler.Update)
	api.Delete("/links/:id", linkHandler.Delete)

	analyticsSvc := services.NewAnalyticsService(repositories.NewAnalyticsRepository(db), linkRepo)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsSvc)
	api.Get("/links/:id/analytics", analyticsHandler.LinkAnalytics)
	api.Get("/links/:id/clicks", analyticsHandler.ClicksList)
	api.Get("/stats", analyticsHandler.Overview)

	// MUST BE LAST: catch-all param route (REQ-006)
	app.Get("/:shortCode", middleware.RateLimit(rdb, "ip", cfg.RateLimitRedirectMax, cfg.RateLimitRedirectWin), redirectHandler.Redirect)

	go func() {
		log.Printf("%s (%s) listening on :%s", cfg.AppName, cfg.AppEnv, cfg.AppPort)
		if err := app.Listen(":" + cfg.AppPort); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Graceful shutdown: tunggu request in-flight selesai BARU tutup DB/Redis
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Println("shutdown signal received, shutting down...")
	if err := app.Shutdown(); err != nil {
		log.Printf("shutdown: %v", err)
	}

	sqlDB, _ := db.DB()
	if sqlDB != nil {
		_ = sqlDB.Close()
	}
	if rdb != nil {
		_ = rdb.Close()
	}
	log.Println("shutdown complete")
}
