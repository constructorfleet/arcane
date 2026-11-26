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

type volumeCompanion struct {
	path    string
	archive []byte
	size    int64
}

const volumeStatExecutable = "volume-stat"

var (
	volumeACFSCompanion = sync.OnceValues(func() (*volumeCompanion, error) {
		return loadPackagedVolumeCompanion("acfs")
	})
	volumeStatCompanion = sync.OnceValues(func() (*volumeCompanion, error) {
		return loadPackagedVolumeCompanion(volumeStatExecutable)
	})
)

func loadPackagedVolumeCompanion(name string) (*volumeCompanion, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return loadVolumeCompanion(filepath.Join(filepath.Dir(executable), name))
}

func loadVolumeCompanion(filename string) (*volumeCompanion, error) {
	name := filepath.Base(filename)
	content, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read packaged %s companion: %w", name, err)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("packaged %s companion is empty", name)
	}
	directory := fmt.Sprintf(".arcane-%s-%x", name, sha256.Sum256(content))
	staged := path.Join(directory, "payload")
	published := path.Join(directory, name)
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
	return &volumeCompanion{archive: archive.Bytes(), path: path.Join("/tmp", published), size: int64(len(content))}, nil
}

func (s *VolumeService) installVolumeCompanion(ctx context.Context, dockerClient *client.Client, containerID string, binary *volumeCompanion) (string, error) {
	result := s.helperGroup.DoChan("companion:"+containerID+":"+binary.path, func() (any, error) {
		installCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeouts.DefaultDockerAPI)
		defer cancel()
		stat, err := dockerClient.ContainerStatPath(installCtx, containerID, client.ContainerStatPathOptions{Path: binary.path})
		if err == nil {
			if !stat.Stat.Mode.IsRegular() || stat.Stat.Mode.Perm() != 0o555 || stat.Stat.Size != binary.size || stat.Stat.LinkTarget != "" {
				return nil, fmt.Errorf("packaged %s companion path has unexpected metadata", path.Base(binary.path))
			}
			return binary.path, nil
		}
		if !errdefs.IsNotFound(err) {
			return nil, fmt.Errorf("inspect packaged %s companion: %w", path.Base(binary.path), err)
		}
		_, err = dockerClient.CopyToContainer(installCtx, containerID, client.CopyToContainerOptions{
			DestinationPath: "/tmp",
			Content:         bytes.NewReader(binary.archive),
			CopyUIDGID:      true,
		})
		if err != nil {
			return nil, fmt.Errorf("install packaged %s companion: %w", path.Base(binary.path), err)
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
	return s.installVolumeCompanion(ctx, dockerClient, containerID, binary)
}
