package utils

import (
	"github.com/gofiber/fiber/v3"
)

// Standard response format §14.2–14.3 dari dokumen perencanaan.

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *MetaInfo   `json:"meta,omitempty"`
}

type MetaInfo struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}

type ErrorResponse struct {
	Success bool      `json:"success"`
	Error   ErrorBody `json:"error"`
}

func Success(c fiber.Ctx, status int, data interface{}) error {
	return c.Status(status).JSON(SuccessResponse{Success: true, Data: data})
}

func SuccessWithMeta(c fiber.Ctx, status int, data interface{}, meta MetaInfo) error {
	return c.Status(status).JSON(SuccessResponse{Success: true, Data: data, Meta: &meta})
}

func Error(c fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(ErrorResponse{
		Success: false,
		Error:   ErrorBody{Code: code, Message: message},
	})
}
