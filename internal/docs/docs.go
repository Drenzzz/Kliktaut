// Package docs embeds the OpenAPI specification.
package docs

import "embed"

//go:embed openapi.yaml
var OpenAPI embed.FS

// Spec returns the raw openapi.yaml bytes.
func Spec() ([]byte, error) {
	return OpenAPI.ReadFile("openapi.yaml")
}
