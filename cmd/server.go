package cmd

import (
	"log"

	"github.com/tiiuae/flyeye/clientsmgr"
	"github.com/tiiuae/flyeye/webroutes"

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
	err := clientsmgr.SetupCron()
	if err != nil {
		log.Panicf("failed to setup cron: %s", err)
	}
	engine := html.New("./templates", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Static("/", "./static")
	app.Get("/", webroutes.HomepageHandler)
	app.Post("/", webroutes.PostHomepageHandler)

	log.Fatal(app.Listen(":3000"))
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
