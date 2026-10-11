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
		// redaksi X-API-Key agar tidak masuk log (REQ-028)
		Format: "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${error}\n",
	}))

	// urutan Penting: route statis dulu, /:shortCode menyusul di Phase 2
	health := handlers.NewHealthHandler(db, rdb)
	app.Get("/health", health.Get)

	// /api/v1/* — API key middleware menyusul di Phase 3
	linkSvc := services.NewLinkService(cfg, repositories.NewLinkRepository(db), services.NewShortener(cfg, services.NewRandomSource(), repositories.NewLinkRepository(db).ExistsShortCode))
	linkHandler := handlers.NewLinkHandler(linkSvc)
	api := app.Group("/api/v1")
	api.Post("/links", linkHandler.Create)

	go func() {
		log.Printf("%s (%s) listen di :%s", cfg.AppName, cfg.AppEnv, cfg.AppPort)
		if err := app.Listen(":" + cfg.AppPort); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Graceful shutdown: tunggu request in-flight selesai BARU tutup DB/Redis
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Println("signal diterima, shutting down...")
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
	log.Println("shutdown selesai")
}
