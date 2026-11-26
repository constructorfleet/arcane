package browse

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"

	"kit.getarcane.app/cache"

	"getarcane.dev/arcane/v3/internal/common"
	"getarcane.dev/types/v3"
)

func proxyHost(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	host := strings.ToLower(parsed.Host)
	switch host {
	case "hub.docker.com", "registry-1.docker.io", "index.docker.io":
		return "index.docker.io"
	case "docker.io":
		return "index.docker.io"
	default:
		return host
	}
}

// ProxyScan resolves an upstream digest through a configured Harbor proxy project.
func (s *Service) ProxyScan(ctx context.Context, id string, options types.ImageRegistryScanOptions) (*types.RegistryScanReport, error) {
	ctx, cancel := s.lookupContext(ctx)
	defer cancel()
	target, err := s.target(ctx, id)
	if err != nil {
		return nil, err
	}
	scheme, err := s.harborScheme(ctx, target)
	if err != nil {
		return nil, err
	}
	available, err := s.harborScanAvailable(ctx, target, scheme)
	if err != nil || !available {
		return nil, err
	}
	proxies, err := s.proxyProjects(ctx, target, scheme)
	if err != nil {
		return nil, err
	}
	for _, ref := range options.References {
		digest, parseErr := name.NewDigest(ref, name.WeakValidation)
		if parseErr != nil || digest.RegistryStr() == target.registry.RegistryStr() {
			continue
		}
		for _, proxy := range proxies {
			host, project, _ := strings.Cut(proxy, "/")
			if proxyHost("https://"+digest.RegistryStr()) != host {
				continue
			}
			repository := project + "/" + digest.RepositoryStr()
			repo, repoErr := target.repository(repository)
			if repoErr != nil {
				return nil, repoErr
			}
			report, scanErr := s.scanProxyDigest(ctx, target, repo, types.RegistryArtifactScanOptions{
				RegistryID: id, Repository: repository, Digest: digest.DigestStr(),
				Platform: options.Platform, Trigger: options.Trigger,
			})
			if scanErr != nil || report != nil {
				return report, scanErr
			}
		}
	}
	return nil, nil
}

func (s *Service) harborScanAvailable(ctx context.Context, target *target, scheme string) (bool, error) {
	provider, known, _ := s.providers.Get(target.cacheKey)
	if known && provider != "harbor" {
		return false, nil
	}
	if !known {
		body, probeErr := s.getHarborScanData(ctx, target, scheme, "/api/v2.0/systeminfo", nil)
		if errors.Is(probeErr, common.ErrNotFound) {
			s.providers.Set(cache.Entry[string, string]{Key: target.cacheKey, Value: "oci"})
			return false, nil
		}
		if probeErr != nil {
			return false, probeErr
		}
		var info struct {
			Version  string `json:"harbor_version"`
			AuthMode string `json:"auth_mode"`
		}
		if decodeErr := json.Unmarshal(body, &info); decodeErr != nil {
			return false, decodeErr
		}
		if info.Version == "" && info.AuthMode == "" {
			return false, nil
		}
		s.providers.Set(cache.Entry[string, string]{Key: target.cacheKey, Value: "harbor"})
	}
	return true, nil
}

func (s *Service) proxyProjects(ctx context.Context, target *target, scheme string) ([]string, error) {
	return s.cachedNames(target.cacheKey+"\x00scan-proxies", func() ([]string, error) {
		var proxies []string
		for page := 1; page <= maxDiscoveryPages; page++ {
			body, readErr := s.getHarborScanData(ctx, target, scheme, "/api/v2.0/projects", url.Values{"page": {strconv.Itoa(page)}, "page_size": {"100"}})
			if readErr != nil {
				return nil, readErr
			}
			var projects []struct {
				Name       string `json:"name"`
				RegistryID int64  `json:"registry_id"`
			}
			if decodeErr := json.Unmarshal(body, &projects); decodeErr != nil {
				return nil, decodeErr
			}
			for _, project := range projects {
				if project.RegistryID == 0 || (target.prefix != "" && target.prefix != project.Name) {
					continue
				}
				if project.Name == "" || strings.ContainsAny(project.Name, "/\\?#") {
					return nil, errors.New("harbor returned an invalid proxy project")
				}
				body, readErr = s.getHarborScanData(ctx, target, scheme, "/api/v2.0/projects/"+url.PathEscape(project.Name)+"/summary", nil)
				if readErr != nil {
					return nil, readErr
				}
				var registry struct {
					Registry struct {
						URL string `json:"url"`
					} `json:"registry"`
				}
				if decodeErr := json.Unmarshal(body, &registry); decodeErr != nil {
					return nil, decodeErr
				}
				proxies = append(proxies, proxyHost(registry.Registry.URL)+"/"+project.Name)
			}
			if len(projects) < 100 {
				return proxies, nil
			}
		}
		return nil, errors.New("harbor proxy discovery exceeded its page limit")
	})
}

func (s *Service) scanProxyDigest(ctx context.Context, target *target, repo name.Repository, options types.RegistryArtifactScanOptions) (*types.RegistryScanReport, error) {
	platform, parseErr := v1.ParsePlatform(options.Platform)
	if parseErr != nil {
		return nil, parseErr
	}
	image, pullErr := remote.Image(repo.Digest(options.Digest), append(target.options, remote.WithPlatform(*platform))...)
	var transportErr *transport.Error
	if errors.As(pullErr, &transportErr) && transportErr.StatusCode == 404 {
		return nil, nil
	}
	if pullErr != nil {
		return nil, classifyBrowseError(pullErr, "Harbor could not cache the running image digest")
	}

	digest, digestErr := image.Digest()
	if digestErr != nil {
		return nil, digestErr
	}
	report, scanErr := s.ScanArtifact(ctx, types.RegistryArtifactScanOptions{
		RegistryID: options.RegistryID, Repository: options.Repository, Digest: digest.String(), Platform: options.Platform, Trigger: options.Trigger,
	})
	if errors.Is(scanErr, common.ErrNotFound) {
		return &types.RegistryScanReport{Digest: digest.String(), Platform: options.Platform, Status: "Not scanned", Scanner: "Harbor"}, nil
	}
	if scanErr != nil {
		return nil, scanErr
	}
	if report.Status == "Unsupported" || report.Status == "Platform unavailable" {
		return nil, nil
	}
	return &report, nil
}
