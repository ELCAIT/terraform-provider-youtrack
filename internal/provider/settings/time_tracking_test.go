package settings

import (
	"context"
	"testing"

	youtrack "github.com/elcait/youtrack-api-client/client"

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
