package clientsmgr

import (
	"github.com/BurntSushi/toml"
	"github.com/sethvargo/go-password/password"

	"log"
	"os"
	"io/ioutil"
	"bytes"
)

var (
	// WorkingDir is the current working directory of the project.
	WorkingDir string
	// ConfigPath is the configuration file name.
	ConfigPath = "clientcfg.toml"
	// Config is the configuration loaded from file.
	Config Configuration
)

func init() {
	var err error
	WorkingDir, err = os.Getwd()
	if err != nil {
		log.Fatal("Cannot get working directory!", err)
	}
}

// Configuration represents a config file format for client management
// (server) part of the application.
type Configuration struct {
	WebPort string
	APIKey string
	Clients []ClientConfig
}

type ClientConfig struct {
	IP, Port string
	ClientName string
}

func newConfiguration() Configuration {
	res, err := password.Generate(64, 10, 0, false, true)
	if err != nil {
		panic(err)
	}
	return Configuration {
		WebPort: "8080",
		APIKey: res,
		Clients: []ClientConfig{
			{
				IP: "10.0.0.25",
				Port: "6457",
				ClientName: "Camera A",
			},
			{
				IP: "10.0.0.26",
				Port: "6457",
				ClientName: "Camera B",
			},
		},
	}
}


// LoadConfig loads the configuration file from disk. It will also generate one
// if it doesn't exist.
func LoadConfig() {
	var err error
	if _, err = toml.DecodeFile(WorkingDir+"/"+ConfigPath, &Config); err != nil {
		log.Printf("Cannot load config file. Error: %s", err)
		if os.IsNotExist(err) {
			log.Println("Generating new configuration file, as it doesn't exist")
			log.Println("Please edit the configuration file and then re-run the program")

			buf := new(bytes.Buffer)
			if err = toml.NewEncoder(buf).Encode(newConfiguration()); err != nil {
				log.Fatal(err)
			}

			err = ioutil.WriteFile(ConfigPath, buf.Bytes(), 0600)
			if err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		}
	}
}
