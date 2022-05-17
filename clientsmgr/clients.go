package clientsmgr

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type SystemState int64

const (
	SystemStandby SystemState = iota
	SystemRecording
	SystemStitching
)

type ClientState int64

const (
	// Disconnected means we never connected to the client.
	Disconnected ClientState = iota
	// ConnectionFailed means connection to the client was
	// unsuccessful, such as client is down or API key is invalid.
	ConnectionFailed
	// ConnectedStandby means the client is standby and ready to
	// start recording.
	ConnectedStandby
	// ConnectedRecording means the client is currently recording.
	ConnectedRecording
	// ConnectedFailed means the connection to the client is
	// successful, but there was issue with starting recording,
	// saving the video, etc.
	ConnectedFailed
)

var LoadedClients []Client
var CurrentSystemState SystemState

// Client represents a client state, and contains a copy of the client
// configuration that is loaded.
type Client struct {
	Config ClientConfig
	State  ClientState
	// StateErr contains an error message in case the state is C
	StateErr error
	// This is the video UUID that is sent by the client when starting
	// or ending the video.
	VideoUUID string
}

func (c *Client) GetURIWithKey() string {
	return fmt.Sprintf("https://%s:%s/client/%s", c.Config.IP,
		c.Config.Port, Config.APIKey)
}

// Connect attempts to load all clients defined in the configuration
// file, and populates
func Connect() {
	wg := new(sync.WaitGroup)
	wg.Add(len(LoadedClients))
	for i, c := range LoadedClients {
		go func(i int, c Client) {
			defer wg.Done()
			client := http.Client{
				Timeout: 1 * time.Second,
			}
			resp, err := client.Get(c.GetURIWithKey() + "/state")
			if err != nil {
				LoadedClients[i].State = ConnectionFailed
				LoadedClients[i].StateErr = fmt.Errorf("cannot connect: %w", err)
				return
			}
			switch resp.StatusCode {
			case http.StatusOK:
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					LoadedClients[i].State = ConnectionFailed
					LoadedClients[i].StateErr = fmt.Errorf("cannot read status body: %w", err)

					return
				}
				switch strings.TrimSpace(string(body)) {
				case "standby":
					LoadedClients[i].State = ConnectedStandby
					LoadedClients[i].StateErr = nil
				case "recording":
					LoadedClients[i].State = ConnectedRecording
					LoadedClients[i].StateErr = nil
				}

			case http.StatusForbidden:
				LoadedClients[i].State = ConnectionFailed
				LoadedClients[i].StateErr = errors.New("return status forbidden, is API key correct?")
			}
		}(i, c)
	}
	wg.Wait()
}

func SetupCron() {
	c := cron.New()
	c.AddFunc("@every 5s", Connect)
	c.Start()
}

func StartRecording() error {
	if CurrentSystemState != SystemStandby {
		return errors.New("System not standby, cannot start!")
	}
	CurrentSystemState = SystemRecording

	// Get time 3s in the future
	n := time.Now().Add(3 * time.Second)

	wg := new(sync.WaitGroup)
	wg.Add(len(LoadedClients))

	// Send that time to all clients
	for i, c := range LoadedClients {
		go func(i int, c Client) {
			defer wg.Done()
			client := http.Client{
				Timeout: 1 * time.Second,
			}
			resp, err := client.Get(c.GetURIWithKey() + "/startAt/" +
				string(n.UnixNano()))
			if err != nil {
				LoadedClients[i].State = ConnectionFailed
				LoadedClients[i].StateErr = fmt.Errorf("cannot connect: %w", err)
				return
			}

			switch resp.StatusCode {
			case http.StatusOK:
				LoadedClients[i].State = ConnectedRecording
				LoadedClients[i].StateErr = nil
			case http.StatusBadRequest:
				LoadedClients[i].State = ConnectionFailed
				LoadedClients[i].StateErr = fmt.Errorf("bad request: %w", err)
			case http.StatusForbidden:
				LoadedClients[i].State = ConnectionFailed
				LoadedClients[i].StateErr = errors.New("return status forbidden, is API key correct?")
				return

			}
		}(i, c)
	}
	wg.Wait()
	return nil
}
