// Package web provides HTTP handlers and REST API endpoints for workflow management.
package web

import (
	"net/http"
	"strconv"
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
	// Parse query parameters
	req, err := h.parseListWorkflowsRequest(c)
	if err != nil {
		return badRequest(c, "Invalid query parameters: "+err.Error())
	}

	// Call service layer
	result, err := h.workflowService.ListWorkflows(c.Context(), *req)
	if err != nil {
		return handleServiceError(c, err)
	}

	// Return structured response with pagination metadata
	return c.JSON(fiber.Map{
		"workflows":     result.Workflows,
		"total_count":   result.TotalCount,
		"has_next_page": result.HasNextPage,
		"pagination": fiber.Map{
			"limit":  req.Limit,
			"offset": req.Offset,
		},
		"sorting": fiber.Map{
			"sort_by":    req.SortBy,
			"sort_order": req.SortOrder,
		},
	})
}

// parseListWorkflowsRequest parses and validates query parameters for listing workflows.
func (h *APIHandlers) parseListWorkflowsRequest(c fiber.Ctx) (*services.ListWorkflowsRequest, error) {
	req := &services.ListWorkflowsRequest{}

	// Parse pagination parameters
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return nil, err
		}

		req.Limit = limit
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return nil, err
		}

		req.Offset = offset
	}

	// Parse filtering parameters
	req.OwnerID = c.Query("owner_id")

	if statusStr := c.Query("status"); statusStr != "" {
		status := models.WorkflowStatus(statusStr)
		req.Status = &status
	}

	// Parse sorting parameters
	req.SortBy = c.Query("sort_by")
	req.SortOrder = c.Query("sort_order")

	// Parse data loading parameters
	if includeNodesStr := c.Query("include_nodes"); includeNodesStr != "" {
		includeNodes, err := strconv.ParseBool(includeNodesStr)
		if err != nil {
			return nil, err
		}

		req.IncludeNodes = includeNodes
	}

	if includeConnectionsStr := c.Query("include_connections"); includeConnectionsStr != "" {
		includeConnections, err := strconv.ParseBool(includeConnectionsStr)
		if err != nil {
			return nil, err
		}

		req.IncludeConnections = includeConnections
	}

	return req, nil
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
		return handleServiceError(c, err)
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
		return handleServiceError(c, err)
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
