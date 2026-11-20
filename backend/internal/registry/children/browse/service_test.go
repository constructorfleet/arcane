package browse

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/containerd/errdefs"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getarcane.dev/arcane/v3/internal/common"
	"getarcane.dev/arcane/v3/pkg/pagination"
	"getarcane.dev/types/v3"
)

type (
	testRegistries        map[string]types.ContainerRegistryCredential
	testDiscoveryRegistry struct {
		t            *testing.T
		mode         string
		httpCalls    int
		foreignCalls int
		apiCalls     int
	}
)

func (f *testDiscoveryRegistry) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != "registry.example" {
		f.foreignCalls++
	}
	if req.URL.Scheme == "http" {
		f.httpCalls++
	}
	status, body := http.StatusOK, ""
	header := make(http.Header)
	switch req.URL.Path {
	case "/v2/":
		assert.Empty(f.t, req.Header.Get("Authorization"), "scheme negotiation must never send credentials")
		switch f.mode {
		case "http-opt-in", "http-not-allowed":
			if req.URL.Scheme == "https" {
				return nil, errors.New("synthetic HTTPS unavailable")
			}
		case "cross-origin-redirect":
			status = http.StatusFound
			header.Set("Location", "https://foreign.example/v2/")
		case "https-downgrade-redirect":
			status = http.StatusFound
			header.Set("Location", "http://registry.example/v2/")
		}
	case "/api/v2.0/projects":
		f.apiCalls++
		if f.mode == "https-denied" {
			assert.Equal(f.t, "https", req.URL.Scheme)
		} else {
			assert.Equal(f.t, "http", req.URL.Scheme)
		}
		user, password, ok := req.BasicAuth()
		assert.True(f.t, ok)
		assert.Equal(f.t, "synthetic-user", user)
		assert.Equal(f.t, "synthetic-secret", password)
		body = `[{"name":"team"}]`
		if f.mode == "https-denied" {
			status = http.StatusForbidden
		}
	default:
		status = http.StatusUnauthorized
	}
	return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
}

func newTestRegistry(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	return "localhost:" + serverURL.Port()
}

func (r testRegistries) add(registryURL, username, token string) string {
	id := fmt.Sprintf("registry-%d", len(r)+1)
	r[id] = types.ContainerRegistryCredential{URL: registryURL, Username: username, Token: token, Enabled: true}
	return id
}

func newTestService(registries testRegistries) *Service {
	return NewService(
		func(_ context.Context, id string) (types.ContainerRegistry, *types.ContainerRegistryCredential, error) {
			credential, ok := registries[id]
			if !ok {
				return types.ContainerRegistry{}, nil, common.Classify(common.ErrNotFound, errors.New("registry not found"))
			}
			registryRecord := types.ContainerRegistry{ID: id, URL: credential.URL}
			if credential.Username == "" {
				return registryRecord, nil, nil
			}
			return registryRecord, &credential, nil
		},
		func(ctx context.Context) (context.Context, context.CancelFunc) {
			return context.WithTimeout(ctx, time.Minute)
		},
		nil,
	)
}

func platformImage(t *testing.T, platform v1.Platform, created time.Time) v1.Image {
	t.Helper()

	img, err := random.Image(128, 2)
	require.NoError(t, err)
	cfg, err := img.ConfigFile()
	require.NoError(t, err)
	img, err = mutate.ConfigFile(img, &v1.ConfigFile{
		Architecture: platform.Architecture,
		Created:      v1.Time{Time: created},
		History:      cfg.History,
		OS:           platform.OS,
		RootFS:       cfg.RootFS,
		Config:       cfg.Config,
		Variant:      platform.Variant,
	})
	require.NoError(t, err)
	return img
}

func mustParseReference(t *testing.T, imageRef string) name.Reference {
	t.Helper()
	ref, err := name.ParseReference(imageRef)
	require.NoError(t, err)
	return ref
}

func writeImage(t *testing.T, imageRef string, img v1.Image, options ...remote.Option) {
	t.Helper()
	require.NoError(t, remote.Write(mustParseReference(t, imageRef), img, options...))
}

func imageSize(t *testing.T, img v1.Image) int64 {
	t.Helper()
	digest, err := img.Digest()
	require.NoError(t, err)
	platform, err := tagPlatform(img, nil, digest)
	require.NoError(t, err)
	return platform.Size
}

func browseParams(search string) pagination.QueryParams {
	return pagination.QueryParams{
		Search: search,
		Start:  0, Limit: 20,
	}
}

func TestService_ListRepositories(t *testing.T) {
	t.Run("insecure option still prefers verified HTTPS for project discovery", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v2/" {
				w.WriteHeader(http.StatusOK)
				return
			}
			if r.URL.Path == "/v2/_catalog" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			user, password, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "synthetic-user", user)
			assert.Equal(t, "synthetic-secret", password)
			_, _ = io.WriteString(w, `[{"name":"team"}]`)
		}))
		t.Cleanup(server.Close)
		svc := newTestService(testRegistries{})
		svc.transport = server.Client().Transport
		svc.resolve = func(context.Context, string) (types.ContainerRegistry, *types.ContainerRegistryCredential, error) {
			return types.ContainerRegistry{URL: server.URL, Insecure: true}, &types.ContainerRegistryCredential{Username: "synthetic-user", Token: "synthetic-secret"}, nil
		}
		projects, _, err := svc.ListNamespaces(t.Context(), "tls-first", browseParams(""))
		require.NoError(t, err)
		require.Equal(t, []types.ContainerRegistryNamespace{{Name: "team"}}, projects)
	})
	host := newTestRegistry(t)
	img := platformImage(t, v1.Platform{OS: "linux", Architecture: "amd64"}, time.Now())
	writeImage(t, host+"/team/api:1.0", img)
	writeImage(t, host+"/team/web:1.0", img)
	writeImage(t, host+"/other/tool:1.0", img)

	registries := testRegistries{}
	svc := newTestService(registries)
	ctx := t.Context()

	id := registries.add(host, "", "")
	repositories, page, err := svc.ListRepositories(ctx, id, "", browseParams(""))
	require.NoError(t, err)
	assert.Equal(t, []string{"other/tool", "team/api", "team/web"}, repositoryNames(repositories))
	assert.EqualValues(t, 3, page.TotalItems)

	repositories, _, err = svc.ListRepositories(ctx, id, "", browseParams("web"))
	require.NoError(t, err)
	assert.Equal(t, []string{"team/web"}, repositoryNames(repositories))

	namespacedID := registries.add("http://"+host+"/team/", "", "")
	repositories, _, err = svc.ListRepositories(ctx, namespacedID, "", browseParams(""))
	require.NoError(t, err)
	assert.Equal(t, []string{"team/api", "team/web"}, repositoryNames(repositories))
	namespaces, _, err := svc.ListNamespaces(ctx, id, browseParams(""))
	require.NoError(t, err)
	require.Equal(t, []types.ContainerRegistryNamespace{{Name: "other"}, {Name: "team"}}, namespaces)
	repositories, _, err = svc.ListRepositories(ctx, id, "team", browseParams(""))
	require.NoError(t, err)
	require.Equal(t, []string{"team/api", "team/web"}, repositoryNames(repositories))

	t.Run("cached project provider keeps upstream auth categories", func(t *testing.T) {
		status := http.StatusOK
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v2/":
				w.WriteHeader(http.StatusOK)
			case "/v2/_catalog":
				w.WriteHeader(http.StatusUnauthorized)
			case "/api/v2.0/projects":
				_, _ = io.WriteString(w, `[{"name":"team"}]`)
			default:
				w.WriteHeader(status)
			}
		}))
		t.Cleanup(server.Close)
		authProfiles := testRegistries{}
		authID := authProfiles.add(server.URL, "user", "synthetic-only")
		authService := newTestService(authProfiles)
		_, _, discoveryErr := authService.ListNamespaces(t.Context(), authID, browseParams(""))
		require.NoError(t, discoveryErr)
		for _, code := range []int{http.StatusUnauthorized, http.StatusForbidden} {
			status = code
			_, _, discoveryErr = authService.ListRepositories(t.Context(), authID, "team", browseParams(""))
			require.ErrorIs(t, discoveryErr, common.ErrBadRequest)
			require.NotErrorIs(t, discoveryErr, common.ErrUnavailable)
		}
	})
	t.Run("all-project repository enumeration shares a request budget", func(t *testing.T) {
		apiCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v2/" {
				w.WriteHeader(http.StatusOK)
				return
			}
			if r.URL.Path == "/v2/_catalog" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			apiCalls++
			if r.URL.Path == "/api/v2.0/projects" {
				w.Header().Set("X-Total-Count", "100")
				_, _ = io.WriteString(w, "[")
				for i := 0; i < harborPageSize; i++ {
					if i > 0 {
						_, _ = io.WriteString(w, ",")
					}
					_, _ = fmt.Fprintf(w, `{"name":"project-%d"}`, i)
				}
				_, _ = io.WriteString(w, "]")
				return
			}
			_, _ = io.WriteString(w, "[]")
		}))
		t.Cleanup(server.Close)
		budgetProfiles := testRegistries{}
		budgetID := budgetProfiles.add(server.URL, "", "")
		budgetService := newTestService(budgetProfiles)
		items, _, budgetErr := budgetService.ListRepositories(t.Context(), budgetID, "", browseParams(""))
		require.ErrorContains(t, budgetErr, "request limit")
		require.Nil(t, items)
		require.Positive(t, apiCalls)
		require.LessOrEqual(t, apiCalls*3, maxDiscoveryPages, "each HTTP-only list reserves two probe slots and consumes an API page")
	})
}

func TestService_ListRepositoryTagsSingleImage(t *testing.T) {
	host := newTestRegistry(t)
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	img := platformImage(t, v1.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"}, created)
	writeImage(t, host+"/team/api:1.0", img)
	writeImage(t, host+"/team/api:2.0", img)

	registries := testRegistries{}
	svc := newTestService(registries)
	id := registries.add(host, "", "")

	tags, page, err := svc.ListRepositoryTags(t.Context(), id, "team/api", pagination.QueryParams{Start: 0, Limit: 1})
	require.NoError(t, err)
	assert.EqualValues(t, 2, page.TotalItems)
	require.Len(t, tags, 1)

	digest, err := img.Digest()
	require.NoError(t, err)
	size := imageSize(t, img)

	tag := tags[0]
	assert.Equal(t, "1.0", tag.Name)
	assert.Empty(t, tag.Error)
	assert.Equal(t, digest.String(), tag.Digest)
	assert.Equal(t, size, tag.Size)
	require.NotNil(t, tag.Created)
	assert.True(t, created.Equal(*tag.Created))
	require.Len(t, tag.Platforms, 1)
	assert.Equal(t, "linux", tag.Platforms[0].OS)
	assert.Equal(t, "arm64", tag.Platforms[0].Architecture)
	assert.Equal(t, "v8", tag.Platforms[0].Variant)
}

func TestService_ListRepositoryTagsIndex(t *testing.T) {
	host := newTestRegistry(t)
	amd64 := v1.Platform{OS: "linux", Architecture: "amd64"}
	arm64 := v1.Platform{OS: "linux", Architecture: "arm64"}
	amd64Image := platformImage(t, amd64, time.Now())
	arm64Image := platformImage(t, arm64, time.Now())
	attestation, err := random.Image(32, 1)
	require.NoError(t, err)

	index := mutate.AppendManifests(empty.Index,
		mutate.IndexAddendum{Add: arm64Image, Platform: &arm64},
		mutate.IndexAddendum{Add: amd64Image, Platform: &amd64},
		mutate.IndexAddendum{Add: attestation, Platform: &v1.Platform{OS: "unknown", Architecture: "unknown"}},
	)
	ref, err := name.ParseReference(host + "/team/api:multi")
	require.NoError(t, err)
	require.NoError(t, remote.WriteIndex(ref, index))

	registries := testRegistries{}
	svc := newTestService(registries)
	id := registries.add(host, "", "")

	tags, _, err := svc.ListRepositoryTags(t.Context(), id, "team/api", browseParams(""))
	require.NoError(t, err)
	require.Len(t, tags, 1)

	amd64Size := imageSize(t, amd64Image)
	arm64Size := imageSize(t, arm64Image)

	tag := tags[0]
	assert.Empty(t, tag.Error)
	assert.Nil(t, tag.Created)
	assert.Equal(t, amd64Size+arm64Size, tag.Size)
	require.Len(t, tag.Platforms, 2)
	assert.Equal(t, "amd64", tag.Platforms[0].Architecture)
	assert.Equal(t, amd64Size, tag.Platforms[0].Size)
	assert.Equal(t, "arm64", tag.Platforms[1].Architecture)
}

func TestService_DeleteRepositoryTag(t *testing.T) {
	host := newTestRegistry(t)
	deleted := platformImage(t, v1.Platform{OS: "linux", Architecture: "amd64"}, time.Now())
	writeImage(t, host+"/team/api:1.0", deleted)
	writeImage(t, host+"/team/api:2.0", platformImage(t, v1.Platform{OS: "linux", Architecture: "amd64"}, time.Now()))

	registries := testRegistries{}
	svc := newTestService(registries)
	id := registries.add(host, "", "")
	ctx := t.Context()

	digest, err := svc.DeleteRepositoryTag(ctx, id, "team/api", "1.0")
	require.NoError(t, err)
	deletedDigest, err := deleted.Digest()
	require.NoError(t, err)
	assert.Equal(t, deletedDigest.String(), digest)

	_, err = remote.Head(mustParseReference(t, host+"/team/api@"+digest))
	require.Error(t, err)
	_, err = remote.Head(mustParseReference(t, host+"/team/api:2.0"))
	require.NoError(t, err)
}

func TestService_BrowseUsesStoredCredentials(t *testing.T) {
	for _, tt := range []struct {
		name             string
		projects         []string
		httpCalls        int
		apiCalls         int
		insecure         bool
		permissionDenied bool
	}{
		{name: "https-denied", insecure: true, apiCalls: 1, permissionDenied: true},
		{name: "http-opt-in", insecure: true, httpCalls: 2, apiCalls: 1, projects: []string{"team"}},
		{name: "http-not-allowed"},
		{name: "cross-origin-redirect"},
		{name: "https-downgrade-redirect"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(testRegistries{})
			svc.resolve = func(context.Context, string) (types.ContainerRegistry, *types.ContainerRegistryCredential, error) {
				return types.ContainerRegistry{URL: "registry.example", Insecure: tt.insecure},
					&types.ContainerRegistryCredential{Username: "synthetic-user", Token: "synthetic-secret"}, nil
			}
			fake := &testDiscoveryRegistry{t: t, mode: tt.name}
			svc.transport = fake
			// Skip the separate OCI catalog path to exercise the provider transport directly.
			target, err := svc.target(t.Context(), "transport")
			require.NoError(t, err)
			remaining := maxDiscoveryPages
			projects, err := svc.harborProjects(t.Context(), target, &remaining)
			require.Equal(t, tt.projects == nil, err != nil)
			require.Equal(t, tt.projects, projects)
			require.Equal(t, tt.httpCalls, fake.httpCalls)
			require.Equal(t, tt.apiCalls, fake.apiCalls)
			if tt.permissionDenied {
				require.ErrorIs(t, err, errdefs.ErrPermissionDenied)
			}
			require.Zero(t, fake.foreignCalls)
		})
	}

	registryHandler := registry.New(registry.Logger(log.New(io.Discard, "", 0)))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "robot" || password != "secret" {
			w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		registryHandler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	host := "localhost:" + serverURL.Port()

	writeImage(t, host+"/private/app:1.0", platformImage(t, v1.Platform{OS: "linux", Architecture: "amd64"}, time.Now()),
		remote.WithAuth(&authn.Basic{Username: "robot", Password: "secret"}))

	registries := testRegistries{}
	svc := newTestService(registries)
	ctx := t.Context()

	id := registries.add(host, "robot", "secret")
	repositories, _, err := svc.ListRepositories(ctx, id, "", browseParams(""))
	require.NoError(t, err)
	assert.Equal(t, []string{"private/app"}, repositoryNames(repositories))

	anonymousID := registries.add(host, "", "")
	_, _, err = svc.ListRepositories(ctx, anonymousID, "", browseParams(""))
	require.ErrorIs(t, err, common.ErrBadRequest)
	for _, username := range []string{"oidc-user", "robot$constructorfleet+reader"} {
		t.Run(username, func(t *testing.T) {
			projectCalls, repositoryCalls := 0, 0
			expectedToken := "synthetic-cli-or-robot-secret"
			denyProjects := false
			harborServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v2/" {
					w.WriteHeader(http.StatusOK)
					return
				}
				if r.URL.Path == "/v2/_catalog" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				user, token, ok := r.BasicAuth()
				if !ok || user != username || token != expectedToken {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				assert.Equal(t, "100", r.URL.Query().Get("page_size"))
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Total-Count", "2")
				switch r.URL.Path {
				case "/api/v2.0/projects":
					projectCalls++
					if denyProjects {
						w.WriteHeader(http.StatusForbidden)
						return
					}
					if r.URL.Query().Get("page") == "1" {
						_, _ = io.WriteString(w, `[{"name":"constructorfleet"}]`)
					} else {
						_, _ = io.WriteString(w, `[{"name":"ajanis"}]`)
					}
				case "/api/v2.0/projects/constructorfleet/repositories":
					repositoryCalls++
					if r.URL.Query().Get("page") == "1" {
						_, _ = io.WriteString(w, `[{"name":"constructorfleet/compass"}]`)
					} else {
						_, _ = io.WriteString(w, `[{"name":"constructorfleet/platform/api"}]`)
					}
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			t.Cleanup(harborServer.Close)
			harborProfiles := testRegistries{}
			harborID := harborProfiles.add(harborServer.URL, username, expectedToken)
			harborService := newTestService(harborProfiles)
			projects, harborPaging, harborErr := harborService.ListNamespaces(t.Context(), harborID, browseParams(""))
			require.NoError(t, harborErr)
			require.Equal(t, []types.ContainerRegistryNamespace{{Name: "ajanis"}, {Name: "constructorfleet"}}, projects)
			require.EqualValues(t, 2, harborPaging.TotalItems)
			repos, harborPaging, harborErr := harborService.ListRepositories(t.Context(), harborID, "constructorfleet", browseParams(""))
			require.NoError(t, harborErr)
			require.Equal(t, []string{"constructorfleet/compass", "constructorfleet/platform/api"}, repositoryNames(repos))
			require.EqualValues(t, 2, harborPaging.TotalItems)
			require.Equal(t, 2, projectCalls)
			require.Equal(t, 2, repositoryCalls)
			_, _, harborErr = harborService.ListRepositories(t.Context(), harborID, "constructorfleet", browseParams("compass"))
			require.NoError(t, harborErr)
			require.Equal(t, 2, repositoryCalls)
			denyProjects = true
			scopedID := harborProfiles.add(harborServer.URL+"/constructorfleet", username, expectedToken)
			projects, _, harborErr = harborService.ListNamespaces(t.Context(), scopedID, browseParams(""))
			require.NoError(t, harborErr)
			require.Equal(t, []types.ContainerRegistryNamespace{{Name: "constructorfleet"}}, projects)
			_, _, harborErr = harborService.ListRepositories(t.Context(), scopedID, "ajanis", browseParams(""))
			require.Error(t, harborErr)
			// Explicit project browsing does not need permission to list every project.
			explicitID := harborProfiles.add(harborServer.URL, username, expectedToken)
			repos, _, harborErr = harborService.ListRepositories(t.Context(), explicitID, "constructorfleet", browseParams(""))
			require.NoError(t, harborErr)
			require.Len(t, repos, 2)
			expectedToken = "replacement-synthetic-secret"
			credential := harborProfiles[harborID]
			credential.Token = expectedToken
			harborProfiles[harborID] = credential
			_, _, harborErr = harborService.ListNamespaces(t.Context(), harborID, browseParams(""))
			require.Error(t, harborErr, "a changed credential must not reuse the previous project's cache")
		})
	}
}

func TestService_BrowseErrors(t *testing.T) {
	host := newTestRegistry(t)
	writeImage(t, host+"/team/api:1.0", platformImage(t, v1.Platform{OS: "linux", Architecture: "amd64"}, time.Now()))

	registries := testRegistries{}
	svc := newTestService(registries)
	id := registries.add(host, "", "")
	namespacedID := registries.add("http://"+host+"/team/", "", "")

	tests := []struct {
		wantErr error
		call    func(ctx context.Context) error
		name    string
	}{
		{
			name: "unknown registry",
			call: func(ctx context.Context) error {
				_, _, err := svc.ListRepositories(ctx, "missing", "", browseParams(""))
				return err
			},
			wantErr: common.ErrNotFound,
		},
		{
			name: "empty repository",
			call: func(ctx context.Context) error {
				_, _, err := svc.ListRepositoryTags(ctx, id, " / ", browseParams(""))
				return err
			},
			wantErr: common.ErrValidation,
		},
		{
			name: "tags outside registry namespace",
			call: func(ctx context.Context) error {
				_, _, err := svc.ListRepositoryTags(ctx, namespacedID, "other/tool", browseParams(""))
				return err
			},
			wantErr: common.ErrValidation,
		},
		{
			name: "delete outside registry namespace",
			call: func(ctx context.Context) error {
				_, err := svc.DeleteRepositoryTag(ctx, namespacedID, "other/tool", "1.0")
				return err
			},
			wantErr: common.ErrValidation,
		},
		{
			name: "delete missing tag",
			call: func(ctx context.Context) error {
				_, err := svc.DeleteRepositoryTag(ctx, id, "team/api", "missing")
				return err
			},
			wantErr: common.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, tt.call(t.Context()), tt.wantErr)
		})
	}
	t.Run("project discovery failures and redirects remain failures", func(t *testing.T) {
		redirected := 0
		other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected++; w.WriteHeader(http.StatusOK) }))
		t.Cleanup(other.Close)
		for _, mode := range []string{"denied", "missing", "html", "object", "redirect", "repeat", "partial", "oversize", "too-many-rows"} {
			t.Run(mode, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/v2/" {
						w.WriteHeader(http.StatusOK)
						return
					}
					if r.URL.Path == "/v2/_catalog" {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					switch mode {
					case "denied":
						w.WriteHeader(http.StatusForbidden)
					case "missing":
						w.WriteHeader(http.StatusNotFound)
					case "html":
						_, _ = io.WriteString(w, "<html>login page</html>")
					case "object":
						_, _ = io.WriteString(w, `{}`)
					case "redirect":
						http.Redirect(w, r, other.URL, http.StatusFound)
					case "repeat":
						w.Header().Set("X-Total-Count", "200")
						_, _ = io.WriteString(w, `[{"name":"repeated"}]`)
					case "partial":
						w.Header().Set("X-Total-Count", "200")
						_, _ = io.WriteString(w, `[]`)
					case "oversize":
						_, _ = io.WriteString(w, strings.Repeat(" ", maxDiscoveryResponseBytes+1))
					case "too-many-rows":
						_, _ = io.WriteString(w, "[")
						for i := 0; i <= harborPageSize; i++ {
							if i > 0 {
								_, _ = io.WriteString(w, ",")
							}
							_, _ = fmt.Fprintf(w, `{"name":"project-%d"}`, i)
						}
						_, _ = io.WriteString(w, "]")
					}
				}))
				t.Cleanup(server.Close)
				failedProfiles := testRegistries{}
				failedID := failedProfiles.add(server.URL, "synthetic-user", "synthetic-secret")
				failedService := newTestService(failedProfiles)
				items, _, err := failedService.ListNamespaces(t.Context(), failedID, browseParams(""))
				require.Error(t, err)
				require.Nil(t, items)
				require.NotContains(t, err.Error(), "synthetic-secret")
			})
		}
		require.Zero(t, redirected, "discovery must not forward credentials by following redirects")
	})
}

func repositoryNames(repositories []types.ContainerRegistryRepository) []string {
	names := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		names = append(names, repository.Name)
	}
	return names
}
