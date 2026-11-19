package volume

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/client"

	"getarcane.dev/arcane/v3/pkg/timeouts"
)

type volumeACFSBinary struct {
	path    string
	archive []byte
	size    int64
}

var volumeACFSCompanion = sync.OnceValues(func() (*volumeACFSBinary, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return loadVolumeACFSCompanion(filepath.Join(filepath.Dir(executable), "acfs"))
})

func loadVolumeACFSCompanion(filename string) (*volumeACFSBinary, error) {
	content, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read packaged ACFS companion: %w", err)
	}
	if len(content) == 0 {
		return nil, errors.New("packaged ACFS companion is empty")
	}
	directory := fmt.Sprintf(".arcane-acfs-%x", sha256.Sum256(content))
	staged := path.Join(directory, "payload")
	published := path.Join(directory, "acfs")
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	if headerErr := writer.WriteHeader(&tar.Header{Name: directory, Typeflag: tar.TypeDir, Mode: 0o755}); headerErr != nil {
		return nil, headerErr
	}
	if headerErr := writer.WriteHeader(&tar.Header{Name: staged, Typeflag: tar.TypeReg, Mode: 0o555, Size: int64(len(content))}); headerErr != nil {
		return nil, headerErr
	}
	if _, err = writer.Write(content); err != nil {
		return nil, err
	}
	// The executable name appears only after the entire payload has been copied.
	if headerErr := writer.WriteHeader(&tar.Header{Name: published, Typeflag: tar.TypeLink, Mode: 0o555, Linkname: staged}); headerErr != nil {
		return nil, headerErr
	}
	if closeErr := writer.Close(); closeErr != nil {
		return nil, closeErr
	}
	return &volumeACFSBinary{archive: archive.Bytes(), path: path.Join("/tmp", published), size: int64(len(content))}, nil
}

func (s *VolumeService) installVolumeACFSCompanion(ctx context.Context, dockerClient *client.Client, containerID string, binary *volumeACFSBinary) (string, error) {
	result := s.helperGroup.DoChan("acfs:"+containerID+":"+binary.path, func() (any, error) {
		installCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeouts.DefaultDockerAPI)
		defer cancel()
		stat, err := dockerClient.ContainerStatPath(installCtx, containerID, client.ContainerStatPathOptions{Path: binary.path})
		if err == nil {
			if !stat.Stat.Mode.IsRegular() || stat.Stat.Mode.Perm() != 0o555 || stat.Stat.Size != binary.size || stat.Stat.LinkTarget != "" {
				return nil, errors.New("packaged ACFS companion path has unexpected metadata")
			}
			return binary.path, nil
		}
		if !errdefs.IsNotFound(err) {
			return nil, fmt.Errorf("inspect packaged ACFS companion: %w", err)
		}
		_, err = dockerClient.CopyToContainer(installCtx, containerID, client.CopyToContainerOptions{
			DestinationPath: "/tmp",
			Content:         bytes.NewReader(binary.archive),
			CopyUIDGID:      true,
		})
		if err != nil {
			return nil, fmt.Errorf("install packaged ACFS companion: %w", err)
		}
		return binary.path, nil
	})
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case installed := <-result:
		if installed.Err != nil {
			return "", installed.Err
		}
		return installed.Val.(string), nil
	}
}

func (s *VolumeService) volumeACFSExecutable(ctx context.Context, containerID string) (string, error) {
	binary, err := volumeACFSCompanion()
	if err != nil {
		return "", err
	}
	if binary == nil {
		return "acfs", nil
	}
	dockerClient, err := s.dockerService.GetClient(ctx)
	if err != nil {
		return "", err
	}
	return s.installVolumeACFSCompanion(ctx, dockerClient, containerID, binary)
}
