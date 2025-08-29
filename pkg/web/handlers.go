// Package web provides HTTP handlers and REST API endpoints for workflow management.
package web

import (
	"net/http"
	"time"

	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/services"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type APIHandlers struct {
	workflowService *services.Workflow
	validator       *validator.Validate
	registry        *registry.Registry
}

func NewAPIHandlers(
	workflowService *services.Workflow,
	validator *validator.Validate,
	registry *registry.Registry,
) *APIHandlers {
	return &APIHandlers{
		workflowService: workflowService,
		validator:       validator,
		registry:        registry,
	}
}

func (h *APIHandlers) GetWorkflows(c fiber.Ctx) error {
	workflows, err := h.workflowService.FetchAll(c.Context())
	if err != nil {
		return internalError(c, err)
	}

	return c.JSON(workflows)
}

func (h *APIHandlers) GetWorkflow(c fiber.Ctx) error {
	id := c.Params("id")

	if id == "" {
		return badRequest(c, "Workflow ID is required")
	}

	workflow, err := h.workflowService.FetchByID(c.Context(), id)
	if err != nil {
		if err.Error() == "workflow not found" {
			return notFound(c, "Workflow not found")
		}

		return internalError(c, err)
	}

	return c.JSON(workflow)
}

func (h *APIHandlers) HealthCheck(c fiber.Ctx) error {
	registryCheck, regOk := h.registry.HealthCheck()
	repositoryCheck, repOk := h.workflowService.HealthCheck(c.Context())

	status := "unhealthy"
	message := "Operion API is unhealthy"
	httpStatus := http.StatusInternalServerError

	if regOk && repOk {
		status = "healthy"
		message = "Operion API is healthy"
		httpStatus = http.StatusOK
	}

	return c.Status(httpStatus).JSON(fiber.Map{
		"status":  status,
		"message": message,
		"checkers": fiber.Map{
			"registry":   registryCheck,
			"repository": repositoryCheck,
		},
		"timestamp": time.Now().UTC(),
	})
}
