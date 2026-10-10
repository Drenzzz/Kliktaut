.PHONY: run migrate migrate-down migrate-force

# DSN golang-migrate (format URL, bukan DSN GORM)
MIGRATE_DSN = postgres://$${DB_USER:-postgres}:$${DB_PASSWORD:-your_password}@$${DB_HOST:-localhost}:$${DB_PORT:-5433}/$${DB_NAME:-kliktaut}?sslmode=$${DB_SSLMODE:-disable}
# Build tag "postgres" wajib: driver CLI golang-migrate dikontrol build tags
MIGRATE = go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate

run:
	go run cmd/api/main.go

migrate:
	$(MIGRATE) -source file://migrations -database "$(MIGRATE_DSN)" up

migrate-down:
	$(MIGRATE) -source file://migrations -database "$(MIGRATE_DSN)" down 1

# Pakai hanya saat migrasi gagal di tengah jalan:
# make migrate-force VERSION=<n> (sinkronkan state tanpa menjalankan SQL)
migrate-force:
	$(MIGRATE) -source file://migrations -database "$(MIGRATE_DSN)" force $(VERSION)
