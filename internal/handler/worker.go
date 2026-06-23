package handler

import (
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	"github.com/wisnuekas/mailtarget-sentinel/internal/worker"
	"github.com/wisnuekas/mailtarget-sentinel/pkg/response"
	"github.com/gofiber/fiber/v2"
)

type WorkerHandler struct {
	cfg      *config.Config
	detector *worker.Detector
}

func NewWorkerHandler(cfg *config.Config, detector *worker.Detector) *WorkerHandler {
	return &WorkerHandler{cfg: cfg, detector: detector}
}

func (h *WorkerHandler) Run(c *fiber.Ctx) error {
	if h.cfg.AdminToken != "" {
		auth := c.Get("Authorization")
		if auth != "Bearer "+h.cfg.AdminToken {
			return response.Unauthorized(c, "invalid admin token")
		}
	}

	go h.detector.Run(c.Context())

	return response.OKMessage(c, "detection worker triggered", fiber.Map{
		"status": "running",
	})
}
