package agent

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/events"
	"github.com/docker/engine-api/types/filters"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
)

func StartAgent(c *cli.Context) {
	cli, err := getDockerClient()
	if err != nil {
		logrus.Fatalf("Could not get Docker client: %s", err)
	}

	filterArgs := filters.NewArgs()
	filterArgs.Add("event", "start")

	eventOptions := types.EventsOptions{
		Filters: filterArgs,
	}

	eventsResp, err := cli.Events(context.Background(), eventOptions)
	if err != nil {
		logrus.Fatal(err)
	}
	defer eventsResp.Close()

	bridgeUrl := strings.TrimSuffix(c.String("bridge-url"), "/")
	logrus.Debugf("Sending events to: %s", bridgeUrl)

	handler, err := NewMessageHandler(map[string]interface{}{
		"metadata-url": c.String("metadata-url"),
		"bridge-url":   bridgeUrl + "/v1/message",
	})
	if err != nil {
		logrus.Fatalf("Error: %s", err)
	}

	logrus.Info("Entering event listening Loop")
	d := json.NewDecoder(eventsResp)
	for {
		msg := &events.Message{}
		d.Decode(msg)

		// For now... will need to add some throttling at some point.
		go wrapHandler(handler.Handle, msg)
	}

	os.Exit(0)
}

func wrapHandler(handlerFunc func(*events.Message) error, msg *events.Message) {
	if err := handlerFunc(msg); err != nil {
		logrus.Debugf("Warning: %s", err)
	}
}

func getDockerClient() (*client.Client, error) {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	return client.NewClient("unix:///var/run/docker.sock", "v1.22", nil, defaultHeaders)
}
