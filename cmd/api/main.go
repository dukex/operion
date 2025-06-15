package main

import (
	"log"
	"os"

	"github.com/dukex/operion/internal/adapters/persistence/file"
	"github.com/dukex/operion/internal/admin/workflows"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/moogar0880/problems"
)

var validate *validator.Validate

func main() {
	persistence := file.NewFilePersistence("./data/workflows/index.json")

	workflowRepository := workflows.NewRepository(persistence)

	validate = validator.New(validator.WithRequiredStructEnabled())

	port, found := os.LookupEnv("PORT")
	if !found {
		port = "3000"
	}

	app := fiber.New()
	app.Use(cors.New())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hi!")
	})

	workflows := app.Group("/workflows")
	// workflows.Post("/", func(c *fiber.Ctx) error {
	// 	workflowDTO := &dto.CreateWorkflowDTO{}
	// 	c.BodyParser(workflowDTO)

	// 	err := validate.Struct(workflowDTO)
	// 	if err != nil {
	// 		problem := problems.NewStatusProblem(422).
	// 			WithInstance(c.Path()).
	// 			WithType("validation_error").
	// 			WithError(err)

	// 		return c.Status(fiber.StatusBadRequest).JSON(problem)
	// 	}

	// 	workflow, err := workflow.Create(workflowDTO)

	// 	if err != nil {
	// 		problem := problems.NewStatusProblem(500).
	// 			WithInstance(c.Path()).
	// 			WithType("internal_error").
	// 			WithError(err)
	// 		return c.Status(fiber.StatusInternalServerError).JSON(problem)
	// 	}

	// 	return c.Status(fiber.StatusCreated).JSON(workflow)
	// });

	workflows.Get("/", func(c *fiber.Ctx) error {
		responses, err := workflowRepository.FetchAll()

		if err != nil {
			problem := problems.NewStatusProblem(500).
				WithInstance(c.Path()).
				WithType("internal_error").
				WithError(err)
			return c.Status(fiber.StatusInternalServerError).JSON(problem)
		}

		return c.JSON(responses)
	})

	log.Fatal(app.Listen(":" + port))
}
