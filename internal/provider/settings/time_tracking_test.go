package settings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testWorkTimeID      = "64-0"
	testMinutesADay     = int64(480)
	testFirstDayOfWeek  = int64(1)
	testDaysAWeek       = int64(5)
	testWorkItemTypeID  = "65-0"
	testWorkItemType    = "Development"
	testAttributeID     = "66-0"
	testAttributeName   = "Type"
	testAttributeValue  = "Internal"
	testDescriptionText = "Internal activity"

	msgExpectedConversionToSucceed = "expected conversion to succeed"
	msgUnexpectedMinutesADay       = "unexpected minutes_a_day: got %d want %d"

	workItemTypesTestPath = "/api/admin/timeTrackingSettings/workItemTypes"
)

func makeTestWorkDaysList(t *testing.T, days []int64) types.List {
	t.Helper()

	list, diags := types.ListValueFrom(context.Background(), types.Int64Type, days)
	if diags.HasError() {
		t.Fatalf("failed to build work days list: %v", diags)
	}

	return list
}

func TestConvertModelToWorkTimeSettings(t *testing.T) {
	t.Parallel()

	model := globalTimeTrackingSettingsResourceModel{
		WorkTimeSettings: globalWorkTimeSettingsModel{
			ID:          types.StringValue(testWorkTimeID),
			MinutesADay: types.Int64Value(testMinutesADay),
			WorkDays:    makeTestWorkDaysList(t, []int64{1, 2, 3, 4, 5}),
		},
	}

	settings, ok := convertModelToWorkTimeSettings(model)
	if !ok {
		t.Fatal(msgExpectedConversionToSucceed)
	}

	if settings.ID != testWorkTimeID {
		t.Fatalf("unexpected ID: got %q want %q", settings.ID, testWorkTimeID)
	}

	if settings.MinutesADay != int(testMinutesADay) {
		t.Fatalf("unexpected minutesADay: got %d want %d", settings.MinutesADay, testMinutesADay)
	}

	if len(settings.WorkDays) != 5 {
		t.Fatalf("unexpected workDays length: got %d want %d", len(settings.WorkDays), 5)
	}
}

func TestConvertGlobalTimeTrackingSettingsToModel(t *testing.T) {
	t.Parallel()

	apiSettings := youtrack.GlobalTimeTrackingSettings{
		ID: testWorkTimeID,
		WorkTimeSettings: youtrack.WorkTimeSettings{
			ID:             testWorkTimeID,
			MinutesADay:    int(testMinutesADay),
			WorkDays:       []int{1, 2, 3, 4, 5},
			FirstDayOfWeek: int(testFirstDayOfWeek),
			DaysAWeek:      int(testDaysAWeek),
		},
		WorkItemTypes: []youtrack.WorkItemType{
			{ID: testWorkItemTypeID, Name: testWorkItemType, AutoAttached: true},
		},
		AttributePrototypes: []youtrack.WorkItemAttributePrototype{
			{
				ID:   testAttributeID,
				Name: testAttributeName,
				Values: []youtrack.WorkItemAttributeValue{
					{ID: testAttributeID, Name: testAttributeValue, Description: testDescriptionText, AutoAttach: true},
				},
			},
		},
	}

	model, ok := convertGlobalTimeTrackingSettingsToModel(context.Background(), apiSettings)
	if !ok {
		t.Fatal(msgExpectedConversionToSucceed)
	}

	if model.ID.ValueString() != globalTimeTrackingSingletonID {
		t.Fatalf("unexpected resource ID: got %q want %q", model.ID.ValueString(), globalTimeTrackingSingletonID)
	}

	if model.WorkTimeSettings.MinutesADay.ValueInt64() != testMinutesADay {
		t.Fatalf(msgUnexpectedMinutesADay, model.WorkTimeSettings.MinutesADay.ValueInt64(), testMinutesADay)
	}

	if len(model.WorkItemTypes.Elements()) != 1 {
		t.Fatalf("unexpected work_item_types length: got %d want %d", len(model.WorkItemTypes.Elements()), 1)
	}

	var workItemTypes []globalWorkItemTypeModel
	if diags := model.WorkItemTypes.ElementsAs(context.Background(), &workItemTypes, false); diags.HasError() {
		t.Fatalf("failed to extract work_item_types: %v", diags)
	}

	if workItemTypes[0].Name.ValueString() != testWorkItemType {
		t.Fatalf("unexpected work item type name: got %q want %q", workItemTypes[0].Name.ValueString(), testWorkItemType)
	}

	if len(model.AttributePrototypes.Elements()) != 1 {
		t.Fatalf("unexpected attribute_prototypes length: got %d want %d", len(model.AttributePrototypes.Elements()), 1)
	}

	var attributePrototypes []globalWorkItemAttributePrototypeResourceModel
	if diags := model.AttributePrototypes.ElementsAs(context.Background(), &attributePrototypes, false); diags.HasError() {
		t.Fatalf("failed to extract attribute_prototypes: %v", diags)
	}

	if attributePrototypes[0].Name.ValueString() != testAttributeName {
		t.Fatalf("unexpected attribute prototype name: got %q want %q", attributePrototypes[0].Name.ValueString(), testAttributeName)
	}
}

func TestUpdateGlobalTimeTrackingSettingsModelWithTimestamp(t *testing.T) {
	t.Parallel()

	apiSettings := youtrack.GlobalTimeTrackingSettings{
		WorkTimeSettings: youtrack.WorkTimeSettings{
			MinutesADay: int(testMinutesADay),
			WorkDays:    []int{1, 2, 3, 4, 5},
		},
	}

	model := globalTimeTrackingSettingsResourceModel{}
	ok := updateGlobalTimeTrackingSettingsModelWithTimestamp(context.Background(), apiSettings, &model)
	if !ok {
		t.Fatal("expected update with timestamp to succeed")
	}

	if model.LastUpdated.IsNull() || model.LastUpdated.ValueString() == "" {
		t.Fatal("expected last_updated to be set")
	}

	if model.WorkTimeSettings.MinutesADay.ValueInt64() != testMinutesADay {
		t.Fatalf(msgUnexpectedMinutesADay, model.WorkTimeSettings.MinutesADay.ValueInt64(), testMinutesADay)
	}
}

func TestIsTransientRemovedWorkItemTypeListError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "matches removed work item type error",
			err:  errors.New("failed to list work item types: unexpected status code 500: {\"error_description\":\"WorkItemType[178-152] was removed.\"}"),
			want: true,
		},
		{
			name: "does not match other server error",
			err:  errors.New("failed to list work item types: unexpected status code 500: {\"error\":\"server_error\"}"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isTransientRemovedWorkItemTypeListError(tt.err)
			if got != tt.want {
				t.Fatalf("isTransientRemovedWorkItemTypeListError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func makeWorkItemTypeModel(autoAttached bool) globalWorkItemTypeModel {
	return globalWorkItemTypeModel{
		Name:         types.StringValue(testWorkItemType),
		AutoAttached: types.BoolValue(autoAttached),
	}
}

func assertWorkItemTypeCreated(t *testing.T, changes []workItemTypeChange) {
	t.Helper()

	if changes[0].create == nil {
		t.Fatal("expected create action")
	}

	if changes[0].create.Name != testWorkItemType {
		t.Fatalf("unexpected name: got %q want %q", changes[0].create.Name, testWorkItemType)
	}

	if !changes[0].create.AutoAttached {
		t.Fatal("expected auto_attached to be true")
	}
}

func assertWorkItemTypeUpdated(t *testing.T, changes []workItemTypeChange) {
	t.Helper()

	if changes[0].update == nil {
		t.Fatal("expected update action")
	}

	if changes[0].update.AutoAttached {
		t.Fatal("expected auto_attached to be false after update")
	}
}

func assertWorkItemTypeDeleted(t *testing.T, changes []workItemTypeChange) {
	t.Helper()

	if changes[0].deleteID != testWorkItemTypeID {
		t.Fatalf("unexpected deleteID: got %q want %q", changes[0].deleteID, testWorkItemTypeID)
	}
}

func assertWorkItemTypeRenamed(t *testing.T, changes []workItemTypeChange) {
	t.Helper()

	if len(changes) != 1 {
		t.Fatalf("expected exactly one change, got %d", len(changes))
	}

	if changes[0].update == nil {
		t.Fatal("expected rename to be modeled as update action")
	}

	if changes[0].update.ID != testWorkItemTypeID {
		t.Fatalf("unexpected rename update ID: got %q want %q", changes[0].update.ID, testWorkItemTypeID)
	}

	if changes[0].update.Name != testWorkItemType {
		t.Fatalf("unexpected rename target name: got %q want %q", changes[0].update.Name, testWorkItemType)
	}
}

func assertPlanWorkItemTypeChanges(t *testing.T, plan []globalWorkItemTypeModel, current []youtrack.WorkItemType, wantChanges int, wantErr bool, checkChange func(*testing.T, []workItemTypeChange)) {
	t.Helper()

	changes, err := planWorkItemTypeChanges(plan, current)
	if wantErr {
		if err == nil {
			t.Fatal("expected error but got none")
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != wantChanges {
		t.Fatalf("expected %d change(s), got %d", wantChanges, len(changes))
	}
	if checkChange != nil {
		checkChange(t, changes)
	}
}

func TestPlanWorkItemTypeChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		plan        []globalWorkItemTypeModel
		current     []youtrack.WorkItemType
		wantChanges int
		wantErr     bool
		checkChange func(t *testing.T, changes []workItemTypeChange)
	}{
		{
			name:        "creates missing type",
			plan:        []globalWorkItemTypeModel{makeWorkItemTypeModel(true)},
			current:     []youtrack.WorkItemType{},
			wantChanges: 1,
			checkChange: assertWorkItemTypeCreated,
		},
		{
			name:        "updates changed auto_attached",
			plan:        []globalWorkItemTypeModel{makeWorkItemTypeModel(false)},
			current:     []youtrack.WorkItemType{{ID: testWorkItemTypeID, Name: testWorkItemType, AutoAttached: true}},
			wantChanges: 1,
			checkChange: assertWorkItemTypeUpdated,
		},
		{
			name:        "no change when state matches plan",
			plan:        []globalWorkItemTypeModel{makeWorkItemTypeModel(true)},
			current:     []youtrack.WorkItemType{{ID: testWorkItemTypeID, Name: testWorkItemType, AutoAttached: true}},
			wantChanges: 0,
		},
		{
			name:        "deletes type absent from plan",
			plan:        []globalWorkItemTypeModel{},
			current:     []youtrack.WorkItemType{{ID: testWorkItemTypeID, Name: testWorkItemType, AutoAttached: true}},
			wantChanges: 1,
			checkChange: assertWorkItemTypeDeleted,
		},
		{
			name:        "renames existing type",
			plan:        []globalWorkItemTypeModel{makeWorkItemTypeModel(true)},
			current:     []youtrack.WorkItemType{{ID: testWorkItemTypeID, Name: "Legacy", AutoAttached: true}},
			wantChanges: 1,
			checkChange: assertWorkItemTypeRenamed,
		},
		{
			name:        "renames and updates auto_attached",
			plan:        []globalWorkItemTypeModel{makeWorkItemTypeModel(false)},
			current:     []youtrack.WorkItemType{{ID: testWorkItemTypeID, Name: "Legacy", AutoAttached: true}},
			wantChanges: 1,
			checkChange: assertWorkItemTypeRenamed,
		},
		{
			name: "returns error for empty name",
			plan: []globalWorkItemTypeModel{{
				Name:         types.StringValue(""),
				AutoAttached: types.BoolValue(false),
			}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertPlanWorkItemTypeChanges(t, tc.plan, tc.current, tc.wantChanges, tc.wantErr, tc.checkChange)
		})
	}
}

func TestWaitOrContextDone(t *testing.T) {
	t.Parallel()

	t.Run("returns nil once the delay elapses", func(t *testing.T) {
		t.Parallel()

		if err := waitOrContextDone(context.Background(), time.Millisecond); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("returns ctx error when cancelled first", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if err := waitOrContextDone(ctx, time.Second); err == nil {
			t.Fatal("expected an error when the context is already cancelled")
		}
	})
}

func TestWorkItemTypeChangesSettled(t *testing.T) {
	t.Parallel()

	deletedIDs := map[string]struct{}{testWorkItemTypeID: {}}
	expectedByName := map[string]bool{testWorkItemType: true}

	tests := []struct {
		name           string
		current        []youtrack.WorkItemType
		deletedIDs     map[string]struct{}
		expectedByName map[string]bool
		want           bool
	}{
		{
			name:           "settled when there is nothing to check",
			current:        []youtrack.WorkItemType{},
			deletedIDs:     map[string]struct{}{},
			expectedByName: map[string]bool{},
			want:           true,
		},
		{
			name:           "not settled while a deleted ID is still present",
			current:        []youtrack.WorkItemType{{ID: testWorkItemTypeID, Name: testWorkItemType, AutoAttached: true}},
			deletedIDs:     deletedIDs,
			expectedByName: map[string]bool{},
			want:           false,
		},
		{
			name:           "settled once the deleted ID is gone",
			current:        []youtrack.WorkItemType{},
			deletedIDs:     deletedIDs,
			expectedByName: map[string]bool{},
			want:           true,
		},
		{
			name:           "not settled while a created type is missing",
			current:        []youtrack.WorkItemType{},
			deletedIDs:     map[string]struct{}{},
			expectedByName: expectedByName,
			want:           false,
		},
		{
			name:           "not settled while auto_attached does not match yet",
			current:        []youtrack.WorkItemType{{ID: testWorkItemTypeID, Name: testWorkItemType, AutoAttached: false}},
			deletedIDs:     map[string]struct{}{},
			expectedByName: expectedByName,
			want:           false,
		},
		{
			name:           "settled once the created type matches",
			current:        []youtrack.WorkItemType{{ID: testWorkItemTypeID, Name: testWorkItemType, AutoAttached: true}},
			deletedIDs:     map[string]struct{}{},
			expectedByName: expectedByName,
			want:           true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := workItemTypeChangesSettled(tc.current, tc.deletedIDs, tc.expectedByName)
			if got != tc.want {
				t.Fatalf("workItemTypeChangesSettled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func startWorkItemTypesServer(t *testing.T, handler func(attempt int) (status int, body []youtrack.WorkItemType)) *youtrack.Client {
	t.Helper()

	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != workItemTypesTestPath {
			http.NotFound(w, r)
			return
		}

		attempt++
		status, body := handler(attempt)
		w.WriteHeader(status)
		if body != nil {
			if err := json.NewEncoder(w).Encode(body); err != nil {
				t.Fatalf("failed to encode work item types response: %v", err)
			}
		}
	}))
	t.Cleanup(server.Close)

	client, err := youtrack.NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	return client
}

func TestWaitForWorkItemTypeChangesToSettleNoOpWithoutRelevantChanges(t *testing.T) {
	t.Parallel()

	r := &globalTimeTrackingSettingsResource{}
	var diagnostics diag.Diagnostics

	if !r.waitForWorkItemTypeChangesToSettle(context.Background(), []workItemTypeChange{}, &diagnostics) {
		t.Fatalf("expected no-op success, got diagnostics: %v", diagnostics)
	}
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}

func TestWaitForWorkItemTypeChangesToSettleRetriesPastTransientListError(t *testing.T) {
	t.Parallel()

	client := startWorkItemTypesServer(t, func(attempt int) (int, []youtrack.WorkItemType) {
		if attempt == 1 {
			// Simulate a transient failure unrelated to the "was removed" case that
			// listWorkItemTypesWithRetry special-cases, e.g. a 502 from a load balancer.
			return http.StatusBadGateway, nil
		}
		return http.StatusOK, []youtrack.WorkItemType{}
	})

	r := &globalTimeTrackingSettingsResource{client: client}
	var diagnostics diag.Diagnostics

	changes := []workItemTypeChange{{deleteID: testWorkItemTypeID}}
	if !r.waitForWorkItemTypeChangesToSettle(context.Background(), changes, &diagnostics) {
		t.Fatalf("expected retry to succeed, got diagnostics: %v", diagnostics)
	}
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}

func TestWaitForWorkItemTypeChangesToSettleTimesOut(t *testing.T) {
	t.Parallel()

	client := startWorkItemTypesServer(t, func(_ int) (int, []youtrack.WorkItemType) {
		// The deleted type never disappears from the list.
		return http.StatusOK, []youtrack.WorkItemType{{ID: testWorkItemTypeID, Name: testWorkItemType, AutoAttached: true}}
	})

	r := &globalTimeTrackingSettingsResource{client: client}
	var diagnostics diag.Diagnostics

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	changes := []workItemTypeChange{{deleteID: testWorkItemTypeID}}
	if r.waitForWorkItemTypeChangesToSettle(ctx, changes, &diagnostics) {
		t.Fatal("expected failure once the context deadline is exceeded")
	}
	if !diagnostics.HasError() {
		t.Fatal("expected diagnostics to report the timeout")
	}
}
