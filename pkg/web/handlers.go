// Package web provides HTTP handlers and REST API endpoints for workflow management.
package web

import (
	"net/http"
	"strings"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/services"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)


type APIHandlers struct {
	workflowService   *services.Workflow
	publishingService *services.Publishing
	validator         *validator.Validate
	registry          *registry.Registry
}

func NewAPIHandlers(
	workflowService *services.Workflow,
	publishingService *services.Publishing,
	validator *validator.Validate,
	registry *registry.Registry,
) *APIHandlers {
	return &APIHandlers{
		workflowService:   workflowService,
		publishingService: publishingService,
		validator:         validator,
		registry:          registry,
	}
}

func (h *APIHandlers) GetWorkflows(c fiber.Ctx) error {
	ownerID := c.Query("owner_id")

	var workflows []*models.Workflow

	var err error

	if ownerID != "" {
		workflows, err = h.workflowService.FetchAllByOwner(c.Context(), ownerID)
	} else {
		workflows, err = h.workflowService.FetchAll(c.Context())
	}

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
		if persistence.IsWorkflowNotFound(err) {
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

func (h *APIHandlers) CreateWorkflow(c fiber.Ctx) error {
	var req CreateWorkflowRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid JSON format")
	}

	if err := h.validator.Struct(req); err != nil {
		return badRequest(c, err.Error())
	}

	workflow := &models.Workflow{
		Name:        req.Name,
		Description: req.Description,
		Variables:   req.Variables,
		Metadata:    req.Metadata,
		Owner:       req.Owner,
		Nodes:       []*models.WorkflowNode{}, // Empty - nodes added separately
		Connections: []*models.Connection{},   // Empty - connections added separately
	}

	created, err := h.workflowService.Create(c.Context(), workflow)
	if err != nil {
		return internalError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

func (h *APIHandlers) UpdateWorkflow(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return badRequest(c, "Workflow ID is required")
	}

	var req UpdateWorkflowRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid JSON format")
	}

	if err := h.validator.Struct(req); err != nil {
		return badRequest(c, err.Error())
	}

	// Get existing workflow and merge changes
	existing, err := h.workflowService.FetchByID(c.Context(), id)
	if err != nil {
		if persistence.IsWorkflowNotFound(err) {
			return notFound(c, "Workflow not found")
		}

		return internalError(c, err)
	}

	// Apply partial updates (nodes and connections managed separately)
	if req.Name != nil {
		existing.Name = *req.Name
	}

	if req.Description != nil {
		existing.Description = *req.Description
	}

	if req.Variables != nil {
		existing.Variables = req.Variables
	}

	if req.Metadata != nil {
		existing.Metadata = req.Metadata
	}

	updated, err := h.workflowService.Update(c.Context(), id, existing)
	if err != nil {
		return internalError(c, err)
	}

	return c.JSON(updated)
}

func (h *APIHandlers) DeleteWorkflow(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return badRequest(c, "Workflow ID is required")
	}

	err := h.workflowService.Delete(c.Context(), id)
	if err != nil {
		if persistence.IsWorkflowNotFound(err) {
			return notFound(c, "Workflow not found")
		}

		return internalError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *APIHandlers) PublishWorkflow(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return badRequest(c, "Workflow ID is required")
	}

	published, err := h.publishingService.PublishWorkflow(c.Context(), id)
	if err != nil {
		if persistence.IsWorkflowNotFound(err) {
			return notFound(c, "Workflow not found")
		}

		if strings.Contains(err.Error(), "validation failed") {
			return badRequest(c, err.Error())
		}

		return internalError(c, err)
	}

	return c.JSON(published)
}

func (h *APIHandlers) CreateDraftFromPublished(c fiber.Ctx) error {
	groupID := c.Params("groupId")
	if groupID == "" {
		return badRequest(c, "Workflow group ID is required")
	}

	draft, err := h.publishingService.CreateDraftFromPublished(c.Context(), groupID)
	if err != nil {
		if persistence.IsPublishedWorkflowNotFound(err) {
			return notFound(c, "Published workflow not found")
		}
		return internalError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(draft)
}
