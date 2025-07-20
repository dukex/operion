package web

import (
	"github.com/gofiber/fiber/v3"
	"github.com/moogar0880/problems"
)

func badRequest(c fiber.Ctx, detail string) error {
	problem := problems.NewStatusProblem(400).
		WithInstance(c.Path()).
		WithType("validation_error").
		WithDetail(detail)
	return c.Status(fiber.StatusBadRequest).JSON(problem)
}


func notFound(c fiber.Ctx, detail string) error {
	problem := problems.NewStatusProblem(404).
		WithInstance(c.Path()).
		WithType("not_found").
		WithDetail(detail)
	return c.Status(fiber.StatusNotFound).JSON(problem)
}

func internalError(c fiber.Ctx, err error) error {
	problem := problems.NewStatusProblem(500).
		WithInstance(c.Path()).
		WithType("internal_error").
		WithError(err)
	return c.Status(fiber.StatusInternalServerError).JSON(problem)
}
