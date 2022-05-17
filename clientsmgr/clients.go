package clientsmgr

import (
	"net/http"
	"errors"
	"fmt"
	"strings"
	"io/ioutil"
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

// Client represents a client state, and contains a copy of the client
// configuration that is loaded.
type Client struct {
	Config ClientConfig
	State ClientState
	// StateErr contains an error message in case the state is C
	StateErr error
}

// Connect attempts to load all clients defined in the configuration
// file, and populates 
func Connect() {
	for _, c := range LoadedClients {
		uri:= c.Config.IP+":"+c.Config.Port
		resp, err := http.Get(uri + "/client/"+Config.APIKey+"/state")
		if err != nil {
			continue
		}
		switch resp.StatusCode {
		case http.StatusOK:
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil{
				c.State = ConnectionFailed
				c.StateErr = fmt.Errorf("Cannot read status body: %w", err)
				continue
			}
			switch strings.TrimSpace(string(body)) {
			case "standby":
				c.State = ConnectedStandby
				c.StateErr = nil
			case "recording":
				c.State = ConnectedRecording
				c.StateErr = nil
			}
			
		case http.StatusForbidden:
			c.State = ConnectionFailed
			c.StateErr = errors.New("return status forbidden, is API key correct?")
		}
	}
}
