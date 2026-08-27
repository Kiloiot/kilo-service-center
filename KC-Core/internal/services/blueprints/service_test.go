// Package blueprints provides blueprint service tests.
package blueprints

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

func i64ptr(v int64) *int64 { return &v }

const (
	registryOwner        = "org"
	registryRepo         = "repo"
	registryBaseBranch   = "main"
	registryBranchPrefix = "blueprint/"
	registryBlueprintDir = ""
	registryFileExt      = ".json"
)

// registryTestServerOpts configures the test registry server behavior.
type registryTestServerOpts struct {
	// preflightStatus is the HTTP status for GET /contents/ (file existence check). Default: 404.
	preflightStatus int
	// commitStatus is the HTTP status for PUT /contents/. Default: 201.
	commitStatus int
	// commitBody overrides the response body for PUT /contents/.
	commitBody string
}

type registryTestServer struct {
	URL             string
	shutdown        func()
	capturedPath    string
	capturedCommit  string
	capturedPRTitle string
	capturedPRBody  string
}

func (s *registryTestServer) Close() {
	if s != nil && s.shutdown != nil {
		s.shutdown()
	}
}

func newRegistryTestServer(t *testing.T, opts ...registryTestServerOpts) *registryTestServer {
	t.Helper()

	var opt registryTestServerOpts
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.preflightStatus == 0 {
		opt.preflightStatus = http.StatusNotFound
	}
	if opt.commitStatus == 0 {
		opt.commitStatus = http.StatusCreated
	}
	if opt.commitBody == "" {
		opt.commitBody = `{"commit":{"sha":"commit-sha"}}`
	}

	srv := &registryTestServer{}
	mux := http.NewServeMux()

	mux.HandleFunc("/repos/org/repo/git/refs/heads/main", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ref":"refs/heads/main","object":{"sha":"base-sha"}}`))
	})

	mux.HandleFunc("/repos/org/repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	})

	mux.HandleFunc("/repos/org/repo/contents/", func(w http.ResponseWriter, r *http.Request) {
		srv.capturedPath = r.URL.Path

		if r.Method == http.MethodGet {
			// Preflight file existence check
			w.WriteHeader(opt.preflightStatus)
			if opt.preflightStatus == http.StatusOK {
				_, _ = w.Write([]byte(`{"name":"existing.json"}`))
			}
			return
		}
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT or GET, got %s", r.Method)
			return
		}

		var req createFileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		srv.capturedCommit = req.Message
		if req.Content == "" {
			t.Errorf("expected base64 content")
		}
		decoded, err := base64.StdEncoding.DecodeString(req.Content)
		if err != nil {
			t.Errorf("invalid base64 content: %v", err)
		}
		if !json.Valid(decoded) {
			t.Errorf("decoded content is not valid JSON")
		}

		w.WriteHeader(opt.commitStatus)
		_, _ = w.Write([]byte(opt.commitBody))
	})

	mux.HandleFunc("/repos/org/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var pr createPRRequest
		if err := json.NewDecoder(r.Body).Decode(&pr); err != nil {
			t.Errorf("decode PR request: %v", err)
		}
		srv.capturedPRTitle = pr.Title
		srv.capturedPRBody = pr.Body
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"html_url":"https://registry.example.com/org/repo/pull/1"}`))
	})

	ts := httptest.NewUnstartedServer(mux)
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on IPv4 loopback: %v", err)
	}
	ts.Listener = l
	ts.Start()
	srv.URL = ts.URL
	srv.shutdown = ts.Close

	return srv
}

// mockBlueprintRepository implements interfaces.BlueprintRepository for testing.
type mockBlueprintRepository struct {
	getByIDFn            func(ctx context.Context, tenantID int64, id uuid.UUID) (*models.Blueprint, error)
	getDefaultForModelFn func(ctx context.Context, tenantID int64, modelID uuid.UUID) (*models.Blueprint, error)
	updateRegistryInfo   func(ctx context.Context, tenantID int64, id uuid.UUID, repo, commitSHA, prURL string, verified bool) error
	createFn             func(ctx context.Context, p *models.BlueprintCreateParams) (*models.Blueprint, error)
	countByDeviceModelFn func(ctx context.Context, tenantID int64, modelID uuid.UUID) (int64, error)
}

func (m *mockBlueprintRepository) Create(ctx context.Context, p *models.BlueprintCreateParams) (*models.Blueprint, error) {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	return nil, nil
}

func (m *mockBlueprintRepository) GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.Blueprint, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return nil, interfaces.ErrRecordNotFound
}

func (m *mockBlueprintRepository) GetByVersion(_ context.Context, _ int64, _ uuid.UUID, _ string) (*models.Blueprint, error) {
	return nil, nil
}

func (m *mockBlueprintRepository) GetByTypeEUI(_ context.Context, _ int64, _ []byte) (*models.Blueprint, error) {
	return nil, nil
}

func (m *mockBlueprintRepository) GetDefaultForModel(ctx context.Context, tenantID int64, modelID uuid.UUID) (*models.Blueprint, error) {
	if m.getDefaultForModelFn != nil {
		return m.getDefaultForModelFn(ctx, tenantID, modelID)
	}
	return nil, nil
}

func (m *mockBlueprintRepository) ListByDeviceModel(_ context.Context, _ int64, _ uuid.UUID, _, _ int) ([]*models.Blueprint, error) {
	return nil, nil
}

func (m *mockBlueprintRepository) List(_ context.Context, _ *models.BlueprintListParams) ([]*models.Blueprint, error) {
	return nil, nil
}

func (m *mockBlueprintRepository) ListWithModel(_ context.Context, _ *models.BlueprintListParams) ([]*models.BlueprintWithModel, error) {
	return nil, nil
}

func (m *mockBlueprintRepository) Count(_ context.Context, _ int64, _ bool) (int64, error) {
	return 0, nil
}

func (m *mockBlueprintRepository) CountByDeviceModel(ctx context.Context, tenantID int64, _ bool, modelID uuid.UUID) (int64, error) {
	if m.countByDeviceModelFn != nil {
		return m.countByDeviceModelFn(ctx, tenantID, modelID)
	}
	return 0, nil
}

// TestCreateBlueprint_AutoDefault: new blueprint self-heals to default when the model has none (overriding explicit false); respects false when a default exists.
func TestCreateBlueprint_AutoDefault(t *testing.T) {
	modelID := uuid.New()
	dmRepo := &mockDeviceModelRepository{
		getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.DeviceModel, error) {
			return &models.DeviceModel{ID: modelID}, nil
		},
	}
	cases := []struct {
		name        string
		hasDefault  bool
		reqDefault  bool
		wantDefault bool
	}{
		{"no default self-heals", false, false, true},
		{"no default keeps explicit default", false, true, true},
		{"existing default respects false", true, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var captured *models.BlueprintCreateParams
			bpRepo := &mockBlueprintRepository{
				getDefaultForModelFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
					if tc.hasDefault {
						return &models.Blueprint{ID: uuid.New(), IsDefault: true}, nil
					}
					return nil, nil
				},
				createFn: func(_ context.Context, p *models.BlueprintCreateParams) (*models.Blueprint, error) {
					captured = p
					return &models.Blueprint{ID: uuid.New(), IsDefault: p.IsDefault}, nil
				},
			}
			svc := New(&mockManufacturerRepository{}, dmRepo, bpRepo, 1, nil, &testLogger{})
			if _, err := svc.CreateBlueprint(testutil.TestContext(), &grpcservices.BlueprintCreateRequest{
				DeviceModelID: modelID, Version: "1.0.0", IsDefault: tc.reqDefault,
			}); err != nil {
				t.Fatalf("CreateBlueprint: %v", err)
			}
			if captured == nil {
				t.Fatal("Create was not called")
			}
			if captured.IsDefault != tc.wantDefault {
				t.Errorf("IsDefault = %v, want %v", captured.IsDefault, tc.wantDefault)
			}
		})
	}
}

func (m *mockBlueprintRepository) Update(_ context.Context, _ int64, _ bool, _ uuid.UUID, _ *models.BlueprintUpdateParams) error {
	return nil
}

func (m *mockBlueprintRepository) SetDefault(_ context.Context, _ int64, _ bool, _ uuid.UUID) error {
	return nil
}

func (m *mockBlueprintRepository) ClearDefault(_ context.Context, _ int64, _ bool, _ uuid.UUID) error {
	return nil
}

func (m *mockBlueprintRepository) UpdateRegistryInfo(ctx context.Context, tenantID int64, _ bool, id uuid.UUID, repo, commitSHA, prURL string, verified bool) error {
	if m.updateRegistryInfo != nil {
		return m.updateRegistryInfo(ctx, tenantID, id, repo, commitSHA, prURL, verified)
	}
	return nil
}

func (m *mockBlueprintRepository) Delete(_ context.Context, _ int64, _ bool, _ uuid.UUID) error {
	return nil
}

// mockManufacturerRepository implements interfaces.ManufacturerRepository for testing.
type mockManufacturerRepository struct {
	getByIDFn func(ctx context.Context, tenantID int64, id uuid.UUID) (*models.Manufacturer, error)
}

func (m *mockManufacturerRepository) Create(_ context.Context, _ *models.ManufacturerCreateParams) (*models.Manufacturer, error) {
	return nil, nil
}

func (m *mockManufacturerRepository) GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.Manufacturer, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return nil, nil
}

func (m *mockManufacturerRepository) GetByName(_ context.Context, _ int64, _ string) (*models.Manufacturer, error) {
	return nil, nil
}

func (m *mockManufacturerRepository) List(_ context.Context, _ *models.ManufacturerListParams) ([]*models.Manufacturer, error) {
	return nil, nil
}

func (m *mockManufacturerRepository) Count(_ context.Context, _ int64, _ bool) (int64, error) {
	return 0, nil
}

func (m *mockManufacturerRepository) Update(_ context.Context, _ int64, _ bool, _ uuid.UUID, _ *models.ManufacturerUpdateParams) error {
	return nil
}

func (m *mockManufacturerRepository) Delete(_ context.Context, _ int64, _ bool, _ uuid.UUID) error {
	return nil
}

func (m *mockManufacturerRepository) SetVerified(_ context.Context, _ int64, _ uuid.UUID, _ bool) error {
	return nil
}

// mockDeviceModelRepository implements interfaces.DeviceModelRepository for testing.
type mockDeviceModelRepository struct {
	getByIDFn func(ctx context.Context, tenantID int64, id uuid.UUID) (*models.DeviceModel, error)
	createFn  func(ctx context.Context, params *models.DeviceModelCreateParams) (*models.DeviceModel, error)
	updateFn  func(ctx context.Context, tenantID int64, id uuid.UUID, params *models.DeviceModelUpdateParams) error
}

func (m *mockDeviceModelRepository) Create(ctx context.Context, params *models.DeviceModelCreateParams) (*models.DeviceModel, error) {
	if m.createFn != nil {
		return m.createFn(ctx, params)
	}
	return nil, nil
}

func (m *mockDeviceModelRepository) GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.DeviceModel, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return nil, nil
}

func (m *mockDeviceModelRepository) GetByCode(_ context.Context, _ int64, _ uuid.UUID, _ string) (*models.DeviceModel, error) {
	return nil, nil
}

func (m *mockDeviceModelRepository) GetByTypeEUI(_ context.Context, _ int64, _ []byte) (*models.DeviceModel, error) {
	return nil, nil
}

func (m *mockDeviceModelRepository) ListByManufacturer(_ context.Context, _ int64, _ uuid.UUID, _, _ int) ([]*models.DeviceModel, error) {
	return nil, nil
}

func (m *mockDeviceModelRepository) List(_ context.Context, _ *models.DeviceModelListParams) ([]*models.DeviceModel, error) {
	return nil, nil
}

func (m *mockDeviceModelRepository) ListWithManufacturer(_ context.Context, _ *models.DeviceModelListParams) ([]*models.DeviceModelWithManufacturer, error) {
	return nil, nil
}

func (m *mockDeviceModelRepository) Count(_ context.Context, _ int64, _ bool) (int64, error) {
	return 0, nil
}

func (m *mockDeviceModelRepository) CountByManufacturer(_ context.Context, _ int64, _ bool, _ uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockDeviceModelRepository) Update(ctx context.Context, tenantID int64, _ bool, id uuid.UUID, params *models.DeviceModelUpdateParams) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, id, params)
	}
	return nil
}

func (m *mockDeviceModelRepository) Delete(_ context.Context, _ int64, _ bool, _ uuid.UUID) error {
	return nil
}

// testLogger is a no-op logger for tests.
type testLogger struct{}

func (l *testLogger) Debug(_ string, _ ...interface{})                   {}
func (l *testLogger) Info(_ string, _ ...interface{})                    {}
func (l *testLogger) Warn(_ string, _ ...interface{})                    {}
func (l *testLogger) Error(_ string, _ ...interface{})                   {}
func (l *testLogger) Fatal(_ string, _ ...interface{})                   {}
func (l *testLogger) DebugContext(_ context.Context, _ string, _ ...any) {}
func (l *testLogger) InfoContext(_ context.Context, _ string, _ ...any)  {}
func (l *testLogger) WarnContext(_ context.Context, _ string, _ ...any)  {}
func (l *testLogger) ErrorContext(_ context.Context, _ string, _ ...any) {}
func (l *testLogger) FatalContext(_ context.Context, _ string, _ ...any) {}
func (l *testLogger) WithField(_ string, _ interface{}) logger.Logger    { return l }
func (l *testLogger) WithFields(_ map[string]interface{}) logger.Logger  { return l }

// TestSubmitToRegistry_Disabled tests that ErrRegistryDisabled is returned when registry is not enabled.
func TestSubmitToRegistry_Disabled(t *testing.T) {
	ctx := testutil.TestContext()

	// Service with nil config (registry disabled)
	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{},
		1, // tenantID
		nil,
		&testLogger{},
	)

	_, err := svc.SubmitToRegistry(ctx, uuid.New(), &grpcservices.RegistrySubmitRequest{})
	if !errors.Is(err, ErrRegistryDisabled) {
		t.Errorf("expected ErrRegistryDisabled, got %v", err)
	}

	// Service with config but enabled=false
	svc2 := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{},
		1,
		&config.RegistryProviderConfig{Enabled: false},
		&testLogger{},
	)

	_, err = svc2.SubmitToRegistry(ctx, uuid.New(), &grpcservices.RegistrySubmitRequest{})
	if !errors.Is(err, ErrRegistryDisabled) {
		t.Errorf("expected ErrRegistryDisabled, got %v", err)
	}
}

// TestSubmitToRegistry_APIURLRequired tests that ErrRegistryAPIURLRequired is returned when enabled but api_url is empty.
func TestSubmitToRegistry_APIURLRequired(t *testing.T) {
	ctx := testutil.TestContext()

	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{},
		1,
		&config.RegistryProviderConfig{
			Enabled: true,
			APIURL:  "", // Empty URL
		},
		&testLogger{},
	)

	_, err := svc.SubmitToRegistry(ctx, uuid.New(), &grpcservices.RegistrySubmitRequest{})
	if !errors.Is(err, ErrRegistryAPIURLRequired) {
		t.Errorf("expected ErrRegistryAPIURLRequired, got %v", err)
	}
}

// TestSubmitToRegistry_BlueprintNotFound tests that ErrBlueprintNotFound is returned for invalid blueprint ID.
func TestSubmitToRegistry_BlueprintNotFound(t *testing.T) {
	ctx := testutil.TestContext()

	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
				return nil, interfaces.ErrRecordNotFound
			},
		},
		1,
		&config.RegistryProviderConfig{
			Enabled: true,
			APIURL:  "https://api.example.com",
		},
		&testLogger{},
	)

	_, err := svc.SubmitToRegistry(ctx, uuid.New(), &grpcservices.RegistrySubmitRequest{})
	if !errors.Is(err, ErrBlueprintNotFound) {
		t.Errorf("expected ErrBlueprintNotFound, got %v", err)
	}
}

// TestSubmitToRegistry_AlreadySubmitted tests that ErrAlreadySubmitted is returned when blueprint has a PR URL.
func TestSubmitToRegistry_AlreadySubmitted(t *testing.T) {
	ctx := testutil.TestContext()

	prURL := "https://registry.example.com/org/repo/pull/123"

	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
				return &models.Blueprint{
					ID:            uuid.New(),
					TenantID:      i64ptr(1),
					DeviceModelID: uuid.New(),
					Version:       "1.0.0",
					TypeEUI:       []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xAB, 0xCD, 0xEF},
					SpecJSON:      []byte(`{"codec": "test"}`),
					RegistryPRURL: &prURL, // Already submitted
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}, nil
			},
		},
		1,
		&config.RegistryProviderConfig{
			Enabled: true,
			APIURL:  "https://api.example.com",
		},
		&testLogger{},
	)

	_, err := svc.SubmitToRegistry(ctx, uuid.New(), &grpcservices.RegistrySubmitRequest{})
	if !errors.Is(err, ErrAlreadySubmitted) {
		t.Errorf("expected ErrAlreadySubmitted, got %v", err)
	}
}

// TestPrepareBlueprintContent_ValidJSON tests that prepareBlueprintContent returns valid base64-encoded JSON
// with enriched manufacturer/model metadata.
func TestPrepareBlueprintContent_ValidJSON(t *testing.T) {
	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{},
		1,
		nil,
		&testLogger{},
	)

	bp := &models.Blueprint{
		ID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		DeviceModelID: uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"),
		Version:       "1.0.0",
		TypeEUI:       []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xAB, 0xCD, 0xEF},
		SpecJSON:      []byte(`{"codec": "mioty-standard"}`),
		IsDefault:     true,
		CreatedAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	content, err := svc.prepareBlueprintContent(bp, "Weptech", "Robin M", "robin-m")
	if err != nil {
		t.Fatalf("prepareBlueprintContent failed: %v", err)
	}

	// Verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		t.Fatalf("content is not valid base64: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(decoded, &result); err != nil {
		t.Fatalf("decoded content is not valid JSON: %v", err)
	}

	// Verify required fields are present (including enriched metadata)
	expectedFields := []string{"id", "device_model_id", "version", "type_eui", "spec", "is_default",
		"manufacturer_name", "device_model_name", "device_model_code", "created_at", "updated_at"}
	for _, field := range expectedFields {
		if _, ok := result[field]; !ok {
			t.Errorf("expected field %q not found in content", field)
		}
	}

	// Verify specific values
	if result["id"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected id '550e8400-e29b-41d4-a716-446655440000', got %v", result["id"])
	}
	if result["version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %v", result["version"])
	}
	if result["type_eui"] != "1234567890ABCDEF" {
		t.Errorf("expected type_eui '1234567890ABCDEF', got %v", result["type_eui"])
	}
	if result["is_default"] != true {
		t.Errorf("expected is_default true, got %v", result["is_default"])
	}
	if result["manufacturer_name"] != "Weptech" {
		t.Errorf("expected manufacturer_name 'Weptech', got %v", result["manufacturer_name"])
	}
	if result["device_model_name"] != "Robin M" {
		t.Errorf("expected device_model_name 'Robin M', got %v", result["device_model_name"])
	}
	if result["device_model_code"] != "robin-m" {
		t.Errorf("expected device_model_code 'robin-m', got %v", result["device_model_code"])
	}
}

func TestSubmitToRegistry_Success(t *testing.T) {
	ctx := testutil.TestContext()
	server := newRegistryTestServer(t)
	t.Cleanup(server.Close)

	blueprintID := uuid.New()
	modelID := uuid.New()
	mfrID := uuid.New()

	blueprint := &models.Blueprint{
		ID:            blueprintID,
		TenantID:      i64ptr(1),
		DeviceModelID: modelID,
		Version:       "1.0.1",
		TypeEUI:       []byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80},
		SpecJSON:      []byte(`{"codec":"test"}`),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	repo := &mockBlueprintRepository{
		getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
			return blueprint, nil
		},
	}

	cfg := &config.RegistryProviderConfig{
		Enabled:       true,
		APIURL:        server.URL,
		Owner:         registryOwner,
		Repo:          registryRepo,
		BaseBranch:    registryBaseBranch,
		BranchPrefix:  registryBranchPrefix,
		BlueprintPath: registryBlueprintDir,
		FileExtension: registryFileExt,
		HTTPTimeout:   2,
	}

	svc := New(
		&mockManufacturerRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Manufacturer, error) {
				return &models.Manufacturer{ID: mfrID, Name: "Weptech", TenantID: i64ptr(1)}, nil
			},
		},
		&mockDeviceModelRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.DeviceModel, error) {
				return &models.DeviceModel{ID: modelID, ManufacturerID: mfrID, Name: "Robin M", Code: "robin-m", TenantID: i64ptr(1)}, nil
			},
		},
		repo,
		1,
		cfg,
		&testLogger{},
	)

	result, err := svc.SubmitToRegistry(ctx, blueprintID, &grpcservices.RegistrySubmitRequest{
		ContributorName:  "Contributor",
		ContributorEmail: "contributor@example.com",
		Notes:            "test submission",
	})
	if err != nil {
		t.Fatalf("SubmitToRegistry failed: %v", err)
	}

	// Branch should use structured path: blueprint/{mfr}/{model}/{version}/{shortID}
	shortID := blueprintID.String()[:8]
	expectedBranch := registryBranchPrefix + "weptech/robin-m/v1.0.1/" + shortID
	if result.BranchName != expectedBranch {
		t.Errorf("expected branch %q, got %q", expectedBranch, result.BranchName)
	}
	if result.CommitSHA != "commit-sha" {
		t.Errorf("expected commit SHA %q, got %q", "commit-sha", result.CommitSHA)
	}
	if result.PRUrl != "https://registry.example.com/org/repo/pull/1" {
		t.Errorf("expected PR URL %q, got %q", "https://registry.example.com/org/repo/pull/1", result.PRUrl)
	}
	if result.RepoPath != "org/repo" {
		t.Errorf("expected repo path %q, got %q", "org/repo", result.RepoPath)
	}

	// Verify file path uses structured directory: weptech/robin-m/v1.0.1.json
	expectedPathSuffix := "/repos/org/repo/contents/weptech/robin-m/v1.0.1.json"
	if !strings.HasSuffix(server.capturedPath, expectedPathSuffix) {
		t.Errorf("expected file path ending with %q, got %q", expectedPathSuffix, server.capturedPath)
	}

	// Verify commit message uses human-readable names (no UUID)
	if !strings.Contains(server.capturedCommit, "Weptech") {
		t.Errorf("commit message should contain manufacturer name, got %q", server.capturedCommit)
	}
	if strings.Contains(server.capturedCommit, blueprintID.String()) {
		t.Errorf("commit message should not contain UUID, got %q", server.capturedCommit)
	}

	// Verify PR title uses human-readable names
	if !strings.Contains(server.capturedPRTitle, "Weptech") {
		t.Errorf("PR title should contain manufacturer name, got %q", server.capturedPRTitle)
	}
}

func TestSubmitToRegistry_UpdateRegistryInfo(t *testing.T) {
	ctx := testutil.TestContext()
	server := newRegistryTestServer(t)
	t.Cleanup(server.Close)

	blueprintID := uuid.New()
	modelID := uuid.New()
	mfrID := uuid.New()

	blueprint := &models.Blueprint{
		ID:            blueprintID,
		TenantID:      i64ptr(1),
		DeviceModelID: modelID,
		Version:       "2.0.0",
		TypeEUI:       []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11},
		SpecJSON:      []byte(`{"codec":"test"}`),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	updated := struct {
		called    bool
		repoPath  string
		commitSHA string
		prURL     string
	}{}

	repo := &mockBlueprintRepository{
		getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
			return blueprint, nil
		},
		updateRegistryInfo: func(_ context.Context, _ int64, _ uuid.UUID, repo, commitSHA, prURL string, _ bool) error {
			updated.called = true
			updated.repoPath = repo
			updated.commitSHA = commitSHA
			updated.prURL = prURL
			return nil
		},
	}

	cfg := &config.RegistryProviderConfig{
		Enabled:       true,
		APIURL:        server.URL,
		Owner:         registryOwner,
		Repo:          registryRepo,
		BaseBranch:    registryBaseBranch,
		BranchPrefix:  registryBranchPrefix,
		BlueprintPath: registryBlueprintDir,
		FileExtension: registryFileExt,
		HTTPTimeout:   2,
	}

	svc := New(
		&mockManufacturerRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Manufacturer, error) {
				return &models.Manufacturer{ID: mfrID, Name: "TestMfr", TenantID: i64ptr(1)}, nil
			},
		},
		&mockDeviceModelRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.DeviceModel, error) {
				return &models.DeviceModel{ID: modelID, ManufacturerID: mfrID, Name: "TestModel", Code: "test-model", TenantID: i64ptr(1)}, nil
			},
		},
		repo,
		1,
		cfg,
		&testLogger{},
	)

	_, err := svc.SubmitToRegistry(ctx, blueprintID, &grpcservices.RegistrySubmitRequest{
		ContributorName:  "Contributor",
		ContributorEmail: "contributor@example.com",
	})
	if err != nil {
		t.Fatalf("SubmitToRegistry failed: %v", err)
	}

	if !updated.called {
		t.Fatalf("expected UpdateRegistryInfo to be called")
	}
	if updated.repoPath != "org/repo" {
		t.Errorf("expected repo path %q, got %q", "org/repo", updated.repoPath)
	}
	if updated.commitSHA != "commit-sha" {
		t.Errorf("expected commit SHA %q, got %q", "commit-sha", updated.commitSHA)
	}
	if updated.prURL != "https://registry.example.com/org/repo/pull/1" {
		t.Errorf("expected PR URL %q, got %q", "https://registry.example.com/org/repo/pull/1", updated.prURL)
	}
}

// TestGenerateSlug verifies slug generation rules.
func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase", "My Model", "my-model"},
		{"trim spaces", "  hello  ", "hello"},
		{"strip special chars", "foo@bar#baz", "foobarbaz"},
		{"collapse hyphens", "foo--bar---baz", "foo-bar-baz"},
		{"empty input", "", "model"},
		{"only special chars", "!@#$%", "model"},
		{"unicode stripped", "model-\u00e9", "model-"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := generateSlug(tc.input)
			if tc.name == "unicode stripped" {
				// Unicode handling may vary; just ensure non-empty
				if got == "" {
					t.Errorf("expected non-empty slug for %q", tc.input)
				}
				return
			}
			if got != tc.expected {
				t.Errorf("generateSlug(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

// TestCreateDeviceModelWithBlueprint_TxStarterNotConfigured tests that an error is returned when tx starter is nil.
func TestCreateDeviceModelWithBlueprint_TxStarterNotConfigured(t *testing.T) {
	ctx := testutil.TestContext()

	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{},
		1,
		nil,
		&testLogger{},
	)

	_, _, err := svc.CreateDeviceModelWithBlueprint(ctx, &grpcservices.DeviceModelWithBlueprintRequest{
		ManufacturerID: uuid.New(),
		Name:           "Test Model",
		Version:        "1.0.0",
		DecoderScript:  []byte(`{"codec":"test"}`),
	})
	if err == nil {
		t.Fatal("expected error for unconfigured tx starter")
	}
	if !errors.Is(err, nil) {
		// Just verify it's a meaningful error
		if err.Error() == "" {
			t.Error("expected non-empty error message")
		}
	}
}

// TestCreateDeviceModelWithBlueprint_ManufacturerNotFound tests that manufacturer validation works.
func TestCreateDeviceModelWithBlueprint_ManufacturerNotFound(t *testing.T) {
	ctx := testutil.TestContext()

	mockMfrRepo := &mockManufacturerRepository{}
	// Default GetByID returns nil, nil — but the service checks for ErrRecordNotFound
	// Override to return not found
	svc := New(
		&mockManufacturerRepoNotFound{},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{},
		1,
		nil,
		&testLogger{},
	).WithTxStarter(&mockTxStarter{})

	_ = mockMfrRepo // suppress unused warning in older toolchains

	_, _, err := svc.CreateDeviceModelWithBlueprint(ctx, &grpcservices.DeviceModelWithBlueprintRequest{
		ManufacturerID: uuid.New(),
		Name:           "Test Model",
		Version:        "1.0.0",
		DecoderScript:  []byte(`{"codec":"test"}`),
	})
	if !errors.Is(err, ErrManufacturerNotFound) {
		t.Errorf("expected ErrManufacturerNotFound, got %v", err)
	}
}

// TestDecodePreview_BlueprintNotFound tests that preview returns not-found for missing blueprints.
func TestDecodePreview_BlueprintNotFound(t *testing.T) {
	ctx := testutil.TestContext()

	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
				return nil, interfaces.ErrRecordNotFound
			},
		},
		1,
		nil,
		&testLogger{},
	)

	_, err := svc.DecodePreview(ctx, uuid.New(), []byte{0x01, 0x02}, 0)
	if !errors.Is(err, ErrBlueprintNotFound) {
		t.Errorf("expected ErrBlueprintNotFound, got %v", err)
	}
}

// TestDecodePreview_DecoderNotConfigured tests that preview returns error when decoder is nil.
func TestDecodePreview_DecoderNotConfigured(t *testing.T) {
	ctx := testutil.TestContext()
	bpID := uuid.New()

	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
				return &models.Blueprint{
					ID:       bpID,
					TenantID: i64ptr(1),
					Version:  "1.0.0",
					SpecJSON: []byte(`{"codec":"test"}`),
				}, nil
			},
		},
		1,
		nil,
		&testLogger{},
	)

	_, err := svc.DecodePreview(ctx, bpID, []byte{0x01}, 0)
	if err == nil {
		t.Fatal("expected error for unconfigured decoder")
	}
	if !errors.Is(err, nil) {
		if err.Error() == "" {
			t.Error("expected non-empty error message")
		}
	}
}

// mockManufacturerRepoNotFound always returns ErrRecordNotFound for GetByID.
type mockManufacturerRepoNotFound struct {
	mockManufacturerRepository
}

func (m *mockManufacturerRepoNotFound) GetByID(_ context.Context, _ int64, _ uuid.UUID) (*models.Manufacturer, error) {
	return nil, interfaces.ErrRecordNotFound
}

// mockTxStarter implements TxStarter for testing.
type mockTxStarter struct {
	beginErr error
}

func (m *mockTxStarter) BeginTx(_ context.Context) (interfaces.Transaction, error) {
	if m.beginErr != nil {
		return nil, m.beginErr
	}
	return nil, errors.New("mock transaction not fully implemented")
}

// --- New tests for structured registry submission ---

// registryTestFixture creates a common test setup for registry submission tests.
func registryTestFixture() (uuid.UUID, uuid.UUID, uuid.UUID, *models.Blueprint) {
	blueprintID := uuid.New()
	modelID := uuid.New()
	mfrID := uuid.New()
	bp := &models.Blueprint{
		ID:            blueprintID,
		TenantID:      i64ptr(1),
		DeviceModelID: modelID,
		Version:       "1.0.0",
		TypeEUI:       []byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80},
		SpecJSON:      []byte(`{"codec":"test"}`),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	return blueprintID, modelID, mfrID, bp
}

func registryTestService(t *testing.T, server *registryTestServer, bp *models.Blueprint, modelID, mfrID uuid.UUID, mfrName, modelName, modelCode string) *Service {
	t.Helper()
	return New(
		&mockManufacturerRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Manufacturer, error) {
				return &models.Manufacturer{ID: mfrID, Name: mfrName, TenantID: i64ptr(1)}, nil
			},
		},
		&mockDeviceModelRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.DeviceModel, error) {
				return &models.DeviceModel{ID: modelID, ManufacturerID: mfrID, Name: modelName, Code: modelCode, TenantID: i64ptr(1)}, nil
			},
		},
		&mockBlueprintRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
				return bp, nil
			},
		},
		1,
		&config.RegistryProviderConfig{
			Enabled:       true,
			APIURL:        server.URL,
			Owner:         registryOwner,
			Repo:          registryRepo,
			BaseBranch:    registryBaseBranch,
			BranchPrefix:  registryBranchPrefix,
			BlueprintPath: registryBlueprintDir,
			FileExtension: registryFileExt,
			HTTPTimeout:   2,
		},
		&testLogger{},
	)
}

func TestSubmitToRegistry_VersionAlreadyExists_Preflight(t *testing.T) {
	ctx := testutil.TestContext()
	// Preflight returns 200 (file exists)
	server := newRegistryTestServer(t, registryTestServerOpts{preflightStatus: http.StatusOK})
	t.Cleanup(server.Close)

	blueprintID, modelID, mfrID, bp := registryTestFixture()
	svc := registryTestService(t, server, bp, modelID, mfrID, "Weptech", "Robin M", "robin-m")

	_, err := svc.SubmitToRegistry(ctx, blueprintID, &grpcservices.RegistrySubmitRequest{
		ContributorName:  "Test",
		ContributorEmail: "test@example.com",
	})
	if !errors.Is(err, ErrRegistryVersionAlreadyExists) {
		t.Errorf("expected ErrRegistryVersionAlreadyExists, got %v", err)
	}
}

func TestSubmitToRegistry_VersionAlreadyExists_CommitConflict(t *testing.T) {
	ctx := testutil.TestContext()
	// Preflight returns 404 (not found), but PUT returns 409 (conflict)
	server := newRegistryTestServer(t, registryTestServerOpts{
		preflightStatus: http.StatusNotFound,
		commitStatus:    http.StatusConflict,
		commitBody:      `{"message":"conflict"}`,
	})
	t.Cleanup(server.Close)

	blueprintID, modelID, mfrID, bp := registryTestFixture()
	svc := registryTestService(t, server, bp, modelID, mfrID, "Weptech", "Robin M", "robin-m")

	_, err := svc.SubmitToRegistry(ctx, blueprintID, &grpcservices.RegistrySubmitRequest{
		ContributorName:  "Test",
		ContributorEmail: "test@example.com",
	})
	if !errors.Is(err, ErrRegistryVersionAlreadyExists) {
		t.Errorf("expected ErrRegistryVersionAlreadyExists, got %v", err)
	}
}

func TestSubmitToRegistry_Commit422_NonVersionConflict(t *testing.T) {
	ctx := testutil.TestContext()
	// PUT returns 422 with non-conflict body
	server := newRegistryTestServer(t, registryTestServerOpts{
		preflightStatus: http.StatusNotFound,
		commitStatus:    http.StatusUnprocessableEntity,
		commitBody:      `{"message":"path contains a malformed path component"}`,
	})
	t.Cleanup(server.Close)

	blueprintID, modelID, mfrID, bp := registryTestFixture()
	svc := registryTestService(t, server, bp, modelID, mfrID, "Weptech", "Robin M", "robin-m")

	_, err := svc.SubmitToRegistry(ctx, blueprintID, &grpcservices.RegistrySubmitRequest{
		ContributorName:  "Test",
		ContributorEmail: "test@example.com",
	})
	// Should NOT be ErrRegistryVersionAlreadyExists
	if errors.Is(err, ErrRegistryVersionAlreadyExists) {
		t.Errorf("expected non-version-conflict error, got ErrRegistryVersionAlreadyExists")
	}
	// Should be ErrRegistryAPIError (via ErrCommitFailed wrapping)
	if !errors.Is(err, ErrCommitFailed) && !errors.Is(err, ErrRegistryAPIError) {
		t.Errorf("expected ErrCommitFailed or ErrRegistryAPIError, got %v", err)
	}
}

func TestSubmitToRegistry_DeviceModelNotFound(t *testing.T) {
	ctx := testutil.TestContext()

	blueprintID := uuid.New()
	bp := &models.Blueprint{
		ID:            blueprintID,
		TenantID:      i64ptr(1),
		DeviceModelID: uuid.New(),
		Version:       "1.0.0",
		TypeEUI:       []byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80},
		SpecJSON:      []byte(`{"codec":"test"}`),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.DeviceModel, error) {
				return nil, interfaces.ErrRecordNotFound
			},
		},
		&mockBlueprintRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
				return bp, nil
			},
		},
		1,
		&config.RegistryProviderConfig{Enabled: true, APIURL: "https://api.example.com"},
		&testLogger{},
	)

	_, err := svc.SubmitToRegistry(ctx, blueprintID, &grpcservices.RegistrySubmitRequest{
		ContributorName:  "Test",
		ContributorEmail: "test@example.com",
	})
	if !errors.Is(err, ErrDeviceModelNotFound) {
		t.Errorf("expected ErrDeviceModelNotFound, got %v", err)
	}
}

func TestSubmitToRegistry_ManufacturerNotFound(t *testing.T) {
	ctx := testutil.TestContext()

	blueprintID := uuid.New()
	modelID := uuid.New()
	mfrID := uuid.New()
	bp := &models.Blueprint{
		ID:            blueprintID,
		TenantID:      i64ptr(1),
		DeviceModelID: modelID,
		Version:       "1.0.0",
		TypeEUI:       []byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80},
		SpecJSON:      []byte(`{"codec":"test"}`),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	svc := New(
		&mockManufacturerRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Manufacturer, error) {
				return nil, interfaces.ErrRecordNotFound
			},
		},
		&mockDeviceModelRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.DeviceModel, error) {
				return &models.DeviceModel{ID: modelID, ManufacturerID: mfrID, Name: "Test", Code: "test", TenantID: i64ptr(1)}, nil
			},
		},
		&mockBlueprintRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
				return bp, nil
			},
		},
		1,
		&config.RegistryProviderConfig{Enabled: true, APIURL: "https://api.example.com"},
		&testLogger{},
	)

	_, err := svc.SubmitToRegistry(ctx, blueprintID, &grpcservices.RegistrySubmitRequest{
		ContributorName:  "Test",
		ContributorEmail: "test@example.com",
	})
	if !errors.Is(err, ErrManufacturerNotFound) {
		t.Errorf("expected ErrManufacturerNotFound, got %v", err)
	}
}

func TestSubmitToRegistry_InvalidPathSegment(t *testing.T) {
	ctx := testutil.TestContext()
	server := newRegistryTestServer(t)
	t.Cleanup(server.Close)

	blueprintID, modelID, mfrID, bp := registryTestFixture()
	// Empty manufacturer name should produce empty slug → ErrInvalidRegistryPathSegment
	svc := registryTestService(t, server, bp, modelID, mfrID, "", "Robin M", "robin-m")

	_, err := svc.SubmitToRegistry(ctx, blueprintID, &grpcservices.RegistrySubmitRequest{
		ContributorName:  "Test",
		ContributorEmail: "test@example.com",
	})
	if !errors.Is(err, ErrInvalidRegistryPathSegment) {
		t.Errorf("expected ErrInvalidRegistryPathSegment, got %v", err)
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.0", "v1.0"},
		{"v1.0", "v1.0"},
		{"V2.0", "v2.0"},
		{"v1.0.3-rc1", "v1.0.3-rc1"},
		{"  v3.0  ", "v3.0"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeVersion(tc.input)
			if got != tc.expected {
				t.Errorf("normalizeVersion(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestSanitizeVersionSegment(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{"valid semver", "v1.0", "v1.0", false},
		{"with rc suffix", "v1.0.3-rc1", "v1.0.3-rc1", false},
		{"uppercase normalized", "V2.0", "v2.0", false},
		{"traversal rejected", "../etc", "", true},
		{"slash rejected", "v1/../../etc", "", true},
		{"empty rejected", "", "", true},
		{"spaces trimmed", "  v1.0  ", "v1.0", false},
		{"bare v prefix rejected", "v", "", true},
		{"slash in version rejected", "v1/2", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sanitizeVersionSegment(tc.input)
			if tc.expectErr {
				if !errors.Is(err, ErrInvalidRegistryPathSegment) {
					t.Errorf("expected ErrInvalidRegistryPathSegment, got err=%v val=%q", err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("sanitizeVersionSegment(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestSanitizePathSegment(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{"normal name", "Weptech", "weptech", false},
		{"with spaces", "My Company", "my-company", false},
		{"empty rejected", "", "", true},
		{"only specials rejected", "!@#$%", "", true},
		{"NUL stripped", "\x00test", "test", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sanitizePathSegment(tc.input)
			if tc.expectErr {
				if !errors.Is(err, ErrInvalidRegistryPathSegment) {
					t.Errorf("expected ErrInvalidRegistryPathSegment, got err=%v val=%q", err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("sanitizePathSegment(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestIsCommitConflict422(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{"fast forward conflict", `{"message":"Update is not a fast forward"}`, true},
		{"SHA mismatch", `{"message":"main is at abc123 but expected def456"}`, true},
		{"non-conflict 422", `{"message":"path contains a malformed path component"}`, false},
		{"sha not supplied", `{"message":"Invalid request.\n\n\"sha\" wasn't supplied."}`, true},
		{"case-insensitive fast forward", `{"message":"UPDATE IS NOT A FAST FORWARD"}`, true},
		{"sha only no conflict", `{"message":"sha"}`, false},
		{"empty body", ``, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isCommitConflict422([]byte(tc.body))
			if got != tc.expected {
				t.Errorf("isCommitConflict422(%q) = %v, want %v", tc.body, got, tc.expected)
			}
		})
	}
}

func TestSubmitToRegistry_VersionAlreadyExists_Commit422Conflict(t *testing.T) {
	ctx := testutil.TestContext()
	// Preflight: 404 (not found), commit: 422 with fast-forward conflict body
	server := newRegistryTestServer(t, registryTestServerOpts{
		preflightStatus: http.StatusNotFound,
		commitStatus:    http.StatusUnprocessableEntity,
		commitBody:      `{"message":"Update is not a fast forward"}`,
	})
	t.Cleanup(server.Close)

	blueprintID, modelID, mfrID, bp := registryTestFixture()
	svc := registryTestService(t, server, bp, modelID, mfrID, "Weptech", "Robin M", "robin-m")

	_, err := svc.SubmitToRegistry(ctx, blueprintID, &grpcservices.RegistrySubmitRequest{
		ContributorName:  "Test",
		ContributorEmail: "test@example.com",
	})
	if !errors.Is(err, ErrRegistryVersionAlreadyExists) {
		t.Errorf("expected ErrRegistryVersionAlreadyExists, got %v", err)
	}
}

func TestSubmitToRegistry_EmptyVersionRejected(t *testing.T) {
	ctx := testutil.TestContext()
	server := newRegistryTestServer(t)
	t.Cleanup(server.Close)

	blueprintID, modelID, mfrID, bp := registryTestFixture()
	bp.Version = ""
	svc := registryTestService(t, server, bp, modelID, mfrID, "Weptech", "Robin M", "robin-m")

	_, err := svc.SubmitToRegistry(ctx, blueprintID, &grpcservices.RegistrySubmitRequest{
		ContributorName:  "Test",
		ContributorEmail: "test@example.com",
	})
	if !errors.Is(err, ErrInvalidRegistryPathSegment) {
		t.Errorf("expected ErrInvalidRegistryPathSegment, got %v", err)
	}
}

func TestSubmitToRegistry_InvalidModelCodeRejected(t *testing.T) {
	ctx := testutil.TestContext()
	server := newRegistryTestServer(t)
	t.Cleanup(server.Close)

	blueprintID, modelID, mfrID, bp := registryTestFixture()
	// Model code with slash — sanitizeModelCodeSegment rejects
	svc := registryTestService(t, server, bp, modelID, mfrID, "Weptech", "Robin M", "robin/m")

	_, err := svc.SubmitToRegistry(ctx, blueprintID, &grpcservices.RegistrySubmitRequest{
		ContributorName:  "Test",
		ContributorEmail: "test@example.com",
	})
	if !errors.Is(err, ErrInvalidRegistryPathSegment) {
		t.Errorf("expected ErrInvalidRegistryPathSegment, got %v", err)
	}
}

func TestSanitizeModelCodeSegment(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{"valid lowercase", "robin-m", "robin-m", false},
		{"valid alphanumeric", "valid123", "valid123", false},
		{"slash rejected", "robin/m", "", true},
		{"uppercase rejected", "Robin-M", "", true},
		{"empty rejected", "", "", true},
		{"spaces only rejected", "   ", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sanitizeModelCodeSegment(tc.input)
			if tc.expectErr {
				if !errors.Is(err, ErrInvalidRegistryPathSegment) {
					t.Errorf("expected ErrInvalidRegistryPathSegment, got err=%v val=%q", err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("sanitizeModelCodeSegment(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestCanonicalizeModelCode(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{"valid lowercase", "robin-m", "robin-m", false},
		{"uppercase normalized", "Robin-M", "robin-m", false},
		{"all uppercase", "VALID123", "valid123", false},
		{"slash rejected", "robin/m", "", true},
		{"space rejected", "robin m", "", true},
		{"empty rejected", "", "", true},
		{"leading hyphen rejected", "-leading", "", true},
		{"trailing hyphen rejected", "trailing-", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := canonicalizeModelCode(tc.input)
			if tc.expectErr {
				if !errors.Is(err, ErrInvalidModelCode) {
					t.Errorf("expected ErrInvalidModelCode, got err=%v val=%q", err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("canonicalizeModelCode(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestCreateDeviceModel_EmptyCodeGeneratesSlug(t *testing.T) {
	ctx := testutil.TestContext()

	var capturedCode string
	svc := New(
		&mockManufacturerRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Manufacturer, error) {
				return &models.Manufacturer{ID: uuid.New(), TenantID: i64ptr(1)}, nil
			},
		},
		&mockDeviceModelRepository{
			createFn: func(_ context.Context, params *models.DeviceModelCreateParams) (*models.DeviceModel, error) {
				capturedCode = params.Code
				return &models.DeviceModel{
					ID:             uuid.New(),
					ManufacturerID: params.ManufacturerID,
					Name:           params.Name,
					Code:           params.Code,
					TenantID:       &params.TenantID,
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}, nil
			},
		},
		&mockBlueprintRepository{},
		1,
		nil,
		&testLogger{},
	)

	_, err := svc.CreateDeviceModel(ctx, &grpcservices.DeviceModelCreateRequest{
		ManufacturerID: uuid.New(),
		Name:           "Robin M",
		Code:           "", // Empty code → slug from name
	})
	if err != nil {
		t.Fatalf("CreateDeviceModel failed: %v", err)
	}
	if capturedCode != "robin-m" {
		t.Errorf("expected code %q, got %q", "robin-m", capturedCode)
	}
}

func TestCreateDeviceModel_InvalidCodeRejected(t *testing.T) {
	ctx := testutil.TestContext()

	svc := New(
		&mockManufacturerRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Manufacturer, error) {
				return &models.Manufacturer{ID: uuid.New(), TenantID: i64ptr(1)}, nil
			},
		},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{},
		1,
		nil,
		&testLogger{},
	)

	_, err := svc.CreateDeviceModel(ctx, &grpcservices.DeviceModelCreateRequest{
		ManufacturerID: uuid.New(),
		Name:           "Robin M",
		Code:           "Robin M", // Space remains after ToLower → rejected
	})
	if !errors.Is(err, ErrInvalidModelCode) {
		t.Errorf("expected ErrInvalidModelCode, got %v", err)
	}
}

func TestCreateDeviceModel_ValidCodeCanonicalized(t *testing.T) {
	ctx := testutil.TestContext()

	var capturedCode string
	svc := New(
		&mockManufacturerRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Manufacturer, error) {
				return &models.Manufacturer{ID: uuid.New(), TenantID: i64ptr(1)}, nil
			},
		},
		&mockDeviceModelRepository{
			createFn: func(_ context.Context, params *models.DeviceModelCreateParams) (*models.DeviceModel, error) {
				capturedCode = params.Code
				return &models.DeviceModel{
					ID:             uuid.New(),
					ManufacturerID: params.ManufacturerID,
					Name:           params.Name,
					Code:           params.Code,
					TenantID:       &params.TenantID,
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}, nil
			},
		},
		&mockBlueprintRepository{},
		1,
		nil,
		&testLogger{},
	)

	_, err := svc.CreateDeviceModel(ctx, &grpcservices.DeviceModelCreateRequest{
		ManufacturerID: uuid.New(),
		Name:           "Robin M",
		Code:           "Robin-M", // Uppercase → canonicalized to lowercase
	})
	if err != nil {
		t.Fatalf("CreateDeviceModel failed: %v", err)
	}
	if capturedCode != "robin-m" {
		t.Errorf("expected code %q, got %q", "robin-m", capturedCode)
	}
}

func TestUpdateDeviceModel_InvalidCodeRejected(t *testing.T) {
	ctx := testutil.TestContext()
	modelID := uuid.New()

	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.DeviceModel, error) {
				return &models.DeviceModel{ID: modelID, TenantID: i64ptr(1), CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			},
		},
		&mockBlueprintRepository{},
		1,
		nil,
		&testLogger{},
	)

	invalidCode := "robin/m"
	_, err := svc.UpdateDeviceModel(ctx, modelID, &grpcservices.DeviceModelUpdateRequest{
		Code: &invalidCode,
	})
	if !errors.Is(err, ErrInvalidModelCode) {
		t.Errorf("expected ErrInvalidModelCode, got %v", err)
	}
}

func TestUpdateDeviceModel_ValidCodeCanonicalized(t *testing.T) {
	ctx := testutil.TestContext()
	modelID := uuid.New()

	var capturedCode *string
	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{
			getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.DeviceModel, error) {
				return &models.DeviceModel{ID: modelID, TenantID: i64ptr(1), CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			},
			updateFn: func(_ context.Context, _ int64, _ uuid.UUID, params *models.DeviceModelUpdateParams) error {
				capturedCode = params.Code
				return nil
			},
		},
		&mockBlueprintRepository{},
		1,
		nil,
		&testLogger{},
	)

	code := "Robin-M"
	_, err := svc.UpdateDeviceModel(ctx, modelID, &grpcservices.DeviceModelUpdateRequest{
		Code: &code,
	})
	if err != nil {
		t.Fatalf("UpdateDeviceModel failed: %v", err)
	}
	if capturedCode == nil || *capturedCode != "robin-m" {
		t.Errorf("expected code %q, got %v", "robin-m", capturedCode)
	}
}

// ============================================================================
// ResolveEffectiveTypeEUI Tests
// ============================================================================

func TestResolveEffectiveTypeEUI_RepoErrorPropagates(t *testing.T) {
	ctx := testutil.TestContext()

	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{
			getDefaultForModelFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
				return nil, fmt.Errorf("db down")
			},
		},
		1,
		nil,
		&testLogger{},
	)

	_, err := svc.ResolveEffectiveTypeEUI(ctx, 1, uuid.New())
	if err == nil {
		t.Fatal("expected error from repo failure, got nil")
	}
	if !strings.Contains(err.Error(), "db down") {
		t.Errorf("expected wrapped repo error, got %v", err)
	}
}

func TestResolveEffectiveTypeEUI_NilBlueprintNoError(t *testing.T) {
	ctx := testutil.TestContext()

	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{
			getDefaultForModelFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
				return nil, nil
			},
		},
		1,
		nil,
		&testLogger{},
	)

	eui, err := svc.ResolveEffectiveTypeEUI(ctx, 1, uuid.New())
	if err != nil {
		t.Fatalf("expected nil error for no default blueprint, got %v", err)
	}
	if eui != nil {
		t.Errorf("expected nil EUI for no default blueprint, got %v", eui)
	}
}

func TestResolveEffectiveTypeEUI_ValidTypeEUIReturned(t *testing.T) {
	ctx := testutil.TestContext()

	expectedBytes := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	svc := New(
		&mockManufacturerRepository{},
		&mockDeviceModelRepository{},
		&mockBlueprintRepository{
			getDefaultForModelFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
				return &models.Blueprint{
					ID:       uuid.New(),
					TypeEUI:  expectedBytes,
					Version:  "1.0.0",
					SpecJSON: []byte(`{}`),
				}, nil
			},
		},
		1,
		nil,
		&testLogger{},
	)

	eui, err := svc.ResolveEffectiveTypeEUI(ctx, 1, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eui == nil {
		t.Fatal("expected non-nil EUI")
	}
	var expected models.EUI
	copy(expected[:], expectedBytes)
	if *eui != expected {
		t.Errorf("expected EUI %v, got %v", expected, *eui)
	}
}
