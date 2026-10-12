package handlers

import (
	"github.com/gofiber/fiber/v3"

	"kliktaut-api/internal/docs"
)

// RegisterDocs serves docs/openapi.yaml and a Scalar UI page at /docs.
// Fiber v3.6 ships no docs middleware, so Scalar loads from its CDN
// (decision #6 of the plan / SC-010 fallback).
func RegisterDocs(app *fiber.App) error {
	spec, err := docs.Spec()
	if err != nil {
		return err
	}

	app.Get("/openapi.yaml", func(c fiber.Ctx) error {
		c.Set("Content-Type", "application/yaml")
		return c.Send(spec)
	})

	app.Get("/docs", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(`<!doctype html>
<html>
<head>
  <title>Kliktaut API Docs</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
  <script id="api-reference" data-url="/openapi.yaml"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`)
	})

	return nil
}
