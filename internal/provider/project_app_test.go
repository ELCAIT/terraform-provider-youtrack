// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testProjectAppID       = "151-12"
	testProjectAppNewID    = "151-99"
	testProjectAppName     = "@jetbrains/youtrack-workflow-due-date"
	testProjectAppUsageID  = "152-1"
	testProjectAppNewUsage = "152-new"
	testProjectAppsPath    = "/api/admin/apps"
	testProjectAppOtherPID = "0-1"
	testProjectAppPlainApp = "my-workflow"
)

// fakeAppsAPI is an in-memory apps API. An app ID missing from usages answers
// 404, like an app that is not installed.
type fakeAppsAPI struct {
	apps          []youtrack.App
	usages        map[string][]youtrack.AppUsage
	attachEnabled bool   // enabled state YouTrack gives a freshly attached usage
	attachAs      string // project ID YouTrack reports on attach, when it differs from the one sent
	failEnabled   bool   // answer 500 to a change of a usage's enabled state
	writes        []string
}

func (f *fakeAppsAPI) handler(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.Method != http.MethodGet {
			f.writes = append(f.writes, req.Method+" "+req.URL.Path)
		}

		if req.URL.Path == testProjectAppsPath {
			writeJSONPage(t, w, req, f.apps)
			return
		}

		parts := strings.Split(strings.TrimPrefix(req.URL.Path, testProjectAppsPath+"/"), "/")
		usages, installed := f.usages[parts[0]]
		if !installed || len(parts) < 2 {
			http.NotFound(w, req)
			return
		}

		if len(parts) == 2 {
			f.serveUsages(t, w, req, parts[0], usages)
			return
		}
		f.serveUsage(t, w, req, parts[0], parts[2])
	}
}

func (f *fakeAppsAPI) serveUsages(t *testing.T, w http.ResponseWriter, req *http.Request, appID string, usages []youtrack.AppUsage) {
	t.Helper()

	if req.Method == http.MethodGet {
		writeJSONPage(t, w, req, usages)
		return
	}

	var payload struct {
		Project youtrack.Project `json:"project"`
	}
	decodeJSON(t, req, &payload)

	projectID := payload.Project.ID
	if f.attachAs != "" {
		projectID = f.attachAs
	}
	created := youtrack.AppUsage{ID: testProjectAppNewUsage, Enabled: f.attachEnabled, Project: &youtrack.Project{ID: projectID}}
	f.usages[appID] = append(usages, created)
	writeJSON(t, w, created)
}

func (f *fakeAppsAPI) serveUsage(t *testing.T, w http.ResponseWriter, req *http.Request, appID, usageID string) {
	t.Helper()

	usages := f.usages[appID]
	for i := range usages {
		if usages[i].ID != usageID {
			continue
		}
		if req.Method == http.MethodDelete {
			f.usages[appID] = append(usages[:i], usages[i+1:]...)
			return
		}
		if f.failEnabled {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		var payload struct {
			Enabled bool `json:"enabled"`
		}
		decodeJSON(t, req, &payload)
		usages[i].Enabled = payload.Enabled
		writeJSON(t, w, usages[i])
		return
	}
	http.NotFound(w, req)
}

// writeJSONPage answers the first page with items and every later page with an
// empty list, which is all the paging the tests need.
func writeJSONPage[T any](t *testing.T, w http.ResponseWriter, req *http.Request, items []T) {
	t.Helper()

	if skip := req.URL.Query().Get("$skip"); skip != "" && skip != "0" {
		items = nil
	}
	if items == nil {
		items = []T{}
	}
	writeJSON(t, w, items)
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()

	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("failed to encode response: %v", err)
	}
}

func decodeJSON(t *testing.T, req *http.Request, v any) {
	t.Helper()

	if err := json.NewDecoder(req.Body).Decode(v); err != nil {
		t.Errorf("failed to decode request body: %v", err)
	}
}

func newProjectAppTestResource(t *testing.T, api *fakeAppsAPI) *projectAppResource {
	t.Helper()

	server := httptest.NewServer(api.handler(t))
	t.Cleanup(server.Close)

	client, err := youtrack.NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	return &projectAppResource{client: client}
}

func testProjectAppUsage(enabled bool) youtrack.AppUsage {
	return youtrack.AppUsage{ID: testProjectAppUsageID, Enabled: enabled, Project: &youtrack.Project{ID: testProjectID}}
}

func installedTestApp(usages ...youtrack.AppUsage) *fakeAppsAPI {
	return &fakeAppsAPI{
		apps:   []youtrack.App{{ID: testProjectAppID, Name: testProjectAppName}},
		usages: map[string][]youtrack.AppUsage{testProjectAppID: usages},
	}
}

func testProjectAppModel(appID string, enabled bool) projectAppResourceModel {
	return projectAppResourceModel{
		ProjectID: types.StringValue(testProjectID),
		AppName:   types.StringValue(testProjectAppName),
		AppID:     types.StringValue(appID),
		Enabled:   types.BoolValue(enabled),
	}
}

func TestProjectAppApply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		api         *fakeAppsAPI
		usageID     string // usage ID in the plan, empty on create
		appID       string // app ID in the plan, empty on create
		enabled     bool
		wantErr     bool
		wantUsageID string
		wantWrites  int
		wantNoUsage bool // the app must end up not attached
	}{
		{
			name:        "attaches and enables an app that is not attached",
			api:         installedTestApp(),
			enabled:     true,
			wantUsageID: testProjectAppNewUsage,
			wantWrites:  2, // attach, then enable: a fresh usage starts disabled here
		},
		{
			name:        "adopts an enabled usage without writing",
			api:         installedTestApp(testProjectAppUsage(true)),
			enabled:     true,
			wantUsageID: testProjectAppUsageID,
		},
		{
			name:        "enables a disabled usage in place",
			api:         installedTestApp(testProjectAppUsage(false)),
			enabled:     true,
			wantUsageID: testProjectAppUsageID,
			wantWrites:  1,
		},
		{
			name:        "disables an enabled usage in place",
			api:         installedTestApp(testProjectAppUsage(true)),
			enabled:     false,
			wantUsageID: testProjectAppUsageID,
			wantWrites:  1,
		},
		{
			name:        "attaches an app disabled",
			api:         func() *fakeAppsAPI { api := installedTestApp(); api.attachEnabled = true; return api }(),
			enabled:     false,
			wantUsageID: testProjectAppNewUsage,
			wantWrites:  2,
		},
		{
			name: "detaches the app again when disabling it after the attach fails",
			api: func() *fakeAppsAPI {
				api := installedTestApp()
				api.attachEnabled = true
				api.failEnabled = true
				return api
			}(),
			enabled:     false,
			wantErr:     true,
			wantWrites:  3, // attach, failed disable, detach
			wantNoUsage: true,
		},
		{
			name:    "fails when the app is not installed",
			api:     &fakeAppsAPI{usages: map[string][]youtrack.AppUsage{}},
			enabled: true,
			wantErr: true,
		},
		{
			name: "fails when YouTrack attaches to another project than project_id",
			api: func() *fakeAppsAPI {
				api := installedTestApp()
				api.attachEnabled = true
				api.attachAs = testProjectAppOtherPID
				return api
			}(),
			enabled:    true,
			wantErr:    true,
			wantWrites: 1,
		},
		{
			name:        "updates the usage in the plan",
			api:         installedTestApp(testProjectAppUsage(true)),
			usageID:     testProjectAppUsageID,
			appID:       testProjectAppID,
			enabled:     false,
			wantUsageID: testProjectAppUsageID,
			wantWrites:  1,
		},
		{
			name:    "fails without writing when the usage was deleted after the plan",
			api:     installedTestApp(),
			usageID: testProjectAppUsageID,
			appID:   testProjectAppID,
			enabled: false,
			wantErr: true,
		},
		{
			name:    "fails without writing when the app was reinstalled after the plan",
			api:     installedTestApp(testProjectAppUsage(true)),
			usageID: testProjectAppUsageID,
			appID:   testProjectAppNewID,
			enabled: false,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := newProjectAppTestResource(t, tc.api)
			model := testProjectAppModel(tc.appID, tc.enabled)
			if tc.usageID != "" {
				model.ID = types.StringValue(tc.usageID)
			}

			err := r.apply(context.Background(), &model)
			if (err != nil) != tc.wantErr {
				t.Fatalf("apply() error = %v, wantErr %v", err, tc.wantErr)
			}
			helpers.AssertFieldEqual(t, "writes", len(tc.api.writes), tc.wantWrites)
			if tc.wantNoUsage {
				helpers.AssertFieldEqual(t, "usages", len(tc.api.usages[testProjectAppID]), 0)
			}
			if tc.wantErr {
				return
			}
			helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), tc.wantUsageID)
			helpers.AssertFieldEqual(t, "AppID", model.AppID.ValueString(), testProjectAppID)
			helpers.AssertFieldEqual(t, "Enabled", model.Enabled.ValueBool(), tc.enabled)
			helpers.AssertFieldEqual(t, "AppName", model.AppName.ValueString(), testProjectAppName)
		})
	}
}

func TestProjectAppFindUsage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		api          *fakeAppsAPI
		stateAppID   string
		wantNotFound bool
		wantAppID    string
		wantUsage    bool
	}{
		{
			name:       "reads the usage through the app ID in state",
			api:        installedTestApp(testProjectAppUsage(true)),
			stateAppID: testProjectAppID,
			wantAppID:  testProjectAppID,
			wantUsage:  true,
		},
		{
			name:       "finds the app by name after an import",
			api:        installedTestApp(testProjectAppUsage(true)),
			stateAppID: "",
			wantAppID:  testProjectAppID,
			wantUsage:  true,
		},
		{
			name:       "falls back to the name when the app was reinstalled under a new ID",
			api:        installedTestApp(testProjectAppUsage(true)),
			stateAppID: testProjectAppNewID,
			wantAppID:  testProjectAppID,
			wantUsage:  true,
		},
		{
			name:       "reports no usage when the app was detached",
			api:        installedTestApp(),
			stateAppID: testProjectAppID,
			wantAppID:  testProjectAppID,
		},
		{
			name:         "reports not found when the app is no longer installed",
			api:          &fakeAppsAPI{usages: map[string][]youtrack.AppUsage{}},
			stateAppID:   testProjectAppID,
			wantNotFound: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := newProjectAppTestResource(t, tc.api)
			appID, usage, err := r.findUsage(context.Background(), testProjectAppModel(tc.stateAppID, true))

			if tc.wantNotFound {
				if !youtrack.IsNotFound(err) {
					t.Fatalf("findUsage() error = %v, want a not found error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("findUsage() unexpected error: %v", err)
			}
			helpers.AssertFieldEqual(t, "appID", appID, tc.wantAppID)
			helpers.AssertFieldEqual(t, "usage found", usage != nil, tc.wantUsage)
		})
	}
}
