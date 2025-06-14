package dto

type CreateWorkflowDTO struct {
	Name string `json:"name" validate:"required,min=3"`
	Description string `json:"description" validate:"required"`
}