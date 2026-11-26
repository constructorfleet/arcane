package browse

import (
	"cmp"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"

	"kit.getarcane.app"
	"kit.getarcane.app/cache"

	"getarcane.dev/arcane/v3/internal/common"
	"getarcane.dev/arcane/v3/pkg/pagination"
	"getarcane.dev/types/v3"
)

type (
	harborPage struct {
		Names []string
		Total int
	}
	discoveryTransport struct {
		base http.RoundTripper
		host string
	}
)

const (
	harborPageSize            = 100
	maxDiscoveryPages         = 100
	maxDiscoveryResponseBytes = 4 << 20
)

// Ping must not follow redirects to another origin or negotiate a redirect downgrade.
func (t discoveryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Response != nil || req.URL.Host != t.host {
		return nil, errors.New("project discovery registry probe redirects are not permitted")
	}
	return t.base.RoundTrip(req)
}

func (s *Service) ListNamespaces(ctx context.Context, id string, params pagination.QueryParams) ([]types.ContainerRegistryNamespace, pagination.Response, error) {
	ctx, cancel := s.lookupContext(ctx)
	defer cancel()
	remaining := maxDiscoveryPages
	target, err := s.target(ctx, id)
	if err != nil {
		return nil, pagination.Response{}, err
	}
	var names []string
	catalogErr := errors.New("OCI catalog is unavailable")
	provider, known, _ := s.providers.Get(target.cacheKey)
	if !known || provider != "harbor" {
		names, catalogErr = s.catalogNames(ctx, target)
	}
	var namespaces []string
	if catalogErr == nil {
		for _, repository := range names {
			if namespace, _, ok := strings.Cut(repository, "/"); ok && (target.prefix == "" || strings.HasPrefix(repository, target.prefix+"/")) {
				namespaces = append(namespaces, namespace)
			}
		}
	} else {
		projects, projectErr := s.harborProjects(ctx, target, &remaining)
		if projectErr != nil && target.prefix != "" {
			project := strings.Split(target.prefix, "/")[0]
			if _, repositoryErr := s.harborRepositories(ctx, target, project, &remaining); repositoryErr == nil {
				projects, projectErr = []string{project}, nil
			}
		}
		if projectErr != nil {
			return nil, pagination.Response{}, classifyBrowseError(errors.Join(catalogErr, projectErr), "namespace discovery unavailable; a configured project or known repository may still be accessible")
		}
		namespaces = slices.DeleteFunc(projects, func(project string) bool {
			return target.prefix != "" && project != strings.Split(target.prefix, "/")[0]
		})
		s.providers.Set(cache.Entry[string, string]{Key: target.cacheKey, Value: "harbor"})
	}
	page, response := paginateNames(kit.Unique(namespaces), params)
	items := make([]types.ContainerRegistryNamespace, 0, len(page))
	for _, namespace := range page {
		items = append(items, types.ContainerRegistryNamespace{Name: namespace})
	}
	return items, response, nil
}

func (s *Service) catalogNames(ctx context.Context, target *target) ([]string, error) {
	return s.cachedNames(target.cacheKey, func() ([]string, error) {
		return remote.Catalog(ctx, target.registry, target.options...)
	})
}

func (s *Service) repositoryNames(ctx context.Context, target *target, namespace string) ([]string, error) {
	remaining := maxDiscoveryPages
	if namespace != "" {
		if strings.ContainsAny(namespace, "/\\?#") {
			return nil, common.Classify(common.ErrValidation, errors.New("namespace must be a single project name"))
		}
		if _, err := name.NewRepository(target.registry.Name()+"/"+namespace+"/discovery", target.nameOptions...); err != nil {
			return nil, common.Classify(common.ErrValidation, errors.New("invalid namespace"))
		}
		if target.prefix != "" && namespace != strings.Split(target.prefix, "/")[0] {
			return nil, common.Classify(common.ErrValidation, errors.New("namespace is outside the configured registry scope"))
		}
	}
	provider, found, _ := s.providers.Get(target.cacheKey)
	var names []string
	var catalogErr error
	if !found || provider != "harbor" {
		names, catalogErr = s.catalogNames(ctx, target)
		if catalogErr == nil {
			return slices.DeleteFunc(names, func(repository string) bool {
				return (target.prefix != "" && !strings.HasPrefix(repository, target.prefix+"/")) || (namespace != "" && !strings.HasPrefix(repository, namespace+"/"))
			}), nil
		}
	}
	names = nil
	projects := []string{namespace}
	if namespace == "" && target.prefix != "" {
		projects = []string{strings.Split(target.prefix, "/")[0]}
	} else if namespace == "" {
		var err error
		projects, err = s.harborProjects(ctx, target, &remaining)
		if err != nil {
			return nil, errors.Join(catalogErr, err)
		}
	}
	for _, project := range projects {
		repositories, err := s.harborRepositories(ctx, target, project, &remaining)
		if err != nil {
			return nil, errors.Join(catalogErr, err)
		}
		names = append(names, repositories...)
		if len(names) > maxDiscoveryPages*harborPageSize {
			return nil, errors.New("repository discovery exceeded the item limit; select a namespace")
		}
	}
	s.providers.Set(cache.Entry[string, string]{Key: target.cacheKey, Value: "harbor"})
	return kit.Unique(slices.DeleteFunc(names, func(repository string) bool {
		return (target.prefix != "" && !strings.HasPrefix(repository, target.prefix+"/"))
	})), nil
}

func (s *Service) harborProjects(ctx context.Context, target *target, remaining *int) ([]string, error) {
	return s.cachedNames(target.cacheKey+"\x00harbor-projects", func() ([]string, error) {
		return s.harborNames(ctx, target, "/api/v2.0/projects", remaining)
	})
}

func (s *Service) harborRepositories(ctx context.Context, target *target, project string, remaining *int) ([]string, error) {
	if strings.ContainsAny(project, "/\\?#") || project == "" {
		return nil, errors.New("project discovery returned an invalid project name")
	}
	if _, err := name.NewRepository(target.registry.Name()+"/"+project+"/discovery", target.nameOptions...); err != nil {
		return nil, errors.New("project discovery returned an invalid project name")
	}
	return s.cachedNames(target.cacheKey+"\x00harbor-repositories\x00"+project, func() ([]string, error) {
		names, err := s.harborNames(ctx, target, "/api/v2.0/projects/"+url.PathEscape(project)+"/repositories", remaining)
		if err != nil {
			return nil, err
		}
		return slices.DeleteFunc(names, func(repository string) bool { return !strings.HasPrefix(repository, project+"/") }), nil
	})
}

func (s *Service) harborNames(ctx context.Context, target *target, path string, remaining *int) ([]string, error) {
	probeBudget := 1
	if target.registry.Scheme() == "http" {
		probeBudget = 2
	}
	if *remaining < probeBudget {
		return nil, errors.New("project discovery request limit reached; select a namespace")
	}
	// Native Ping may race HTTPS and the explicitly permitted HTTP fallback.
	*remaining -= probeBudget
	scheme, err := s.harborScheme(ctx, target)
	if err != nil {
		return nil, err
	}
	var names []string
	seenPages := make(map[string]struct{})
	seenNames := make(map[string]struct{})
	for page := 1; page <= maxDiscoveryPages; page++ {
		if *remaining <= 0 {
			return nil, errors.New("project discovery request limit reached; select a namespace")
		}
		*remaining--
		result, pageErr := s.harborPage(ctx, target, path, page, scheme)
		if pageErr != nil {
			return nil, pageErr
		}
		if len(result.Names) == 0 {
			if result.Total > len(names) {
				return nil, errors.New("project discovery ended before the reported total")
			}
			return names, nil
		}
		fingerprint := kit.SHA256Hex([]byte(strings.Join(result.Names, "\x00")))
		if _, exists := seenPages[fingerprint]; exists {
			return nil, errors.New("project discovery API repeated a page")
		}
		seenPages[fingerprint] = struct{}{}
		before := len(names)
		for _, value := range result.Names {
			if _, found := seenNames[value]; !found {
				seenNames[value] = struct{}{}
				names = append(names, value)
			}
		}
		if len(names) == before {
			return nil, errors.New("project discovery pagination made no progress")
		}
		if (result.Total > 0 && len(names) >= result.Total) || (result.Total <= 0 && len(result.Names) < harborPageSize) {
			return names, nil
		}
	}
	return nil, errors.New("project discovery exceeded the page limit")
}

func (s *Service) harborScheme(ctx context.Context, target *target) (string, error) {
	base := cmp.Or(s.transport, http.DefaultTransport)
	challenge, err := transport.Ping(ctx, target.registry, discoveryTransport{base: base, host: target.registry.RegistryStr()})
	if err != nil {
		return "", err
	}
	if challenge.Insecure {
		return "http", nil
	}
	return "https", nil
}

// The Harbor API accepts registry username/password, including CLI and robot secrets.
func (s *Service) harborGET(ctx context.Context, target *target, scheme, path string, query url.Values, mediaType string) (*http.Response, error) {
	auth, err := target.auth.Authorization()
	if err != nil {
		return nil, errors.New("failed to resolve Harbor credentials")
	}
	endpoint := url.URL{Scheme: scheme, Host: target.registry.RegistryStr(), Path: path, RawQuery: query.Encode()}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if mediaType != "" {
		req.Header.Set("X-Accept-Vulnerabilities", mediaType)
	}
	if auth.Username != "" || auth.Password != "" {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	client := &http.Client{Transport: s.transport, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	return client.Do(req)
}

func (s *Service) harborPage(ctx context.Context, target *target, path string, page int, scheme string) (harborPage, error) {
	query := url.Values{"page": {strconv.Itoa(page)}, "page_size": {strconv.Itoa(harborPageSize)}}
	response, err := s.harborGET(ctx, target, scheme, path, query, "")
	if err != nil {
		return harborPage{}, err
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxDiscoveryResponseBytes+1))
	_ = response.Body.Close()
	if err != nil {
		return harborPage{}, err
	}
	if response.StatusCode != http.StatusOK {
		message := fmt.Sprintf("project discovery API %s: HTTP %d %s", path, response.StatusCode, http.StatusText(response.StatusCode))
		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			return harborPage{}, errdefs.ErrPermissionDenied.WithMessage(message)
		}
		return harborPage{}, errors.New(message)
	}
	if len(body) > maxDiscoveryResponseBytes {
		return harborPage{}, errors.New("project discovery response exceeds size limit")
	}
	var rows []struct {
		Name string `json:"name"`
	}
	if decodeErr := json.Unmarshal(body, &rows, jsontext.AllowDuplicateNames(false)); decodeErr != nil || rows == nil {
		return harborPage{}, errors.New("project discovery API did not return a valid list")
	}
	if len(rows) > harborPageSize {
		return harborPage{}, errors.New("project discovery API exceeded the requested page size")
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Name) == "" {
			return harborPage{}, errors.New("project discovery API returned an unnamed entry")
		}
		names = append(names, row.Name)
	}
	total, _ := strconv.Atoi(response.Header.Get("X-Total-Count"))
	return harborPage{Names: names, Total: total}, nil
}
