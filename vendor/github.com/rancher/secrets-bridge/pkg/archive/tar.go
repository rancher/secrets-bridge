package archive

import (
	"archive/tar"
	"bytes"
)

type ArchiveFile struct {
	Name    string
	Content string
}

func CreateTarArchive(files []ArchiveFile) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	tw := tar.NewWriter(buffer)

	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Mode: 0600,
			Size: int64(len(file.Content)),
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return buffer, err
		}
		if _, err := tw.Write([]byte(file.Content)); err != nil {
			return buffer, err
		}
	}

	if err := tw.Close(); err != nil {
		return buffer, err
	}
	return buffer, nil
}
