package web

import (
	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/workflow"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type APIHandlers struct {
	repository *workflow.Repository
	validator  *validator.Validate
	registry   *registry.Registry
}

func NewAPIHandlers(
	repository *workflow.Repository,
	validator *validator.Validate,
	registry *registry.Registry,
) *APIHandlers {
	return &APIHandlers{
		repository: repository,
		validator:  validator,
		registry:   registry,
	}
}

// func (h *APIHandlers) prepareWorkflowSteps(steps []domain.WorkflowStep) []domain.WorkflowStep {
// 	prepared := make([]domain.WorkflowStep, len(steps))
// 	for i, step := range steps {
// 		prepared[i] = step
// 		// Auto-generate step ID if not provided or empty
// 		if prepared[i].ID == "" {
// 			prepared[i].ID = uuid.New().String()
// 		}
// 		// Auto-generate action ID if not provided or empty
// 		if prepared[i].Action.ID == "" {
// 			prepared[i].Action.ID = uuid.New().String()
// 		}
// 	}
// 	return prepared
// }

// func (h *APIHandlers) prepareWorkflowTriggers(triggers []domain.TriggerItem) []domain.TriggerItem {
// 	prepared := make([]domain.TriggerItem, len(triggers))
// 	for i, trigger := range triggers {
// 		prepared[i] = trigger
// 		// Auto-generate trigger ID if not provided or empty
// 		if prepared[i].ID == "" {
// 			prepared[i].ID = uuid.New().String()
// 		}
// 	}
// 	return prepared
// }

// func (h *APIHandlers) validateWorkflowSteps(steps []domain.WorkflowStep) error {
// 	availableActions := []string{"http_request", "transform", "file_write", "log"}
// 	stepNameRegex := regexp.MustCompile(`^[a-z0-9_]+$`)

// 	for _, step := range steps {
// 		// Validate step name format
// 		if !stepNameRegex.MatchString(step.Name) {
// 			return fmt.Errorf("invalid step name '%s'. Step names must be lowercase alphanumeric with underscores only (e.g., 'fetch_data', 'log_result')", step.Name)
// 		}

// 		// Validate action type
// 		actionType := step.Action.Type
// 		isValid := false
// 		for _, validType := range availableActions {
// 			if actionType == validType {
// 				isValid = true
// 				break
// 			}
// 		}
// 		if !isValid {
// 			return fmt.Errorf("invalid action type '%s' in step '%s'. Available types: %v", actionType, step.Name, availableActions)
// 		}
// 	}
// 	return nil
// }

func (h *APIHandlers) GetWorkflows(c fiber.Ctx) error {
	workflows, err := h.repository.FetchAll()
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

	workflow, err := h.repository.FetchByID(id)
	if err != nil {
		if err.Error() == "workflow not found" {
			return notFound(c, "Workflow not found")
		}
		return internalError(c, err)
	}

	return c.JSON(workflow)
}

// func (h *APIHandlers) CreateWorkflow(c *fiber.Ctx) error {
// 	var workflow domain.Workflow
// 	if err := c.BodyParser(&workflow); err != nil {
// 		return badRequest(c, "Invalid JSON format")
// 	}

// 	// Auto-generate IDs for triggers
// 	if len(workflow.Triggers) > 0 {
// 		workflow.Triggers = h.prepareWorkflowTriggers(workflow.Triggers)
// 	}

// 	// Auto-generate IDs for steps and actions
// 	if len(workflow.Steps) > 0 {
// 		workflow.Steps = h.prepareWorkflowSteps(workflow.Steps)
// 		if err := h.validateWorkflowSteps(workflow.Steps); err != nil {
// 			return badRequest(c, err.Error())
// 		}
// 	}

// 	if err := h.validator.Struct(workflow); err != nil {
// 		return badRequestWithError(c, err)
// 	}

// 	createdWorkflow, err := h.repository.Create(&workflow)
// 	if err != nil {
// 		return internalError(c, err)
// 	}

// 	return c.Status(fiber.StatusCreated).JSON(createdWorkflow)
// }

// func (h *APIHandlers) PatchWorkflow(c *fiber.Ctx) error {
// 	id := c.Params("id")
// 	if id == "" {
// 		return badRequest(c, "Workflow ID is required")
// 	}

// 	existing, err := h.repository.FetchByID(id)
// 	if err != nil {
// 		if err.Error() == "workflow not found" {
// 			return notFound(c, "Workflow not found")
// 		}
// 		return internalError(c, err)
// 	}

// 	originalData, err := json.Marshal(existing)
// 	if err != nil {
// 		return internalError(c, err)
// 	}

// 	patchData := c.Body()
// 	if len(patchData) == 0 {
// 		return badRequest(c, "Request body is required")
// 	}

// 	patchedData, err := jsonpatch.MergePatch(originalData, patchData)
// 	if err != nil {
// 		return badRequest(c, "Invalid JSON merge patch: "+err.Error())
// 	}

// 	var patchedWorkflow domain.Workflow
// 	if err := json.Unmarshal(patchedData, &patchedWorkflow); err != nil {
// 		return badRequest(c, "Invalid workflow data after patch")
// 	}

// 	if len(patchedWorkflow.Triggers) > 0 {
// 		patchedWorkflow.Triggers = h.prepareWorkflowTriggers(patchedWorkflow.Triggers)
// 	}

// 	if len(patchedWorkflow.Steps) > 0 {
// 		patchedWorkflow.Steps = h.prepareWorkflowSteps(patchedWorkflow.Steps)
// 		if err := h.validateWorkflowSteps(patchedWorkflow.Steps); err != nil {
// 			return badRequest(c, err.Error())
// 		}
// 	}

// 	if err := h.validator.Struct(patchedWorkflow); err != nil {
// 		return badRequestWithError(c, err)
// 	}

// 	updatedWorkflow, err := h.repository.Update(id, &patchedWorkflow)
// 	if err != nil {
// 		return internalError(c, err)
// 	}

// 	return c.JSON(updatedWorkflow)
// }

// func (h *APIHandlers) DeleteWorkflow(c *fiber.Ctx) error {
// 	id := c.Params("id")
// 	if id == "" {
// 		return badRequest(c, "Workflow ID is required")
// 	}

// 	err := h.repository.Delete(id)
// 	if err != nil {
// 		if err.Error() == "workflow not found" {
// 			return notFound(c, "Workflow not found")
// 		}
// 		return internalError(c, err)
// 	}

// 	return c.SendStatus(fiber.StatusNoContent)
// }

// func (h *APIHandlers) PatchWorkflowSteps(c *fiber.Ctx) error {
// 	id := c.Params("id")
// 	if id == "" {
// 		return badRequest(c, "Workflow ID is required")
// 	}

// 	existing, err := h.repository.FetchByID(id)
// 	if err != nil {
// 		if err.Error() == "workflow not found" {
// 			return notFound(c, "Workflow not found")
// 		}
// 		return internalError(c, err)
// 	}

// 	patchData := c.Body()
// 	if len(patchData) == 0 {
// 		return badRequest(c, "Request body is required")
// 	}

// 	var newSteps []domain.WorkflowStep
// 	if err := json.Unmarshal(patchData, &newSteps); err != nil {
// 		return badRequest(c, "Invalid JSON format for steps array")
// 	}

// 	newSteps = h.prepareWorkflowSteps(newSteps)
// 	if err := h.validateWorkflowSteps(newSteps); err != nil {
// 		return badRequest(c, err.Error())
// 	}

// 	existing.Steps = newSteps
// 	updatedWorkflow, err := h.repository.Update(id, existing)
// 	if err != nil {
// 		return internalError(c, err)
// 	}

// 	return c.JSON(updatedWorkflow.Steps)
// }

// func (h *APIHandlers) PatchWorkflowTriggers(c *fiber.Ctx) error {
// 	id := c.Params("id")
// 	if id == "" {
// 		return badRequest(c, "Workflow ID is required")
// 	}

// 	existing, err := h.repository.FetchByID(id)
// 	if err != nil {
// 		if err.Error() == "workflow not found" {
// 			return notFound(c, "Workflow not found")
// 		}
// 		return internalError(c, err)
// 	}

// 	patchData := c.Body()
// 	if len(patchData) == 0 {
// 		return badRequest(c, "Request body is required")
// 	}

// 	var newTriggers []domain.TriggerItem
// 	if err := json.Unmarshal(patchData, &newTriggers); err != nil {
// 		return badRequest(c, "Invalid JSON format for triggers array")
// 	}

// 	newTriggers = h.prepareWorkflowTriggers(newTriggers)

// 	existing.Triggers = newTriggers
// 	updatedWorkflow, err := h.repository.Update(id, existing)
// 	if err != nil {
// 		return internalError(c, err)
// 	}

// 	return c.JSON(updatedWorkflow.Triggers)
// }

// type ActionInfo struct {
// 	Type         string                 `json:"type"`
// 	Name         string                 `json:"name"`
// 	Description  string                 `json:"description"`
// 	ConfigSchema map[string]interface{} `json:"config_schema"`
// }

// type TriggerInfo struct {
// 	Type         string                 `json:"type"`
// 	Name         string                 `json:"name"`
// 	Description  string                 `json:"description"`
// 	ConfigSchema map[string]interface{} `json:"config_schema"`
// }

// // convertSchemaToMap converts a JSONSchema to a map for backward compatibility
// func convertSchemaToMap(schema *domain.JSONSchema) map[string]interface{} {
// 	if schema == nil {
// 		return map[string]interface{}{}
// 	}

// 	result := make(map[string]interface{})

// 	// Convert properties
// 	for propName, prop := range schema.Properties {
// 		propMap := map[string]interface{}{
// 			"type":        prop.Type,
// 			"description": prop.Description,
// 		}

// 		if prop.Enum != nil && len(prop.Enum) > 0 {
// 			propMap["enum"] = prop.Enum
// 		}

// 		if prop.Default != nil {
// 			propMap["default"] = prop.Default
// 		}

// 		if prop.Pattern != "" {
// 			propMap["pattern"] = prop.Pattern
// 		}

// 		if prop.Format != "" {
// 			propMap["format"] = prop.Format
// 		}

// 		// Check if this property is required
// 		isRequired := false
// 		for _, req := range schema.Required {
// 			if req == propName {
// 				isRequired = true
// 				break
// 			}
// 		}
// 		propMap["required"] = isRequired

// 		result[propName] = propMap
// 	}

// 	return result
// }

// func (h *APIHandlers) GetAvailableActions(c *fiber.Ctx) error {
// 	components := h.actionRegistry.GetRegisteredComponents()

// 	// Convert to the expected format for backward compatibility
// 	actions := make([]ActionInfo, len(components))
// 	for i, component := range components {
// 		actions[i] = ActionInfo{
// 			Type:         component.Type,
// 			Name:         component.Name,
// 			Description:  component.Description,
// 			ConfigSchema: convertSchemaToMap(component.Schema),
// 		}
// 	}

// 	return c.JSON(actions)
// }

// func (h *APIHandlers) GetAvailableTriggers(c *fiber.Ctx) error {
// 	components := h.triggerRegistry.GetRegisteredComponents()

// 	// Convert to the expected format for backward compatibility
// 	triggers := make([]TriggerInfo, len(components))
// 	for i, component := range components {
// 		triggers[i] = TriggerInfo{
// 			Type:         component.Type,
// 			Name:         component.Name,
// 			Description:  component.Description,
// 			ConfigSchema: convertSchemaToMap(component.Schema),
// 		}
// 	}

// 	return c.JSON(triggers)
// }
