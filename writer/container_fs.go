package writer

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/rancher/secrets-bridge/pkg/archive"
	"golang.org/x/net/context"
)

type DockerContainerFSWriter struct {
	message      string
	path         string
	dockerClient *client.Client
	containerId  string
}

type SecretWriter interface {
	Write() error
}

func NewSecretWriter(config map[string]interface{}) (SecretWriter, error) {
	return NewDockerContainerFSWriter(config)
}

func NewDockerContainerFSWriter(opts map[string]interface{}) (*DockerContainerFSWriter, error) {
	return &DockerContainerFSWriter{
		message:      opts["message"].(string),
		path:         opts["path"].(string),
		dockerClient: opts["dockerClient"].(*client.Client),
		containerId:  opts["containerId"].(string),
	}, nil
}

func (d *DockerContainerFSWriter) Write() error {
	// this will log the temp token.. but with short TTLs
	// the debug value outweighs the risk.
	logrus.Debugf("Writing message: %#v", d.message)
	files := []archive.ArchiveFile{
		{"secrets.txt", d.message},
	}
	tarball, err := archive.CreateTarArchive(files)
	if err != nil {
		logrus.Error("Failed to create Tar file")
		return err
	}

	// verify the path
	_, err = d.dockerClient.ContainerStatPath(context.Background(), d.containerId, d.path)
	if err != nil {
		return err
	}

	opts := types.CopyToContainerOptions{}

	if err = d.dockerClient.CopyToContainer(context.Background(), d.containerId, d.path, tarball, opts); err != nil {
		return err
	}

	return nil
}
