package provider

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMatchBundleValues(t *testing.T) {
	t.Parallel()

	existing := []youtrack.EnumBundleElement{
		{ID: "67-1", Name: "Major"},
		{ID: "67-2", Name: "Minor"},
		{ID: "67-3", Name: "Legacy"},
	}
	planned := []enumBundleValueModel{
		{Name: types.StringValue(" minor ")},
		{Name: types.StringValue("Critical")},
		{Name: types.StringValue("Major")},
	}

	matched, unmatched := matchBundleValues(planned, existing,
		func(v enumBundleValueModel) string { return v.Name.ValueString() },
		func(v youtrack.EnumBundleElement) string { return v.Name },
	)

	helpers.AssertFieldEqual(t, "FirstMatch", matched[0].ID, "67-2")
	helpers.AssertFieldEqual(t, "NewValueUnmatched", matched[1] == nil, true)
	helpers.AssertFieldEqual(t, "ThirdMatch", matched[2].ID, "67-1")
	helpers.AssertFieldEqual(t, "UnmatchedLength", len(unmatched), 1)
	helpers.AssertFieldEqual(t, "UnmatchedID", unmatched[0].ID, "67-3")
}

// recordingEnumReconciler is an enum reconciler whose client calls are
// recorded instead of sent.
func recordingEnumReconciler(calls *[]string, removeErr error) bundleValueReconciler[enumBundleValueModel, youtrack.EnumBundleElement, youtrack.EnumBundleValueUpdate] {
	return bundleValueReconciler[enumBundleValueModel, youtrack.EnumBundleElement, youtrack.EnumBundleValueUpdate]{
		plannedName: func(v enumBundleValueModel) string { return v.Name.ValueString() },
		apiID:       func(v youtrack.EnumBundleElement) string { return v.ID },
		apiName:     func(v youtrack.EnumBundleElement) string { return v.Name },
		toUpdate: func(v enumBundleValueModel, current youtrack.EnumBundleElement, ordinal int) youtrack.EnumBundleValueUpdate {
			return v.toValueUpdate(current, ordinal)
		},
		currentAsUpdate: enumValueAsUpdate,
		add: func(_ context.Context, v enumBundleValueModel, ordinal int) error {
			*calls = append(*calls, fmt.Sprintf("add %s@%d", v.Name.ValueString(), ordinal))
			return nil
		},
		replace: func(_ context.Context, valueID string, update youtrack.EnumBundleValueUpdate) error {
			*calls = append(*calls, fmt.Sprintf("replace %s=%s@%d", valueID, update.Name, *update.Ordinal))
			return nil
		},
		remove: func(_ context.Context, valueID string) error {
			*calls = append(*calls, "remove "+valueID)
			return removeErr
		},
	}
}

// TestBundleValueReconcileKeepsIDs is the regression test for values being
// recreated on every update: unchanged values must not be written at all, and
// changed ones must be edited in place rather than re-added.
func TestBundleValueReconcileKeepsIDs(t *testing.T) {
	t.Parallel()

	existing := []youtrack.EnumBundleElement{
		{ID: "67-1", Name: "Major", Ordinal: 1},
		{ID: "67-2", Name: "Minor", Ordinal: 2, Description: "Low impact"},
		{ID: "67-3", Name: "Legacy", Ordinal: 3},
	}
	planned := []enumBundleValueModel{
		{Name: types.StringValue("Major"), Description: types.StringUnknown(), LocalizedName: types.StringUnknown(), Archived: types.BoolValue(false)},
		{Name: types.StringValue("Critical"), Description: types.StringUnknown(), LocalizedName: types.StringUnknown(), Archived: types.BoolValue(false)},
		{Name: types.StringValue("Minor"), Description: types.StringUnknown(), LocalizedName: types.StringUnknown(), Archived: types.BoolValue(true)},
	}

	var calls []string
	if err := recordingEnumReconciler(&calls, nil).reconcile(context.Background(), planned, existing); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"remove 67-3", "add Critical@2", "replace 67-2=Minor@3"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("unexpected calls:\n got: %v\nwant: %v", calls, want)
	}
}

func TestBundleValueReconcileExplainsWorkflowRemoval(t *testing.T) {
	t.Parallel()

	workflowErr := errors.New(`{"error_type":"workflow","error_description":"Priority is required"}`)
	existing := []youtrack.EnumBundleElement{{ID: "67-1", Name: "Major", Ordinal: 1}}

	var calls []string
	err := recordingEnumReconciler(&calls, workflowErr).reconcile(context.Background(), nil, existing)
	if err == nil || !strings.Contains(err.Error(), "set archived = true") {
		t.Fatalf("expected a hint to archive the value, got %v", err)
	}
}

func TestEnumValueUpdateKeepsUnsetOptionalFields(t *testing.T) {
	t.Parallel()

	current := youtrack.EnumBundleElement{ID: "67-1", Name: "Major", LocalizedName: "Majeur", Description: "Big", Ordinal: 1}

	unset := enumBundleValueModel{Name: types.StringValue("Major"), LocalizedName: types.StringUnknown(), Description: types.StringUnknown(), Archived: types.BoolValue(false)}
	helpers.AssertFieldEqual(t, "UnsetIsUnchanged", reflect.DeepEqual(unset.toValueUpdate(current, 1), enumValueAsUpdate(current)), true)

	cleared := enumBundleValueModel{Name: types.StringValue("Major"), LocalizedName: types.StringNull(), Description: types.StringValue("Big"), Archived: types.BoolValue(false)}
	update := cleared.toValueUpdate(current, 1)
	helpers.AssertFieldEqual(t, "NullLocalizedNameIsCleared", update.LocalizedName == nil, true)
}

func TestStateValueUpdateCarriesResolved(t *testing.T) {
	t.Parallel()

	current := youtrack.StateBundleElement{ID: "68-1", Name: "Done", Ordinal: 1}
	value := stateBundleValueModel{
		Name: types.StringValue("Done"), LocalizedName: types.StringUnknown(), Description: types.StringUnknown(),
		IsResolved: types.BoolValue(true), Archived: types.BoolValue(false),
	}

	update := value.toValueUpdate(current, 1)
	helpers.AssertFieldEqual(t, "IsResolved", update.IsResolved, true)
	helpers.AssertFieldEqual(t, "NeedsUpdate", !reflect.DeepEqual(update, stateValueAsUpdate(current)), true)
}

func TestSortedByOrdinal(t *testing.T) {
	t.Parallel()

	values := []youtrack.EnumBundleElement{{Name: "C", Ordinal: 3}, {Name: "A", Ordinal: 1}, {Name: "B", Ordinal: 2}}
	sorted := sortedByOrdinal(values, func(v youtrack.EnumBundleElement) int { return v.Ordinal })

	helpers.AssertFieldEqual(t, "First", sorted[0].Name, "A")
	helpers.AssertFieldEqual(t, "Last", sorted[2].Name, "C")
	helpers.AssertFieldEqual(t, "InputUntouched", values[0].Name, "C")
}
