package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	ipostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"

	"kliktaut-api/internal/cache"
	"kliktaut-api/internal/config"
	"kliktaut-api/internal/database"
	"kliktaut-api/internal/handlers"
	"kliktaut-api/internal/middleware"
	"kliktaut-api/internal/repositories"
	"kliktaut-api/internal/services"
	"kliktaut-api/internal/utils"
)

// MigrateUp runs pending migrations before the server listens. Used by the
// production container (distroless has no shell, decision #1) and safe to
// call when the database is already up to date.
func MigrateUp(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("migrate connection: %w", err)
	}
	defer func() { _ = db.Close() }()

	driver, err := ipostgres.WithInstance(db, &ipostgres.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file:///migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Production containers run migrations from the app binary itself
	// (distroless has no shell). Locally, `make migrate` does the same.
	if os.Getenv("RUN_MIGRATIONS") == "1" {
		if err := MigrateUp(cfg); err != nil {
			log.Fatalf("migrations: %v", err)
		}
		log.Println("migrations applied")
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
		// REQ-034: every error exits as the standard JSON envelope (§14.2),
		// never Fiber's plain-text default.
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			msg := "An unexpected server error occurred"
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
				switch code {
				case fiber.StatusNotFound:
					msg = "Route not found"
				case fiber.StatusMethodNotAllowed:
					msg = "Method not allowed"
				case fiber.StatusBadRequest:
					msg = "Bad request"
				}
			}
			return utils.Error(c, code, "HTTP_ERROR", msg)
		},
	})

	// REQ-029: restrict origins in production; allow all in development
	corsOrigins := []string{"*"}
	if cfg.AppEnv == "production" {
		corsOrigins = strings.Split(cfg.CORSOrigins, ",")
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: corsOrigins,
		AllowMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
	}))

	app.Use(logger.New(logger.Config{
		// redact X-API-Key so it never lands in logs (REQ-028)
		Format: "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${error}\n",
	}))

	// order matters: static routes first, /:shortCode added in Phase 2
	health := handlers.NewHealthHandler(db, rdb)
	app.Get("/health", health.Get)

	if err := handlers.RegisterDocs(app); err != nil {
		log.Fatalf("docs: %v", err)
	}

	// /api/v1/*: API key middleware arrives in Phase 3
	linkRepo := repositories.NewLinkRepository(db)
	shortener := services.NewShortener(cfg, services.NewRandomSource(), linkRepo.ExistsShortCode)
	linkCache := cache.NewLinkCache(rdb, cfg.CacheTTLSeconds)
	linkSvc := services.NewLinkService(cfg, linkRepo, shortener, linkCache)
	clickRepo := repositories.NewClickRepository(db)
	linkHandler := handlers.NewLinkHandler(linkSvc)
	redirectHandler := handlers.NewRedirectHandler(linkSvc, clickRepo, linkCache)
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
