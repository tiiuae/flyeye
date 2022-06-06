package webroutes

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/tiiuae/flyeye/clientsmgr"
)

func HomepageHandler(c *fiber.Ctx) error {
	return c.Render("index", fiber.Map{
		"Clients": clientsmgr.LoadedClients,
	})
}

func PostHomepageHandler(c *fiber.Ctx) error {
	fmt.Println(c.FormValue("action"))
	switch c.FormValue("action") {
	case "start": // start recording
		clientsmgr.StartRecording()
	case "refresh": // refresh clients info
		clientsmgr.Connect()
	}
	return c.Redirect("/")
}
