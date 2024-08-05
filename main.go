package main

import (
	"errors"
	"github.com/Aquilabot/KreaPC-API/pkg/scraper"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"log"
)

type SearchRequest struct {
	Query  string `json:"query"`
	Region string `json:"region"`
}

type GetPartRequest struct {
	URL string `json:"url"`
}

func main() {
	// Initialize the scraper
	scrap := scraper.NewScraper()
	scrap.RandomizeUserAgent()

	// Create a Fiber app
	app := fiber.New()
	app.Use(helmet.New())
	app.Use(logger.New(logger.Config{
		Format: "${pid} | ${time} | ${latency} | [${ip}]:${port} | ${status} - ${method} ${path}\n",
	}))

	// Endpoint for searching PC parts
	app.Post("/search", func(c *fiber.Ctx) error {
		var req SearchRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request payload"})
		}

		searchResults, err := scrap.SearchPCParts(req.Query, req.Region)
		if err != nil {
			var redirectError *scraper.RedirectError
			if errors.As(err, &redirectError) {
				// Handle redirect to a single product page
				part, err := scrap.GetPart(redirectError.Error())
				if err != nil {
					return c.Status(500).JSON(fiber.Map{"error": "Error fetching product details"})
				}
				return c.JSON(part)
			}
			return c.Status(500).JSON(fiber.Map{"error": "Error searching parts"})
		}

		return c.JSON(searchResults)
	})

	// Endpoint for getting details of a single part
	app.Post("/getPart", func(c *fiber.Ctx) error {
		var req GetPartRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request payload"})
		}

		part, err := scrap.GetPart(req.URL)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Error fetching part"})
		}
		return c.JSON(part)
	})

	// Start the server
	log.Fatal(app.Listen(":3000"))
}
