package cmd

import (
	"log"

	"github.com/tiiuae/flyeye/clientsmgr"
	"github.com/tiiuae/flyeye/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs the FlyEye server",
	Run:   Serve,
}

func Serve(cmd *cobra.Command, args []string) {
	clientsmgr.LoadConfig()
	clientsmgr.SetupCron()
	engine := html.New("./templates", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Static("/", "./static")
	app.Get("/", routes.HomepageHandler)
	app.Post("/", routes.PostHomepageHandler)

	log.Fatal(app.Listen(":3000"))
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
