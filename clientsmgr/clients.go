package clientsmgr

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"log"
	"strings"
	"sync"
	"time"
	"os"
	"io"

	"github.com/robfig/cron/v3"
)

// SystemState defines the current system state, which should be
// reflected on the clients.
type SystemState int64

const (
	// SystemStandby means that the system is not performing any
	// action, and is ready to start recording -- assuming clients are
	// online.
	SystemStandby SystemState = iota
	// SystemRecording means that the system assumes all clients are
	// currently recording.
	SystemRecording
	// SystemFetching means that the system is fetching the video
	// files from the clients.
	SystemFetching
	// SystemStitching means that the system is stitching all the
	// video files that were fetched from the clients.
	SystemStitching
)

// ClientState represents the state of a specific client.
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

// LoadedClients is a list of all Clients, which includes their state,
// config, etc.
var LoadedClients []Client
// CurrentSystemState is the state of the system. This should be
// mostly reflective of the LoadedClients state.
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

// GetURIWithKey returns the URI of a client with the API key from the
// configuration pre-filled.
func (c *Client) GetURIWithKey() string {
	return fmt.Sprintf("http://%s:%s/client/%s", c.Config.IP,
		c.Config.Port, Config.APIKey)
}

// Connect attempts to load all clients defined in the configuration
// file, and refreshes the state.
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
				log.Printf("%s: %s", LoadedClients[i].Config.ClientName, err)
				LoadedClients[i].StateErr = fmt.Errorf("error while connecting")
				return
			}
			switch resp.StatusCode {
			case http.StatusOK:
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					LoadedClients[i].State = ConnectionFailed
					log.Printf("%s: %s", LoadedClients[i].Config.ClientName, err)
					LoadedClients[i].StateErr = fmt.Errorf("cannot read status body")

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

// SetupCron sets up a timer that refreshes the clients states at a configured interval.
func SetupCron() error {
	c := cron.New()
	// TODO make this configurable
	_, err := c.AddFunc("@every 5s", Connect)
	if err != nil {
		return err
	}
	c.Start()
	return nil
}

// StartRecording instructs all clients to start recording at some
// configured time in the future, giving drones time to start
// recording on-sync.
func StartRecording() error {
	if CurrentSystemState != SystemStandby {
		return errors.New("System not standby, cannot start!")
	}
	CurrentSystemState = SystemRecording

	// Get time 3s in the future, this is when all cameras will start recording.
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
				fmt.Sprint(n.UnixNano()))

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
				log.Printf("%s: %s", LoadedClients[i].Config.ClientName, err)
				LoadedClients[i].StateErr = fmt.Errorf("bad request")
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

// StopRecording instructs all clients to stop recording to make the
// recordings available.
func StopRecording() error {
	if CurrentSystemState != SystemRecording {
		return errors.New("System is not recording, no recording to stop!")
	}
	CurrentSystemState = SystemFetching
	wg := new(sync.WaitGroup)
	wg.Add(len(LoadedClients))

	// Send that time to all clients
	for i, c := range LoadedClients {
		go func(i int, c Client) {
			defer wg.Done()
			client := http.Client{
				Timeout: 1 * time.Second,
			}
			resp, err := client.Get(c.GetURIWithKey() + "/stop")

			if err != nil {
				LoadedClients[i].State = ConnectionFailed
				log.Printf("%s: %s", LoadedClients[i].Config.ClientName, err)
				LoadedClients[i].StateErr = fmt.Errorf("cannot connect")
				return
			}

			switch resp.StatusCode {
			case http.StatusOK:
				LoadedClients[i].State = ConnectedStandby
				LoadedClients[i].StateErr = nil
				// TODO check error then stitch videos
				fetchVideo(&LoadedClients[i])
			case http.StatusBadRequest:
				LoadedClients[i].State = ConnectionFailed
				log.Printf("stop recording: %s: bad request: %s", LoadedClients[i].Config.ClientName, err)
				LoadedClients[i].StateErr = fmt.Errorf("bad request")
			case http.StatusForbidden:
				LoadedClients[i].State = ConnectionFailed
				log.Printf("stop recording: %s: forbidden: %s", LoadedClients[i].Config.ClientName, err)
				LoadedClients[i].StateErr = errors.New("return status forbidden, is API key correct?")
				return
			}
		}(i, c)
	}
	wg.Wait()

	return nil
}


// fetchVideo pulls the video of a specific client and downloads it
// to disk.
func fetchVideo(c *Client) error {
	client := http.Client{
		Timeout: 1 * time.Second,
	}
	file, err := os.Create(WorkingDir + "/downloads/" + c.VideoUUID)
	if err != nil {
		return fmt.Errorf("cannot create video file: %w", err)
	}

	defer file.Close()
	resp, err := client.Get(c.GetURIWithKey() + "/video/" +
		c.VideoUUID)
	if err != nil {
		return fmt.Errorf("cannot get video response: %w", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("cannot copy video file: %w", err)
	}

	return nil
}
