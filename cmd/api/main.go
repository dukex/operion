package main

import (
	"log"
	"os"

	"github.com/dukex/operion/pkg/persistence/file"
	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/web"
	"github.com/dukex/operion/pkg/workflow"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

var validate *validator.Validate

func main() {
	persistence := file.NewFilePersistence("./examples/data")
	workflowRepository := workflow.NewRepository(persistence)

	registry.RegisterAllComponents()
	registry := registry.DefaultRegistry

	validate = validator.New(validator.WithRequiredStructEnabled())

	handlers := web.NewAPIHandlers(workflowRepository, validate, registry)

	port, found := os.LookupEnv("PORT")
	if !found {
		port = "3000"
	}

	app := fiber.New()
	app.Use(cors.New())
	app.Use(logger.New(logger.Config{
		DisableColors: true,
	}))
	app.Use(healthcheck.New())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Operion Workflow Automation API")
	})

	w := app.Group("/workflows")
	w.Get("/", handlers.GetWorkflows)
	// w.Post("/", handlers.CreateWorkflow)
	w.Get("/:id", handlers.GetWorkflow)
	// w.Patch("/:id", handlers.PatchWorkflow)
	// w.Delete("/:id", handlers.DeleteWorkflow)
	// w.Patch("/:id/steps", handlers.PatchWorkflowSteps)
	// w.Patch("/:id/triggers", handlers.PatchWorkflowTriggers)

	// registry := app.Group("/registry")
	// registry.Get("/actions", handlers.GetAvailableActions)
	// registry.Get("/triggers", handlers.GetAvailableTriggers)

	log.Fatal(app.Listen(":" + port))
}
