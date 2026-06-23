package response

import "github.com/gofiber/fiber/v2"

type Envelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func OK(c *fiber.Ctx, data interface{}) error {
	return c.JSON(Envelope{Success: true, Data: data})
}

func OKMessage(c *fiber.Ctx, message string, data interface{}) error {
	return c.JSON(Envelope{Success: true, Message: message, Data: data})
}

func BadRequest(c *fiber.Ctx, err string) error {
	return c.Status(fiber.StatusBadRequest).JSON(Envelope{Success: false, Error: err})
}

func Unauthorized(c *fiber.Ctx, err string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(Envelope{Success: false, Error: err})
}

func InternalError(c *fiber.Ctx, err string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(Envelope{Success: false, Error: err})
}
