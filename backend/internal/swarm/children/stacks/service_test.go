package stacks

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/libtnb/sqlite"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"getarcane.dev/arcane/v3/internal/common"
	"getarcane.dev/arcane/v3/internal/config"
	"getarcane.dev/arcane/v3/internal/database"
	dockerInternal "getarcane.dev/arcane/v3/internal/docker"
	"getarcane.dev/arcane/v3/internal/environment"
	"getarcane.dev/arcane/v3/internal/settings"
	"getarcane.dev/arcane/v3/pkg/httpx"
	"getarcane.dev/arcane/v3/pkg/pagination"
	"getarcane.dev/arcane/v3/pkg/projects"
	"getarcane.dev/types/v3"
)

type (
	stackSourceDeployRecorder struct {
		lastReq types.SwarmStackDeployRequest
		calls   int
	}
	sharedStackDockerRecorder struct {
		service  *swarm.ServiceSpec
		config   *swarm.ConfigSpec
		secret   *swarm.SecretSpec
		requests []string
		mu       sync.Mutex
	}
)

var fullStackSourceSubmitted = map[string]bool{
	"composeContent":  true,
	"overrideContent": true,
	"envContent":      true,
	"files":           true,
}

func setupSwarmServiceTestDB(t *testing.T) *database.DB {
	t.Helper()
	t.Setenv("ARCANE_PROJECTS_DIRECTORY", t.TempDir())
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&settings.SettingVariable{}, &environment.Environment{}))
	return &database.DB{DB: db}
}

func newSettingsServiceForSwarmTest(t testing.TB, ctx context.Context, db *database.DB) (*settings.SettingsService, error) {
	t.Helper()
	settingsEnv, err := settings.LoadEnv()
	require.NoError(t, err)
	svc, err := settings.NewSettingsService(settings.SettingsServiceParams{
		Ctx:    ctx,
		DB:     db,
		Config: &config.Config{},
		Env:    settingsEnv,
	})
	if err == nil {
		t.Cleanup(func() { require.NoError(t, svc.Stop(context.WithoutCancel(t.Context()))) })
	}
	return svc, err
}

func stubStackSourceUpdateDeploy(t *testing.T) *stackSourceDeployRecorder {
	t.Helper()
	original := deployStackAfterSourceUpdate
	t.Cleanup(func() { deployStackAfterSourceUpdate = original })
	rec := &stackSourceDeployRecorder{}
	deployStackAfterSourceUpdate = func(_ *Service, _ context.Context, _ string, req types.SwarmStackDeployRequest) (*types.SwarmStackDeployResponse, error) {
		rec.calls++
		rec.lastReq = req
		return &types.SwarmStackDeployResponse{Name: req.Name}, nil
	}
	return rec
}

func TestService_UpdateAndGetStackSource_UsesStoredFilesWithoutSwarmManager(t *testing.T) {
	ctx := t.Context()
	db := setupSwarmServiceTestDB(t)
	rootDir := t.TempDir()
	t.Setenv("ARCANE_SWARM_STACK_SOURCES_DIRECTORY", rootDir)

	settingsSvc, err := newSettingsServiceForSwarmTest(t, ctx, db)
	require.NoError(t, err)

	svc := NewService(ServiceParams{
		SettingsService: settingsSvc,
	})
	deployRec := stubStackSourceUpdateDeploy(t)

	updated, err := svc.UpdateStackSource(ctx, "0", "demo-stack", StackSourceUpdate{
		Source: types.SwarmStackSource{
			ComposeContent: "services:\n  web:\n    image: nginx:alpine\n",
			EnvContent:     "FOO=bar\n",
		},
		Submitted: fullStackSourceSubmitted,
	})
	require.NoError(t, err)
	require.Equal(t, "demo-stack", updated.Name)
	// Saving stack source must trigger an actual stack deploy (#3463).
	require.Equal(t, 1, deployRec.calls)
	// The edit-path redeploy must carry registry auth, the same as Git Sync (#3778):
	// a service added in the edit has no previous spec to fall back on.
	require.True(t, deployRec.lastReq.WithRegistryAuth)

	composePath := filepath.Join(rootDir, "0", "demo-stack", "compose.yaml")
	envPath := filepath.Join(rootDir, "0", "demo-stack", ".env")
	require.FileExists(t, composePath)
	require.FileExists(t, envPath)

	source, err := svc.GetStackSource(ctx, "0", "demo-stack")
	require.NoError(t, err)
	require.Equal(t, updated.ComposeContent, source.ComposeContent)
	require.Equal(t, updated.EnvContent, source.EnvContent)

	_, err = svc.UpdateStackSource(ctx, "0", "demo-stack", StackSourceUpdate{
		Source: types.SwarmStackSource{
			ComposeContent: "services:\n  web:\n    image: nginx:alpine\n",
			Files: []types.SwarmSyncFile{
				{RelativePath: "config/nginx.conf", Content: []byte("worker_processes 1;")},
				{RelativePath: "scripts/setup.sh", Content: []byte("#!/bin/sh")},
				// The editor's "Create File" sends a root .env as a workspace file, not as envContent.
				{RelativePath: ".env", Content: []byte("FOO=baz\n")},
			},
		},
		Submitted: fullStackSourceSubmitted,
	})
	require.NoError(t, err)
	require.Equal(t, "FOO=baz\n", deployRec.lastReq.EnvContent)
	require.FileExists(t, filepath.Join(rootDir, "0", "demo-stack", "config", "nginx.conf"))
	require.FileExists(t, filepath.Join(rootDir, "0", "demo-stack", "scripts", "setup.sh"))

	source, err = svc.GetStackSource(ctx, "0", "demo-stack")
	require.NoError(t, err)
	require.Equal(t, "FOO=baz\n", source.EnvContent)
	require.Len(t, source.Files, 2)

	_, err = svc.UpdateStackSource(ctx, "0", "demo-stack", StackSourceUpdate{
		Source: types.SwarmStackSource{
			ComposeContent: "services:\n  web:\n    image: nginx:alpine\n",
			Files: []types.SwarmSyncFile{
				{RelativePath: "config/nginx.conf", Content: []byte("worker_processes auto;")},
			},
		},
		Submitted: fullStackSourceSubmitted,
	})
	require.NoError(t, err)
	require.NoFileExists(t, filepath.Join(rootDir, "0", "demo-stack", "scripts", "setup.sh"))
	source, err = svc.GetStackSource(ctx, "0", "demo-stack")
	require.NoError(t, err)
	require.Len(t, source.Files, 1)
	require.Equal(t, []byte("worker_processes auto;"), source.Files[0].Content)
	t.Run("source persistence failure stops Docker deployment", func(t *testing.T) {
		failureCtx := t.Context()
		blocked := filepath.Join(t.TempDir(), "not-a-directory")
		require.NoError(t, os.WriteFile(blocked, []byte("owned source-storage failure fixture"), 0o600))
		t.Setenv("ARCANE_SWARM_STACK_SOURCES_DIRECTORY", blocked)
		failureDB := setupSwarmServiceTestDB(t)
		failureSettings, setupErr := newSettingsServiceForSwarmTest(t, failureCtx, failureDB)
		require.NoError(t, setupErr)
		var requests []string
		var requestsMu sync.Mutex
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestsMu.Lock()
			requests = append(requests, r.Method+" "+r.URL.Path)
			requestsMu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			resource := strings.TrimPrefix(r.URL.Path, "/v1.41")
			switch {
			case r.Method == http.MethodGet && ((strings.HasPrefix(resource, "/networks/") && resource != "/networks") || (strings.HasPrefix(resource, "/containers/") && resource != "/containers/json")):
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message":"owned fake resource not found"}`))
			case r.Method == http.MethodGet:
				_, _ = w.Write([]byte(`[]`))
			case r.Method == http.MethodPost && resource == "/networks/create":
				_, _ = w.Write([]byte(`{"Id":"owned-fake-network","Warning":""}`))
			case r.Method == http.MethodPost && resource == "/services/create":
				_, _ = w.Write([]byte(`{"ID":"owned-fake-service","Warnings":[]}`))
			default:
				http.Error(w, "unexpected fake Docker request", http.StatusInternalServerError)
			}
		}))
		t.Cleanup(server.Close)
		dockerClient, setupErr := client.New(client.WithHost(server.URL), client.WithAPIVersion("1.41"), client.WithHTTPClient(server.Client()))
		require.NoError(t, setupErr)
		t.Cleanup(func() { _ = dockerClient.Close() })
		service := NewService(ServiceParams{
			DockerService:   dockerInternal.NewDockerClientService(failureCtx, failureDB, failureSettings, dockerInternal.ClientOptions{}).WithClient(dockerClient),
			EnsureManager:   func(context.Context) error { return nil },
			SettingsService: failureSettings,
		})
		response, deployErr := service.DeployStack(failureCtx, "0", types.SwarmStackDeployRequest{
			Name:           "owned-storage-failure",
			ComposeContent: "services:\n  web:\n    image: nginx:alpine\n    deploy:\n      replicas: 0\n",
			WorkingDir:     t.TempDir(),
			ResolveImage:   "never",
		})
		requestsMu.Lock()
		t.Logf("source-storage failure response=%+v error=%v fake-Docker requests=%v", response, deployErr, requests)
		observed := slices.Clone(requests)
		requestsMu.Unlock()
		require.ErrorContains(t, deployErr, "failed to create swarm stack source root")
		require.Nil(t, response)
		require.Empty(t, observed, "source persistence failure must stop before Docker calls")
	})
}

func TestService_UpdateAndGetStackSource_RoundTripsOverride(t *testing.T) {
	ctx := t.Context()
	db := setupSwarmServiceTestDB(t)
	rootDir := t.TempDir()
	t.Setenv("ARCANE_SWARM_STACK_SOURCES_DIRECTORY", rootDir)

	settingsSvc, err := newSettingsServiceForSwarmTest(t, ctx, db)
	require.NoError(t, err)

	svc := NewService(ServiceParams{
		SettingsService: settingsSvc,
	})
	stubStackSourceUpdateDeploy(t)

	overridePath := filepath.Join(rootDir, "0", "demo-stack", "compose.override.yaml")

	updated, err := svc.UpdateStackSource(ctx, "0", "demo-stack", StackSourceUpdate{
		Source: types.SwarmStackSource{
			ComposeContent:  "services:\n  web:\n    image: nginx:alpine\n",
			OverrideContent: "services:\n  web:\n    image: busybox:latest\n",
		},
		Submitted: fullStackSourceSubmitted,
	})
	require.NoError(t, err)
	require.Equal(t, "services:\n  web:\n    image: busybox:latest\n", updated.OverrideContent)
	require.FileExists(t, overridePath)

	source, err := svc.GetStackSource(ctx, "0", "demo-stack")
	require.NoError(t, err)
	require.Equal(t, updated.OverrideContent, source.OverrideContent)
	// The override is a first-class field, never surfaced as an extra file.
	require.Empty(t, source.Files)

	// Clearing the override removes the file so a UI redeploy stops merging it.
	_, err = svc.UpdateStackSource(ctx, "0", "demo-stack", StackSourceUpdate{
		Source: types.SwarmStackSource{
			ComposeContent: "services:\n  web:\n    image: nginx:alpine\n",
		},
		Submitted: fullStackSourceSubmitted,
	})
	require.NoError(t, err)
	require.NoFileExists(t, overridePath)

	source, err = svc.GetStackSource(ctx, "0", "demo-stack")
	require.NoError(t, err)
	require.Empty(t, source.OverrideContent)
}

func TestService_UpdateStackSource_PrunesAndRestoresOnDeployFailure(t *testing.T) {
	ctx := t.Context()
	db := setupSwarmServiceTestDB(t)
	rootDir := t.TempDir()
	t.Setenv("ARCANE_SWARM_STACK_SOURCES_DIRECTORY", rootDir)

	settingsSvc, err := newSettingsServiceForSwarmTest(t, ctx, db)
	require.NoError(t, err)

	svc := NewService(ServiceParams{
		SettingsService: settingsSvc,
	})

	original := deployStackAfterSourceUpdate
	t.Cleanup(func() { deployStackAfterSourceUpdate = original })
	var lastReq types.SwarmStackDeployRequest
	var deployErr error
	deployStackAfterSourceUpdate = func(_ *Service, _ context.Context, _ string, req types.SwarmStackDeployRequest) (*types.SwarmStackDeployResponse, error) {
		lastReq = req
		if deployErr != nil {
			return nil, deployErr
		}
		return &types.SwarmStackDeployResponse{Name: req.Name}, nil
	}

	firstCompose := "services:\n  web:\n    image: nginx:alpine\n"
	_, err = svc.UpdateStackSource(ctx, "0", "demo-stack", StackSourceUpdate{
		Source: types.SwarmStackSource{
			ComposeContent: firstCompose,
		},
		Submitted: fullStackSourceSubmitted,
	})
	require.NoError(t, err)
	// The saved source is the full stack spec: services removed from it must
	// be pruned from the swarm on redeploy.
	require.True(t, lastReq.Prune)

	deployErr = errors.New("deploy failed")
	_, err = svc.UpdateStackSource(ctx, "0", "demo-stack", StackSourceUpdate{
		Source: types.SwarmStackSource{
			ComposeContent: "services:\n  web:\n    image: nginx:broken\n",
		},
		Submitted: fullStackSourceSubmitted,
	})
	require.ErrorContains(t, err, "deploy failed")

	// A failed deploy must not commit the edited source.
	source, err := svc.GetStackSource(ctx, "0", "demo-stack")
	require.NoError(t, err)
	require.Equal(t, firstCompose, source.ComposeContent)
}

func stacksScopeContext(t *testing.T, target string, bodies map[string]any) context.Context {
	t.Helper()
	return httpx.WithEnvironmentScope(t.Context(), &httpx.EnvironmentScope{
		IDs:     []string{"a", "b", "c"},
		Request: httptest.NewRequest(http.MethodGet, target, http.NoBody),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, ok := bodies[strings.Split(strings.TrimPrefix(r.URL.Path, "/api/agents/"), "/")[0]]
			if !ok {
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte(`{"detail":"environment offline"}`))
				return
			}
			_ = json.MarshalWrite(w, body)
		}),
		Resolve: func(_ context.Context, id string) (types.EnvironmentRef, error) {
			return types.EnvironmentRef{ID: id, Name: "Env " + id}, nil
		},
	})
}

func TestAggregateMergesEnvironments(t *testing.T) {
	ctx := stacksScopeContext(t, "/api/agents/a,b,c/swarm/stacks", map[string]any{
		"a": types.Paginated[types.SwarmStackSummary]{
			Success: true,
			Data:    []types.SwarmStackSummary{{ID: "web", Name: "web"}},
		},
		"b": types.Paginated[types.SwarmStackSummary]{
			Success: true,
			Data:    []types.SwarmStackSummary{{ID: "api", Name: "api"}},
		},
	})

	body, err := (&Service{}).Aggregate(ctx, pagination.QueryParams{Params: pagination.Params{Limit: 20}})
	require.NoError(t, err)

	require.Len(t, body.Data, 2)
	require.ElementsMatch(t, []string{"Env a", "Env b"}, []string{body.Data[0].Environment().Name, body.Data[1].Environment().Name})
	require.EqualValues(t, 2, body.Pagination.TotalItems)
	require.Len(t, body.FailedEnvironments, 1)
	require.Equal(t, "c", body.FailedEnvironments[0].Environment.ID)
}

func TestAggregateKeepsSourceOwningManagerPerCluster(t *testing.T) {
	ctx := stacksScopeContext(t, "/api/agents/a,b,c/swarm/stacks", map[string]any{
		"a": types.Paginated[types.SwarmStackSummary]{
			Success: true,
			Data: []types.SwarmStackSummary{
				{ID: "web", Name: "web", ClusterID: "cluster-1"},
				{ID: "api", Name: "api", ClusterID: "cluster-1"},
			},
		},
		"b": types.Paginated[types.SwarmStackSummary]{
			Success: true,
			Data: []types.SwarmStackSummary{
				{ID: "web", Name: "web", ClusterID: "cluster-1", HasSource: true},
				{ID: "api", Name: "api", ClusterID: "cluster-1"},
				{ID: "saved", Name: "saved", HasSource: true},
			},
		},
		"c": types.Paginated[types.SwarmStackSummary]{
			Success: true,
			Data:    []types.SwarmStackSummary{{ID: "web", Name: "web", ClusterID: "cluster-2"}},
		},
	})

	body, err := (&Service{}).Aggregate(ctx, pagination.QueryParams{Params: pagination.Params{Limit: 20}})
	require.NoError(t, err)

	rows := make([]string, 0, len(body.Data))
	for _, item := range body.Data {
		rows = append(rows, item.Environment().ID+"/"+item.Name)
	}
	require.ElementsMatch(t, []string{"a/api", "b/web", "b/saved", "c/web"}, rows)
	require.EqualValues(t, 4, body.Pagination.TotalItems)
}

func TestService_UpStack_DeploysSavedSource(t *testing.T) {
	ctx := t.Context()
	db := setupSwarmServiceTestDB(t)
	t.Setenv("ARCANE_SWARM_STACK_SOURCES_DIRECTORY", t.TempDir())

	settingsSvc, err := newSettingsServiceForSwarmTest(t, ctx, db)
	require.NoError(t, err)

	svc := NewService(ServiceParams{SettingsService: settingsSvc})
	deployRec := stubStackSourceUpdateDeploy(t)

	require.ErrorIs(t, svc.UpStack(ctx, "0", "demo-stack"), errdefs.ErrNotFound)
	require.Equal(t, 0, deployRec.calls)

	_, err = svc.UpdateStackSource(ctx, "0", "demo-stack", StackSourceUpdate{
		Source: types.SwarmStackSource{
			ComposeContent: "services:\n  web:\n    image: nginx:alpine\n",
			EnvContent:     "FOO=bar\n",
		},
		Submitted: fullStackSourceSubmitted,
	})
	require.NoError(t, err)

	require.NoError(t, svc.UpStack(ctx, "0", " demo-stack "))
	require.Equal(t, 2, deployRec.calls)
	require.Equal(t, "demo-stack", deployRec.lastReq.Name)
	require.Equal(t, "services:\n  web:\n    image: nginx:alpine\n", deployRec.lastReq.ComposeContent)
	require.Equal(t, "FOO=bar\n", deployRec.lastReq.EnvContent)
	require.True(t, deployRec.lastReq.WithRegistryAuth)
}

func TestService_DownStack_RequiresSavedSource(t *testing.T) {
	ctx := t.Context()
	db := setupSwarmServiceTestDB(t)
	t.Setenv("ARCANE_SWARM_STACK_SOURCES_DIRECTORY", t.TempDir())

	settingsSvc, err := newSettingsServiceForSwarmTest(t, ctx, db)
	require.NoError(t, err)

	svc := NewService(ServiceParams{SettingsService: settingsSvc})

	err = svc.DownStack(ctx, "0", "demo-stack")
	require.ErrorIs(t, err, common.ErrValidation)
	require.False(t, errdefs.IsNotFound(err))
}

func newDownStackTestService(t *testing.T, removed *[]string) *Service {
	t.Helper()
	ctx := t.Context()
	db := setupSwarmServiceTestDB(t)
	t.Setenv("ARCANE_SWARM_STACK_SOURCES_DIRECTORY", t.TempDir())

	settingsSvc, err := newSettingsServiceForSwarmTest(t, ctx, db)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/v1.41")
		switch {
		case r.Method == http.MethodGet && path == "/services":
			_, _ = w.Write([]byte(`[{"ID":"svc-1","Spec":{"Name":"demo-stack_web"}}]`))
		case r.Method == http.MethodDelete:
			*removed = append(*removed, path)
		case r.Method == http.MethodGet && slices.Contains([]string{"/tasks", "/networks", "/configs", "/secrets"}, path):
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	cli, err := client.New(client.WithHost(server.URL), client.WithAPIVersion("1.41"), client.WithHTTPClient(server.Client()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })

	return NewService(ServiceParams{
		DockerService:   dockerInternal.NewDockerClientService(ctx, db, settingsSvc, dockerInternal.ClientOptions{}).WithClient(cli),
		EnsureManager:   func(context.Context) error { return nil },
		SettingsService: settingsSvc,
	})
}

func TestService_DownStack_RemovesServicesAndKeepsSource(t *testing.T) {
	ctx := t.Context()
	var removed []string
	svc := newDownStackTestService(t, &removed)
	stubStackSourceUpdateDeploy(t)

	_, err := svc.UpdateStackSource(ctx, "0", "demo-stack", StackSourceUpdate{
		Source:    types.SwarmStackSource{ComposeContent: "services:\n  web:\n    image: nginx:alpine\n", EnvContent: "FOO=bar\n"},
		Submitted: fullStackSourceSubmitted,
	})
	require.NoError(t, err)

	require.NoError(t, svc.DownStack(ctx, "0", " demo-stack "))
	require.Equal(t, []string{"/services/svc-1"}, removed)

	source, err := svc.GetStackSource(ctx, "0", "demo-stack")
	require.NoError(t, err)
	require.Equal(t, "services:\n  web:\n    image: nginx:alpine\n", source.ComposeContent)

	// A deploy that supplies no variables of its own, such as a Git sync, keeps the saved .env.
	_, err = svc.DeployStack(ctx, "0", types.SwarmStackDeployRequest{Name: "demo-stack", ComposeContent: "services: {}\n", KeepSavedEnv: true})
	require.NoError(t, err)
	source, err = svc.GetStackSource(ctx, "0", "demo-stack")
	require.NoError(t, err)
	require.Equal(t, "FOO=bar\n", source.EnvContent)
}

func TestService_DownStack_RefusesWhenSavedSourceCannotRender(t *testing.T) {
	ctx := t.Context()
	var removed []string
	svc := newDownStackTestService(t, &removed)
	stubStackSourceUpdateDeploy(t)

	_, err := svc.UpdateStackSource(ctx, "0", "demo-stack", StackSourceUpdate{
		Source: types.SwarmStackSource{
			ComposeContent: "services:\n  web:\n    image: nginx:alpine\n    configs: [app]\nconfigs:\n  app:\n    file: ./app.conf\n",
		},
		Submitted: fullStackSourceSubmitted,
	})
	require.NoError(t, err)

	sharedPath := filepath.Join(svc.settingsService.GetStringSetting(ctx, "projectsDirectory"), ".env.global")
	require.NoError(t, os.WriteFile(sharedPath, []byte("# arcane:metadata 2\n# arcane:secret SECRET\nSECRET=owned-down-secret\n"), 0o600))
	err = svc.DownStack(ctx, "0", "demo-stack")
	require.ErrorIs(t, err, common.ErrValidation)
	require.NotContains(t, err.Error(), "owned-down-secret")
	require.Empty(t, removed)
}

func newSharedStackTestService(t *testing.T, db *database.DB, settingsSvc *settings.SettingsService) (*Service, *sharedStackDockerRecorder) {
	t.Helper()
	ctx := t.Context()
	recorded := &sharedStackDockerRecorder{service: new(swarm.ServiceSpec), config: new(swarm.ConfigSpec), secret: new(swarm.SecretSpec)}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded.mu.Lock()
		defer recorded.mu.Unlock()
		recorded.requests = append(recorded.requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		resource := strings.TrimPrefix(r.URL.Path, "/v1.41")
		switch {
		case r.Method == http.MethodGet &&
			(strings.HasPrefix(resource, "/networks/") || strings.HasPrefix(resource, "/configs/") || strings.HasPrefix(resource, "/secrets/")):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"owned resource not found"}`))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && resource == "/networks/create":
			_, _ = w.Write([]byte(`{"Id":"owned-network"}`))
		case r.Method == http.MethodPost && (resource == "/configs/create" || resource == "/secrets/create"):
			body, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				http.Error(w, "owned body read failed", 500)
				return
			}
			var destination any = recorded.config
			if resource == "/secrets/create" {
				destination = recorded.secret
			}
			if decodeErr := json.Unmarshal(body, destination); decodeErr != nil {
				http.Error(w, "owned body decode failed", 500)
				return
			}
			if resource == "/secrets/create" {
				_, _ = w.Write([]byte(`{"ID":"owned-secret"}`))
			} else {
				_, _ = w.Write([]byte(`{"ID":"owned-config"}`))
			}
		case r.Method == http.MethodPost && resource == "/services/create":
			body, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				http.Error(w, "owned body read failed", 500)
				return
			}
			if decodeErr := json.Unmarshal(body, recorded.service); decodeErr != nil {
				http.Error(w, "owned body decode failed", 500)
				return
			}
			_, _ = w.Write([]byte(`{"ID":"owned-service"}`))
		default:
			http.Error(w, "unexpected owned Docker request", 500)
		}
	}))
	t.Cleanup(server.Close)
	cli, err := client.New(client.WithHost(server.URL), client.WithAPIVersion("1.41"), client.WithHTTPClient(server.Client()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })

	service := NewService(ServiceParams{
		DockerService: dockerInternal.NewDockerClientService(ctx, db, settingsSvc, dockerInternal.ClientOptions{}).WithClient(cli),
		EnsureManager: func(context.Context) error { return nil }, SettingsService: settingsSvc,
	})
	return service, recorded
}

func TestService_SharedStackInputs(t *testing.T) {
	for _, tc := range []struct {
		name, shared, env, read, overridden string
		secret                              bool
	}{
		{name: "defaults and injection", shared: "global", read: "global", overridden: "default"},
		{name: "stack overrides and expansion", shared: "global", env: "SHARED=stack\nLOCAL=${SHARED}-expanded\nOVERRIDE=stack\n", read: "stack", overridden: "stack"},
		{name: "explicit blank overrides", shared: "global", env: "SHARED=\nOVERRIDE=\n", read: "", overridden: ""},
		{name: "protected preview", shared: "owned-secret-value", read: "owned-secret-value", overridden: "default", secret: true},
		{name: "literal values", shared: "literal $TOKEN ${UNKNOWN} \"quoted\" \\path\nnext", read: "literal $TOKEN ${UNKNOWN} \"quoted\" \\path\nnext", overridden: "default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			db := setupSwarmServiceTestDB(t)
			t.Setenv("ARCANE_SWARM_STACK_SOURCES_DIRECTORY", t.TempDir())
			settingsSvc, err := newSettingsServiceForSwarmTest(t, ctx, db)
			require.NoError(t, err)
			header := "# arcane:metadata 2\n# arcane:noinject SHARED\n"
			if tc.secret {
				header += "# arcane:secret SHARED\n"
			}
			sharedPath := filepath.Join(settingsSvc.GetStringSetting(ctx, "projectsDirectory"), ".env.global")
			globalContent := header + projects.FormatEnvMap(projects.EnvMap{
				"SHARED": tc.shared, "AUTO": "automatic", "INJECT": "global-injection", "OVERRIDE": "default", "EMPTY": "must-not-overwrite-empty",
			})
			require.NoError(t, os.WriteFile(sharedPath, []byte(globalContent), 0o600))
			svc, recorded := newSharedStackTestService(t, db, settingsSvc)
			compose := `services:
  app:
    image: busybox:1.37
    environment:
      READ: ${SHARED}
      INDIRECT: ${LOCAL:-none}
      INJECT: service-wins
      EMPTY: ""
    deploy:
      replicas: 0
`
			if tc.secret {
				compose += "    configs: [shared-config]\n    secrets: [shared-secret]\nconfigs:\n  shared-config:\n    environment: SHARED\nsecrets:\n  shared-secret:\n    environment: SHARED\n"
			}
			rendered, err := svc.RenderStackConfig(ctx, "0", types.SwarmStackRenderConfigRequest{Name: "owned-inputs", ComposeContent: compose, EnvContent: tc.env})
			require.NoError(t, err)
			require.Equal(t, tc.secret, rendered.PreviewWithheld)
			if tc.secret {
				require.Empty(t, rendered.RenderedCompose)
				require.Empty(t, rendered.Services)
				require.Empty(t, rendered.Networks)
				require.Empty(t, rendered.Volumes)
				require.Empty(t, rendered.Configs)
				require.Empty(t, rendered.Secrets)
				require.NotEmpty(t, rendered.Warnings)
				encoded, marshalErr := json.Marshal(rendered)
				require.NoError(t, marshalErr)
				require.NotContains(t, string(encoded), tc.shared)
			} else {
				require.Contains(t, rendered.RenderedCompose, "AUTO: automatic")
			}
			response, err := svc.DeployStack(ctx, "0", types.SwarmStackDeployRequest{Name: "owned-inputs", ComposeContent: compose, EnvContent: tc.env, ResolveImage: "never"})
			require.NoError(t, err)
			require.Equal(t, "owned-inputs", response.Name)
			recorded.mu.Lock()
			captured := slices.Clone(recorded.service.TaskTemplate.ContainerSpec.Env)
			configData := slices.Clone(recorded.config.Data)
			secretData := slices.Clone(recorded.secret.Data)
			configs := slices.Clone(recorded.service.TaskTemplate.ContainerSpec.Configs)
			secrets := slices.Clone(recorded.service.TaskTemplate.ContainerSpec.Secrets)
			recorded.mu.Unlock()
			if tc.secret {
				require.Equal(t, tc.shared, string(configData))
				require.Equal(t, tc.shared, string(secretData))
				require.Len(t, configs, 1)
				require.Len(t, secrets, 1)
				require.Equal(t, "owned-config", configs[0].ConfigID)
				require.Equal(t, "owned-secret", secrets[0].SecretID)
			}
			require.Contains(t, captured, "READ="+tc.read)
			require.Contains(t, captured, "AUTO=automatic")
			require.Contains(t, captured, "OVERRIDE="+tc.overridden)
			require.Contains(t, captured, "INJECT=service-wins")
			require.Contains(t, captured, "EMPTY=")
			require.False(t, slices.ContainsFunc(captured, func(value string) bool { return strings.HasPrefix(value, "SHARED=") }))
			if tc.name == "stack overrides and expansion" {
				require.Contains(t, captured, "INDIRECT=stack-expanded")
			}
			source, err := svc.GetStackSource(ctx, "0", "owned-inputs")
			require.NoError(t, err)
			require.Equal(t, compose, source.ComposeContent)
			require.Equal(t, tc.env, source.EnvContent)
			if tc.secret {
				sourceJSON, marshalErr := json.Marshal(source)
				require.NoError(t, marshalErr)
				require.NotContains(t, string(sourceJSON), tc.shared)
				invalidCompose := "services:\n  app:\n    image: busybox\n    deploy:\n      replicas: ${SHARED}\n"
				_, invalidErr := svc.RenderStackConfig(ctx, "0", types.SwarmStackRenderConfigRequest{
					Name: "owned-invalid", ComposeContent: invalidCompose,
				})
				require.Error(t, invalidErr)
				require.NotContains(t, invalidErr.Error(), tc.shared)
				require.Equal(t, 400, common.ToAPIError(invalidErr).HTTPStatus())
				require.Nil(t, common.ToAPIError(invalidErr).Details)
				invalidResponse, deployErr := svc.DeployStack(ctx, "0", types.SwarmStackDeployRequest{
					Name: "owned-invalid", ComposeContent: invalidCompose, ResolveImage: "never",
				})
				require.Nil(t, invalidResponse)
				require.Error(t, deployErr)
				require.NotContains(t, deployErr.Error(), tc.shared)
				require.Equal(t, 400, common.ToAPIError(deployErr).HTTPStatus())
				require.Nil(t, common.ToAPIError(deployErr).Details)
			}
			recorded.mu.Lock()
			recorded.requests = nil
			recorded.mu.Unlock()
			require.NoError(t, os.Remove(sharedPath))
			require.NoError(t, os.Mkdir(sharedPath, 0o700))
			failed, inputErr := svc.DeployStack(ctx, "0", types.SwarmStackDeployRequest{Name: "unreadable-inputs", ComposeContent: compose})
			require.ErrorIs(t, inputErr, common.ErrUnavailable)
			require.Nil(t, failed)
			recorded.mu.Lock()
			observed := slices.Clone(recorded.requests)
			recorded.mu.Unlock()
			require.Empty(t, observed)
			_, sourceErr := svc.GetStackSource(ctx, "0", "unreadable-inputs")
			require.Error(t, sourceErr)
		})
	}
}
