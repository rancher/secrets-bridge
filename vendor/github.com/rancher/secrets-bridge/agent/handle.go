package agent

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types/events"
	"github.com/rancher/go-rancher-metadata/metadata"
	"github.com/rancher/secrets-bridge/writer"
)

type ContainerEventMessage struct {
	Event  *events.Message
	UUID   string `json:"UUID"`
	Action string `json:"Action"`
	Host   string `json:"Host"`
}

type VaultResponseThing struct {
	ExternalId string
	TempToken  string
	CubbyPath  string
}

type JsonHandler struct {
	metadataCli           *metadata.Client
	remoteVerificationUrl string
	agentUUID             string
	signingKey            string
}

type MessageHandler interface {
	Handle(*events.Message) error
}

func NewMessageHandler(opts map[string]interface{}) (MessageHandler, error) {
	handler := &JsonHandler{}

	mdUrl, ok := opts["metadata-url"]
	if !ok {
		return handler, errors.New("No metadataURL defined")
	}

	client, err := metadata.NewClientAndWait(mdUrl.(string))
	if err != nil {
		return handler, err
	}
	handler.metadataCli = client

	selfContainer, err := handler.metadataCli.GetSelfContainer()
	if err != nil {
		return handler, err
	}

	handler.agentUUID = selfContainer.UUID

	rsUrl, ok := opts["bridge-url"]
	if !ok || rsUrl.(string) == "" {
		return handler, errors.New("No bridge URL defined")
	}

	handler.signingKey = os.Getenv("CATTLE_SECRET_KEY")
	if handler.signingKey == "" {
		return handler, errors.New("No signing key available.")
	}

	handler.remoteVerificationUrl = rsUrl.(string)

	return handler, nil
}

func (j *JsonHandler) Handle(msg *events.Message) error {
	message, err := j.buildRequestMessage(msg)

	jMsg, err := json.Marshal(message)
	if err != nil {
		return err
	}

	b := bytes.NewBuffer(jMsg)

	resp, err := j.postRequestToSecretBridge(b)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var vaultThing VaultResponseThing
	decoder := json.NewDecoder(resp.Body)
	decoder.Decode(&vaultThing)

	err = writeResponse(&vaultThing)
	if err != nil {
		return err
	}

	return nil
}

func (j *JsonHandler) buildRequestMessage(msg *events.Message) (*ContainerEventMessage, error) {
	message := &ContainerEventMessage{}
	logrus.Infof("Received action: %s, from container: %s", msg.Action, msg.ID)

	message.Event = msg
	message.Action = msg.Action

	uuid, err := j.getUUIDFromMetadata(message.Event.Actor.Attributes["name"])
	if err != nil {
		return message, err
	}
	message.UUID = uuid

	message.Host, err = os.Hostname()
	if err != nil {
		return message, err
	}

	return message, nil
}

func (j *JsonHandler) generateSignatureHeader() string {
	mac := hmac.New(sha256.New, []byte(j.signingKey))

	logrus.Debugf("UUID: %s", j.agentUUID)

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	logrus.Debugf("Time: %s", ts)

	mac.Write([]byte(j.agentUUID + ts))
	hmacMessage := string(mac.Sum(nil)[:mac.Size()])
	logrus.Debugf("hmac: %x", hmacMessage)

	// UUID:TIMESTAMP:SIGNATURE
	message := strings.Join([]string{j.agentUUID, ts, hmacMessage}, ":")

	return base64.StdEncoding.EncodeToString([]byte(message))
}

func (j *JsonHandler) postRequestToSecretBridge(buffer *bytes.Buffer) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", j.remoteVerificationUrl, buffer)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-Agent-Signature", j.generateSignatureHeader())

	return client.Do(req)
}

func writeResponse(message *VaultResponseThing) error {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.22", nil, defaultHeaders)
	if err != nil {
		logrus.Fatal(err)
	}

	opts := map[string]interface{}{
		"dockerClient": cli,
		"message":      formatMessage(message),
		"path":         "/tmp",
		"containerId":  message.ExternalId,
	}

	writer, err := writer.NewSecretWriter(opts)
	if err != nil {
		return err
	}

	return writer.Write()
}

func formatMessage(message *VaultResponseThing) string {
	return fmt.Sprintf("export CUBBY_PATH=%s\nexport TEMP_TOKEN=%s\n", message.CubbyPath, message.TempToken)
}

func (j *JsonHandler) getUUIDFromMetadata(name string) (string, error) {
	var uuid string

	// I feel like this is going to be a problem some day.
	name = strings.Replace(name, "r-", "", 1)

	containers, err := j.metadataCli.GetContainers()
	if err != nil {
		logrus.Errorf("Failed to get containers: %s", err)
		return "", err
	}

	for _, container := range containers {
		if container.Name == name {
			uuid = container.UUID
			break
		}
	}

	if uuid == "" {
		return uuid, errors.New("No UUID found")
	}

	return uuid, nil
}
