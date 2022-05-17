package routes

import (
	"github.com/gofiber/fiber/v2"
)

func HomepageHandler (c *fiber.Ctx) error {
	return c.Render("index", fiber.Map{})
}

func PostHomepageHandler(c *fiber.Ctx) error {
	return nil
}
