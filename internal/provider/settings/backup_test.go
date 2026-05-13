package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testUsersPath         = "/api/users"
	testUserIDOne         = "u-1"
	testUserIDTwo         = "u-2"
	testCronExpressionTwo = "0 30 1 * * ?"
	testArchiveFormatOne  = "ZIP"
	testArchiveFormatTwo  = "TAR_GZ"
	testBackupID          = "backup-1"
	testBackupIDTwo       = "backup-2"
)

type backupSettingsTestCase struct {
	name           string
	id             string
	location       string
	filesToKeep    int64
	cronExpression string
	archiveFormat  string
	enabled        bool
	notifiedUsers  []string
}

var testBackupNotifiedUsers = []string{"admin", "ops.user"}
var commonBackupSettingsTestCases = []backupSettingsTestCase{
	{
		name:           "converts all fields correctly with two users",
		id:             testBackupID,
		location:       TestBackupPath,
		filesToKeep:    FilesToKeep,
		cronExpression: TestCronExpressionOne,
		archiveFormat:  testArchiveFormatOne,
		enabled:        true,
		notifiedUsers:  testBackupNotifiedUsers,
	},
	{
		name:           "converts all fields correctly with no users",
		id:             testBackupIDTwo,
		location:       "/mnt/backup",
		filesToKeep:    3,
		cronExpression: testCronExpressionTwo,
		archiveFormat:  testArchiveFormatTwo,
		enabled:        false,
		notifiedUsers:  []string{},
	},
}

func makeBackupModel(tc backupSettingsTestCase) (backupSettingsResourceModel, error) {
	ctx := context.Background()
	notifiedUsers, diags := types.ListValueFrom(ctx, types.StringType, tc.notifiedUsers)
	if diags.HasError() {
		return backupSettingsResourceModel{}, fmt.Errorf("failed to create test model: %v", diags)
	}

	return backupSettingsResourceModel{
		ID:             types.StringValue(tc.id),
		Location:       types.StringValue(tc.location),
		FilesToKeep:    types.Int64Value(tc.filesToKeep),
		CronExpression: types.StringValue(tc.cronExpression),
		ArchiveFormat:  types.StringValue(tc.archiveFormat),
		Enabled:        types.BoolValue(tc.enabled),
		NotifiedUsers:  notifiedUsers,
	}, nil
}

func makeBackupSettings(tc backupSettingsTestCase) youtrack.BackupSettings {
	users := make([]youtrack.User, 0, len(tc.notifiedUsers))
	for _, login := range tc.notifiedUsers {
		users = append(users, youtrack.User{Login: login})
	}

	return youtrack.BackupSettings{
		ID:             tc.id,
		Location:       tc.location,
		FilesToKeep:    int(tc.filesToKeep),
		CronExpression: tc.cronExpression,
		ArchiveFormat:  tc.archiveFormat,
		Enabled:        tc.enabled,
		NotifiedUsers:  users,
	}
}

func TestConvertBackupSettingsToModel(t *testing.T) {
	for _, tt := range commonBackupSettingsTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			backupSettings := makeBackupSettings(tt)

			got := convertBackupSettingsToModel(ctx, backupSettings)
			if got == nil {
				t.Fatal("convertBackupSettingsToModel returned nil")
			}

			helpers.AssertFieldEqual(t, "ID", got.ID.ValueString(), tt.id)
			helpers.AssertFieldEqual(t, "Location", got.Location.ValueString(), tt.location)
			helpers.AssertFieldEqual(t, "FilesToKeep", got.FilesToKeep.ValueInt64(), tt.filesToKeep)
			helpers.AssertFieldEqual(t, "CronExpression", got.CronExpression.ValueString(), tt.cronExpression)
			helpers.AssertFieldEqual(t, "ArchiveFormat", got.ArchiveFormat.ValueString(), tt.archiveFormat)
			helpers.AssertFieldEqual(t, "Enabled", got.Enabled.ValueBool(), tt.enabled)

			var gotUsers []string
			if diags := got.NotifiedUsers.ElementsAs(ctx, &gotUsers, false); diags.HasError() {
				t.Fatalf("failed to convert notified users list: %v", diags)
			}
			helpers.AssertFieldEqual(t, "NotifiedUsers length", len(gotUsers), len(tt.notifiedUsers))
			for i, login := range gotUsers {
				helpers.AssertFieldEqual(t, "NotifiedUsers item", login, tt.notifiedUsers[i])
			}
		})
	}
}

func TestUpdateBackupSettingsModelWithTimestamp(t *testing.T) {
	ctx := context.Background()
	backupSettings := makeBackupSettings(commonBackupSettingsTestCases[0])

	var model backupSettingsResourceModel
	ok := updateBackupSettingsModelWithTimestamp(ctx, backupSettings, &model)
	if !ok {
		t.Fatal("updateBackupSettingsModelWithTimestamp failed")
	}

	if model.LastUpdated.IsNull() {
		t.Fatal("LastUpdated should not be null")
	}
	if model.LastUpdated.ValueString() == "" {
		t.Fatal("LastUpdated should not be empty")
	}
}

func TestConvertModelToBackupSettingsResolvesLoginsToUserIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != testUsersPath {
			http.NotFound(w, r)
			return
		}

		if err := json.NewEncoder(w).Encode([]map[string]string{
			{"id": testUserIDOne, "login": "admin", "name": "Admin"},
			{"id": testUserIDTwo, "login": "ops.user", "name": "Operations"},
		}); err != nil {
			t.Fatalf("failed to encode users response: %v", err)
		}
	}))
	defer server.Close()

	client, err := youtrack.NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	r := &backupSettingsResource{client: client}
	plan, err := makeBackupModel(backupSettingsTestCase{
		id:             testBackupID,
		location:       TestBackupPath,
		filesToKeep:    FilesToKeep,
		cronExpression: TestCronExpressionOne,
		archiveFormat:  testArchiveFormatOne,
		enabled:        true,
		notifiedUsers:  testBackupNotifiedUsers,
	})
	if err != nil {
		t.Fatalf("failed to create backup plan: %v", err)
	}

	var diagnostics diag.Diagnostics
	got, ok := r.convertModelToBackupSettings(context.Background(), plan, &diagnostics)
	if !ok {
		t.Fatalf("convertModelToBackupSettings failed: %v", diagnostics)
	}
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}

	helpers.AssertFieldEqual(t, "NotifiedUsers length", len(got.NotifiedUsers), 2)
	helpers.AssertFieldEqual(t, "NotifiedUsers[0].ID", got.NotifiedUsers[0].ID, testUserIDOne)
	helpers.AssertFieldEqual(t, "NotifiedUsers[1].ID", got.NotifiedUsers[1].ID, testUserIDTwo)
	helpers.AssertFieldEqual(t, "ArchiveFormat", got.ArchiveFormat, TestBackupFormatZIP)
	helpers.AssertFieldEqual(t, "FilesToKeep", got.FilesToKeep, FilesToKeep)
}

func TestConvertModelToBackupSettingsUnknownLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != testUsersPath {
			http.NotFound(w, r)
			return
		}

		if err := json.NewEncoder(w).Encode([]map[string]string{
			{"id": testUserIDOne, "login": "admin", "name": "Admin"},
		}); err != nil {
			t.Fatalf("failed to encode users response: %v", err)
		}
	}))
	defer server.Close()

	client, err := youtrack.NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	r := &backupSettingsResource{client: client}
	plan, err := makeBackupModel(backupSettingsTestCase{
		id:             testBackupID,
		location:       TestBackupPath,
		filesToKeep:    FilesToKeep,
		cronExpression: TestCronExpressionOne,
		archiveFormat:  testArchiveFormatOne,
		enabled:        true,
		notifiedUsers:  []string{"missing.user"},
	})
	if err != nil {
		t.Fatalf("failed to create backup plan: %v", err)
	}

	var diagnostics diag.Diagnostics
	_, ok := r.convertModelToBackupSettings(context.Background(), plan, &diagnostics)
	if ok {
		t.Fatal("expected conversion to fail for unknown login")
	}
	if !diagnostics.HasError() {
		t.Fatal("expected diagnostics error for unknown login")
	}
}
