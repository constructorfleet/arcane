package registry

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"

	"getarcane.dev/arcane/v3/internal/common"
	"getarcane.dev/types/v3"
)

// ScanImage uses the saved registry's scanner when the exact artifact is available.
func (s *ContainerRegistryService) ScanImage(ctx context.Context, options types.ImageRegistryScanOptions) (*types.RegistryScanReport, error) {
	if len(options.References) == 0 {
		return nil, nil
	}
	registries, err := s.GetAllRegistries(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(registries, func(a, b ContainerRegistry) int {
		return len(b.Namespace()) - len(a.Namespace())
	})
	for _, ref := range options.References {
		digest, parseErr := name.NewDigest(ref, name.WeakValidation)
		if parseErr != nil {
			continue
		}
		for _, registry := range registries {
			if !registry.Enabled || registry.Host() != digest.RegistryStr() || (registry.Namespace() != "" && !strings.HasPrefix(digest.RepositoryStr(), registry.Namespace()+"/")) {
				continue
			}
			report, scanErr := s.browse.ScanArtifact(ctx, types.RegistryArtifactScanOptions{
				RegistryID: registry.ID, Repository: digest.RepositoryStr(), Digest: digest.DigestStr(), Platform: options.Platform, Trigger: options.Trigger,
			})
			if errors.Is(scanErr, common.ErrNotFound) {
				continue
			}
			if scanErr != nil {
				return nil, scanErr
			}
			if report.Status == "Unsupported" || report.Status == "Platform unavailable" {
				continue
			}
			return &report, nil
		}
	}
	for _, registry := range registries {
		if !registry.Enabled {
			continue
		}
		report, proxyErr := s.browse.ProxyScan(ctx, registry.ID, options)
		if proxyErr != nil {
			return nil, proxyErr
		}
		if report != nil {
			return report, nil
		}
	}
	return nil, nil
}
