package handler

import (
	"crypto/subtle"

	"github.com/gofiber/fiber/v2"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	"github.com/wisnuekas/mailtarget-sentinel/pkg/response"
)

type AuthHandler struct {
	cfg *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.Username == "" || req.Password == "" {
		return response.BadRequest(c, "username and password are required")
	}

	userOK := subtle.ConstantTimeCompare([]byte(req.Username), []byte(h.cfg.DashboardUsername)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.cfg.DashboardPassword)) == 1
	if !userOK || !passOK {
		return response.Unauthorized(c, "invalid username or password")
	}

	return response.OK(c, loginResponse{
		Username: req.Username,
		Token:    h.cfg.AdminToken,
	})
}
