package settings

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	errUnableToReadTimeTracking     = "Unable to Read YouTrack Global Time Tracking Settings"
	errUnableToReadWorkItemTypes    = "Unable to Read YouTrack Work Item Types"
	errUpdatingWorkTimeSettings     = "Error updating work time settings"
	errConvertingTimeTracking       = "Error converting time tracking settings"
	errConvertingWorkDays           = "Failed to convert work_days list"
	errConvertingNestedTimeTracking = "Failed to convert nested time tracking attributes"
	errManagingWorkItemTypes        = "Error managing work item types"
	errWorkItemTypeEmptyName        = "each work_item_types entry must have a non-empty name"
	warnWorkItemTypesNotConfirmed   = "Work item type changes not fully confirmed"
	globalTimeTrackingSingletonID   = "global"
	// workItemTypeBeingRemovedSuffix is appended by YouTrack when a work item type
	// is soft-deleted. The provider filters these out so they never appear in state.
	workItemTypeBeingRemovedSuffix = " (being removed)"
	defaultWorkMinutesADay         = 480
	// workItemTypePollRetryDelay is the pause between polls shared by both the read-retry (up to
	// workItemTypesReadMaxAttempts) and post-mutation settle-wait (up to workItemTypeSettleMaxAttempts)
	// loops below; they poll the same endpoint for the same kind of eventual-consistency flakiness.
	workItemTypePollRetryDelay    = 500 * time.Millisecond
	workItemTypesReadMaxAttempts  = 8
	workItemTypeSettleMaxAttempts = 20
)

// computedWhenUnconfiguredSetModifier marks a set attribute as (known after apply)
// when it is not set by the user in configuration. This prevents plan-consistency
// errors when the API may return a different set than what was in prior state.
type computedWhenUnconfiguredSetModifier struct{}

var defaultWorkDays = []int{1, 2, 3, 4, 5}

// workItemTypeChange represents a single reconciliation action for a work item type.
type workItemTypeChange struct {
	create   *youtrack.WorkItemType
	update   *youtrack.WorkItemType
	deleteID string
}

func (m computedWhenUnconfiguredSetModifier) Description(_ context.Context) string {
	return "If not configured, this list is always populated from the API after apply."
}

func (m computedWhenUnconfiguredSetModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m computedWhenUnconfiguredSetModifier) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if req.ConfigValue.IsNull() {
		resp.PlanValue = planValueWhenUnconfigured(req.StateValue)
		return
	}

	stateByName := buildStateByName(ctx, req.StateValue)

	var configItems []globalWorkItemTypeModel
	if diags := req.ConfigValue.ElementsAs(ctx, &configItems, false); diags.HasError() {
		return
	}

	merged := mergeConfigWithState(configItems, stateByName)

	planned, diags := types.SetValueFrom(ctx, workItemTypeObjectType, merged)
	if diags.HasError() {
		return
	}

	resp.PlanValue = planned
}

// planValueWhenUnconfigured returns the plan value to use when work_item_types is not set in config.
func planValueWhenUnconfigured(stateValue types.Set) types.Set {
	if !stateValue.IsNull() && !stateValue.IsUnknown() {
		return stateValue
	}

	return types.SetUnknown(workItemTypeObjectType)
}

// buildStateByName builds a name-keyed map of work item type models from state.
func buildStateByName(ctx context.Context, stateValue types.Set) map[string]globalWorkItemTypeModel {
	byName := make(map[string]globalWorkItemTypeModel)
	if stateValue.IsNull() || stateValue.IsUnknown() {
		return byName
	}

	var stateItems []globalWorkItemTypeModel
	if diags := stateValue.ElementsAs(ctx, &stateItems, false); diags.HasError() {
		return byName
	}

	for _, s := range stateItems {
		byName[s.Name.ValueString()] = s
	}

	return byName
}

// mergeConfigWithState produces the planned work item type list by merging config entries
// with their existing state counterparts (to preserve server-assigned IDs).
func mergeConfigWithState(configItems []globalWorkItemTypeModel, stateByName map[string]globalWorkItemTypeModel) []globalWorkItemTypeModel {
	merged := make([]globalWorkItemTypeModel, 0, len(configItems))
	for _, ci := range configItems {
		merged = append(merged, mergeWorkItemTypeWithState(ci, stateByName))
	}

	return merged
}

// mergeWorkItemTypeWithState returns the planned model for a single config item.
func mergeWorkItemTypeWithState(ci globalWorkItemTypeModel, stateByName map[string]globalWorkItemTypeModel) globalWorkItemTypeModel {
	name := ci.Name.ValueString()
	si, exists := stateByName[name]

	if !exists {
		return newWorkItemTypeFromConfig(ci)
	}

	autoAttached := si.AutoAttached
	if !ci.AutoAttached.IsNull() && !ci.AutoAttached.IsUnknown() {
		autoAttached = ci.AutoAttached
	}

	return globalWorkItemTypeModel{
		Name:         types.StringValue(name),
		AutoAttached: autoAttached,
	}
}

// newWorkItemTypeFromConfig creates a planned model for a work item type that does not yet exist in state.
func newWorkItemTypeFromConfig(ci globalWorkItemTypeModel) globalWorkItemTypeModel {
	autoAttached := ci.AutoAttached
	if autoAttached.IsNull() || autoAttached.IsUnknown() {
		autoAttached = types.BoolValue(false)
	}

	return globalWorkItemTypeModel{
		Name:         ci.Name,
		AutoAttached: autoAttached,
	}
}

func boolValueOrFalse(value types.Bool) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueBool()
}

func buildCurrentWorkItemTypesByName(current []youtrack.WorkItemType) map[string]youtrack.WorkItemType {
	currentByName := make(map[string]youtrack.WorkItemType, len(current))
	for _, item := range current {
		currentByName[item.Name] = item
	}

	return currentByName
}

func processPlannedWorkItemTypes(plan []globalWorkItemTypeModel, currentByName map[string]youtrack.WorkItemType) ([]workItemTypeChange, map[string]struct{}, map[string]globalWorkItemTypeModel, error) {
	changes := make([]workItemTypeChange, 0, len(plan))
	desiredNames := make(map[string]struct{}, len(plan))
	unmatchedPlanByName := make(map[string]globalWorkItemTypeModel)

	for _, plannedType := range plan {
		name := plannedType.Name.ValueString()
		if name == "" {
			return nil, nil, nil, fmt.Errorf("%s", errWorkItemTypeEmptyName)
		}

		desiredNames[name] = struct{}{}
		planAutoAttached := boolValueOrFalse(plannedType.AutoAttached)

		existing, exists := currentByName[name]
		if !exists {
			unmatchedPlanByName[name] = plannedType
			continue
		}

		if existing.AutoAttached != planAutoAttached {
			updated := youtrack.WorkItemType{ID: existing.ID, Name: name, AutoAttached: planAutoAttached}
			changes = append(changes, workItemTypeChange{update: &updated})
		}
	}

	return changes, desiredNames, unmatchedPlanByName, nil
}

func findUnmatchedCurrentByName(currentByName map[string]youtrack.WorkItemType, desiredNames map[string]struct{}) map[string]youtrack.WorkItemType {
	unmatchedCurrentByName := make(map[string]youtrack.WorkItemType)
	for name, existing := range currentByName {
		if _, ok := desiredNames[name]; !ok {
			unmatchedCurrentByName[name] = existing
		}
	}

	return unmatchedCurrentByName
}

func appendCreateChanges(changes []workItemTypeChange, unmatchedPlanByName map[string]globalWorkItemTypeModel) []workItemTypeChange {
	unmatchedPlanNames := sortedWorkItemTypePlanNames(unmatchedPlanByName)
	for _, name := range unmatchedPlanNames {
		planItem := unmatchedPlanByName[name]
		created := youtrack.WorkItemType{Name: name, AutoAttached: boolValueOrFalse(planItem.AutoAttached)}
		changes = append(changes, workItemTypeChange{create: &created})
	}

	return changes
}

func appendDeleteChanges(changes []workItemTypeChange, unmatchedCurrentByName map[string]youtrack.WorkItemType) []workItemTypeChange {
	unmatchedCurrentNames := sortedWorkItemTypeCurrentNames(unmatchedCurrentByName)
	for _, name := range unmatchedCurrentNames {
		existing := unmatchedCurrentByName[name]
		changes = append(changes, workItemTypeChange{deleteID: existing.ID})
	}

	return changes
}

// planWorkItemTypeChanges computes the set of create/update/delete actions required to reconcile
// the desired plan list against the current API state.
func planWorkItemTypeChanges(plan []globalWorkItemTypeModel, current []youtrack.WorkItemType) ([]workItemTypeChange, error) {
	currentByName := buildCurrentWorkItemTypesByName(current)

	changes, desiredNames, unmatchedPlanByName, err := processPlannedWorkItemTypes(plan, currentByName)
	if err != nil {
		return nil, err
	}

	unmatchedCurrentByName := findUnmatchedCurrentByName(currentByName, desiredNames)

	renameChanges := planRenameWorkItemTypeChanges(unmatchedPlanByName, unmatchedCurrentByName)
	changes = append(changes, renameChanges...)
	changes = appendCreateChanges(changes, unmatchedPlanByName)
	changes = appendDeleteChanges(changes, unmatchedCurrentByName)

	return changes, nil
}

func planRenameWorkItemTypeChanges(unmatchedPlanByName map[string]globalWorkItemTypeModel, unmatchedCurrentByName map[string]youtrack.WorkItemType) []workItemTypeChange {
	if len(unmatchedPlanByName) == 0 || len(unmatchedCurrentByName) == 0 {
		return nil
	}

	var renameChanges []workItemTypeChange
	planNames := sortedWorkItemTypePlanNames(unmatchedPlanByName)

	for _, name := range planNames {
		planItem := unmatchedPlanByName[name]
		planAutoAttached := boolValueOrFalse(planItem.AutoAttached)

		candidates := findCurrentRenameCandidates(unmatchedCurrentByName, planAutoAttached)
		if len(candidates) != 1 {
			continue
		}

		existing := unmatchedCurrentByName[candidates[0]]
		updated := youtrack.WorkItemType{ID: existing.ID, Name: name, AutoAttached: planAutoAttached}
		renameChanges = append(renameChanges, workItemTypeChange{update: &updated})

		delete(unmatchedPlanByName, name)
		delete(unmatchedCurrentByName, candidates[0])
	}

	if len(unmatchedPlanByName) == 1 && len(unmatchedCurrentByName) == 1 {
		remainingPlanNames := sortedWorkItemTypePlanNames(unmatchedPlanByName)
		remainingCurrentNames := sortedWorkItemTypeCurrentNames(unmatchedCurrentByName)

		planName := remainingPlanNames[0]
		planItem := unmatchedPlanByName[planName]
		planAutoAttached := boolValueOrFalse(planItem.AutoAttached)
		existing := unmatchedCurrentByName[remainingCurrentNames[0]]

		updated := youtrack.WorkItemType{ID: existing.ID, Name: planName, AutoAttached: planAutoAttached}
		renameChanges = append(renameChanges, workItemTypeChange{update: &updated})

		delete(unmatchedPlanByName, planName)
		delete(unmatchedCurrentByName, remainingCurrentNames[0])
	}

	return renameChanges
}

func findCurrentRenameCandidates(unmatchedCurrentByName map[string]youtrack.WorkItemType, autoAttached bool) []string {
	var candidates []string
	for name, existing := range unmatchedCurrentByName {
		if existing.AutoAttached == autoAttached {
			candidates = append(candidates, name)
		}
	}

	sort.Strings(candidates)
	return candidates
}

func sortedWorkItemTypePlanNames(items map[string]globalWorkItemTypeModel) []string {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

func sortedWorkItemTypeCurrentNames(items map[string]youtrack.WorkItemType) []string {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

var (
	attrValueObjectType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":          types.StringType,
			"name":        types.StringType,
			"description": types.StringType,
			"auto_attach": types.BoolType,
		},
	}

	projectAttrObjectType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":      types.StringType,
			"name":    types.StringType,
			"ordinal": types.Int64Type,
			"values":  types.ListType{ElemType: attrValueObjectType},
		},
	}

	workItemTypeObjectType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":          types.StringType,
			"auto_attached": types.BoolType,
		},
	}

	attrPrototypeObjectType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":        types.StringType,
			"name":      types.StringType,
			"values":    types.ListType{ElemType: attrValueObjectType},
			"instances": types.ListType{ElemType: projectAttrObjectType},
		},
	}
)

// getGlobalTimeTrackingSettingsAndHandleError fetches global time tracking settings via API and handles errors.
func (r *globalTimeTrackingSettingsResource) getGlobalTimeTrackingSettingsAndHandleError(ctx context.Context, diagnostics *diag.Diagnostics) (youtrack.GlobalTimeTrackingSettings, bool) {
	workItemTypes, err := r.listWorkItemTypesWithRetry(ctx)
	if err != nil {
		diagnostics.AddError(
			errUnableToReadWorkItemTypes,
			err.Error(),
		)
		return youtrack.GlobalTimeTrackingSettings{}, false
	}

	return r.getGlobalTimeTrackingSettingsWithWorkItemTypes(ctx, workItemTypes, diagnostics)
}

// getGlobalTimeTrackingSettingsWithWorkItemTypes fetches global time tracking settings via API and
// combines them with an already-known-fresh work item types list, avoiding a redundant list call
// right after a caller (e.g. syncWorkItemTypesIfConfigured) has already confirmed the list reflects
// a just-applied mutation.
func (r *globalTimeTrackingSettingsResource) getGlobalTimeTrackingSettingsWithWorkItemTypes(ctx context.Context, workItemTypes []youtrack.WorkItemType, diagnostics *diag.Diagnostics) (youtrack.GlobalTimeTrackingSettings, bool) {
	settings, err := r.client.GetGlobalTimeTrackingSettings(ctx)
	if err != nil {
		diagnostics.AddError(
			errUnableToReadTimeTracking,
			err.Error(),
		)
		return youtrack.GlobalTimeTrackingSettings{}, false
	}

	settings.WorkItemTypes = workItemTypes

	return settings, true
}

func (r *globalTimeTrackingSettingsResource) listWorkItemTypesWithRetry(ctx context.Context) ([]youtrack.WorkItemType, error) {
	var lastErr error

	for attempt := 1; attempt <= workItemTypesReadMaxAttempts; attempt++ {
		workItemTypes, err := r.client.ListWorkItemTypes(ctx)
		if err == nil {
			return workItemTypes, nil
		}

		lastErr = err
		if !isRetryableWorkItemTypeListError(err) || attempt == workItemTypesReadMaxAttempts {
			break
		}

		if err := helpers.WaitOrContextDone(ctx, workItemTypePollRetryDelay); err != nil {
			return nil, err
		}
	}

	return nil, lastErr
}

func isTransientRemovedWorkItemTypeListError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "workitemtype[") && strings.Contains(errMsg, "was removed")
}

// isRetryableWorkItemTypeListError reports whether a failed work item type list read is worth
// polling again. Besides the classified "was removed" error YouTrack returns right after a
// deletion, this covers the ordinary transient failures a YouTrack instance behind a proxy
// produces — 5xx responses, 408/429, and transport errors that never reached the API at all.
// Only a definitive client error (401/403/404, a malformed request) is treated as permanent,
// because no amount of retrying will change the outcome.
func isRetryableWorkItemTypeListError(err error) bool {
	if err == nil {
		return false
	}

	if isTransientRemovedWorkItemTypeListError(err) {
		return true
	}

	var httpErr *youtrack.HTTPError
	if !errors.As(err, &httpErr) {
		// Transport failure or an error the client didn't classify: assume transient, since
		// giving up on a connection reset would abandon a mutation that already succeeded.
		return true
	}

	switch httpErr.StatusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return true
	default:
		return httpErr.StatusCode >= http.StatusInternalServerError
	}
}

// updateWorkTimeSettingsAndHandleError updates work time settings via API and handles errors.
func (r *globalTimeTrackingSettingsResource) updateWorkTimeSettingsAndHandleError(ctx context.Context, settings youtrack.WorkTimeSettings, diagnostics *diag.Diagnostics) bool {
	_, err := r.client.UpdateWorkTimeSettings(ctx, settings)
	if err != nil {
		diagnostics.AddError(
			errUpdatingWorkTimeSettings,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "work time settings", err),
		)
		return false
	}

	return true
}

// updateGlobalTimeTrackingSettingsModelWithTimestamp updates model from API response and sets timestamp.
func updateGlobalTimeTrackingSettingsModelWithTimestamp(ctx context.Context, settings youtrack.GlobalTimeTrackingSettings, resourceModel *globalTimeTrackingSettingsResourceModel) bool {
	converted, ok := convertGlobalTimeTrackingSettingsToModel(ctx, settings)
	if !ok {
		return false
	}

	*resourceModel = *converted
	resourceModel.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	return true
}

// applyWorkTimeSettingsAndUpdateModel converts the plan model to work time settings, updates them via API,
// fetches back the global settings, and updates the model with a timestamp.
func (r *globalTimeTrackingSettingsResource) applyWorkTimeSettingsAndUpdateModel(ctx context.Context, plan *globalTimeTrackingSettingsResourceModel, diagnostics *diag.Diagnostics) bool {
	workTimeSettings, ok := convertModelToWorkTimeSettings(*plan)
	if !ok {
		diagnostics.AddError(errConvertingTimeTracking, errConvertingWorkDays)
		return false
	}

	if !r.updateWorkTimeSettingsAndHandleError(ctx, workTimeSettings, diagnostics) {
		return false
	}

	syncedWorkItemTypes, ok := r.syncWorkItemTypesIfConfigured(ctx, plan, diagnostics)
	if !ok {
		return false
	}

	var globalSettings youtrack.GlobalTimeTrackingSettings
	if syncedWorkItemTypes.known {
		globalSettings, ok = r.getGlobalTimeTrackingSettingsWithWorkItemTypes(ctx, syncedWorkItemTypes.types, diagnostics)
	} else {
		globalSettings, ok = r.getGlobalTimeTrackingSettingsAndHandleError(ctx, diagnostics)
	}
	if !ok {
		return false
	}

	if !updateGlobalTimeTrackingSettingsModelWithTimestamp(ctx, globalSettings, plan) {
		diagnostics.AddError(errConvertingTimeTracking, errConvertingNestedTimeTracking)
		return false
	}

	return true
}

// convertModelToWorkTimeSettings converts a resource model to the API work time settings model.
func convertModelToWorkTimeSettings(model globalTimeTrackingSettingsResourceModel) (youtrack.WorkTimeSettings, bool) {
	var workDaysInt64 []int64
	diags := model.WorkTimeSettings.WorkDays.ElementsAs(context.Background(), &workDaysInt64, false)
	if diags.HasError() {
		return youtrack.WorkTimeSettings{}, false
	}

	workDays := make([]int, 0, len(workDaysInt64))
	for _, day := range workDaysInt64 {
		workDays = append(workDays, int(day))
	}

	return youtrack.WorkTimeSettings{
		ID:          model.WorkTimeSettings.ID.ValueString(),
		MinutesADay: int(model.WorkTimeSettings.MinutesADay.ValueInt64()),
		WorkDays:    workDays,
	}, true
}

// convertGlobalTimeTrackingSettingsToModel converts API response to Terraform resource model.
func convertGlobalTimeTrackingSettingsToModel(ctx context.Context, settings youtrack.GlobalTimeTrackingSettings) (*globalTimeTrackingSettingsResourceModel, bool) {
	workTimeSettings, ok := convertWorkTimeSettingsToModel(ctx, settings.WorkTimeSettings)
	if !ok {
		return nil, false
	}

	workItemTypes, diags := convertWorkItemTypesToModel(ctx, settings.WorkItemTypes)
	if diags.HasError() {
		return nil, false
	}

	attributePrototypes, diags := convertAttributePrototypesToModel(ctx, settings.AttributePrototypes)
	if diags.HasError() {
		return nil, false
	}

	return &globalTimeTrackingSettingsResourceModel{
		ID:                  types.StringValue(globalTimeTrackingSingletonID),
		WorkTimeSettings:    workTimeSettings,
		WorkItemTypes:       workItemTypes,
		AttributePrototypes: attributePrototypes,
	}, true
}

func convertWorkTimeSettingsToModel(ctx context.Context, settings youtrack.WorkTimeSettings) (globalWorkTimeSettingsModel, bool) {
	workDaysInt64 := make([]int64, 0, len(settings.WorkDays))
	for _, day := range settings.WorkDays {
		workDaysInt64 = append(workDaysInt64, int64(day))
	}

	workDaysList, diags := types.ListValueFrom(ctx, types.Int64Type, workDaysInt64)
	if diags.HasError() {
		return globalWorkTimeSettingsModel{}, false
	}

	id := settings.ID
	if id == "" {
		id = globalTimeTrackingSingletonID
	}

	return globalWorkTimeSettingsModel{
		ID:             types.StringValue(id),
		MinutesADay:    types.Int64Value(int64(settings.MinutesADay)),
		WorkDays:       workDaysList,
		FirstDayOfWeek: types.Int64Value(int64(settings.FirstDayOfWeek)),
		DaysAWeek:      types.Int64Value(int64(settings.DaysAWeek)),
	}, true
}

func convertWorkItemTypesToModel(ctx context.Context, items []youtrack.WorkItemType) (types.Set, diag.Diagnostics) {
	active := make([]youtrack.WorkItemType, 0, len(items))
	for _, item := range items {
		if !strings.HasSuffix(item.Name, workItemTypeBeingRemovedSuffix) {
			active = append(active, item)
		}
	}

	converted := make([]globalWorkItemTypeModel, 0, len(active))
	for _, item := range active {
		converted = append(converted, globalWorkItemTypeModel{
			Name:         types.StringValue(item.Name),
			AutoAttached: types.BoolValue(item.AutoAttached),
		})
	}

	return types.SetValueFrom(ctx, workItemTypeObjectType, converted)
}

func convertAttributePrototypesToModel(ctx context.Context, items []youtrack.WorkItemAttributePrototype) (types.List, diag.Diagnostics) {
	converted := make([]globalWorkItemAttributePrototypeResourceModel, 0, len(items))
	for _, item := range items {
		values, diags := convertAttributeValuesToModel(ctx, item.Values)
		if diags.HasError() {
			return types.ListNull(attrPrototypeObjectType), diags
		}

		instances, diags := convertProjectAttributesToModel(ctx, item.Instances)
		if diags.HasError() {
			return types.ListNull(attrPrototypeObjectType), diags
		}

		converted = append(converted, globalWorkItemAttributePrototypeResourceModel{
			ID:        types.StringValue(item.ID),
			Name:      types.StringValue(item.Name),
			Values:    values,
			Instances: instances,
		})
	}

	return types.ListValueFrom(ctx, attrPrototypeObjectType, converted)
}

func convertProjectAttributesToModel(ctx context.Context, items []youtrack.WorkItemProjectAttribute) (types.List, diag.Diagnostics) {
	converted := make([]globalWorkItemProjectAttributeModel, 0, len(items))
	for _, item := range items {
		values, diags := convertAttributeValuesToModel(ctx, item.Values)
		if diags.HasError() {
			return types.ListNull(projectAttrObjectType), diags
		}

		converted = append(converted, globalWorkItemProjectAttributeModel{
			ID:      types.StringValue(item.ID),
			Name:    types.StringValue(item.Name),
			Ordinal: types.Int64Value(int64(item.Ordinal)),
			Values:  values,
		})
	}

	return types.ListValueFrom(ctx, projectAttrObjectType, converted)
}

func convertAttributeValuesToModel(ctx context.Context, items []youtrack.WorkItemAttributeValue) (types.List, diag.Diagnostics) {
	converted := make([]globalWorkItemAttributeValueResourceModel, 0, len(items))
	for _, item := range items {
		converted = append(converted, globalWorkItemAttributeValueResourceModel{
			ID:          types.StringValue(item.ID),
			Name:        types.StringValue(item.Name),
			Description: helpers.StringOrNull(item.Description),
			AutoAttach:  types.BoolValue(item.AutoAttach),
		})
	}

	return types.ListValueFrom(ctx, attrValueObjectType, converted)
}

// applyWorkItemTypeChanges executes a slice of reconciliation actions against the API.
func (r *globalTimeTrackingSettingsResource) applyWorkItemTypeChanges(ctx context.Context, changes []workItemTypeChange, diagnostics *diag.Diagnostics) bool {
	for _, c := range changes {
		switch {
		case c.create != nil:
			if _, err := r.client.CreateWorkItemType(ctx, *c.create); err != nil {
				diagnostics.AddError(errManagingWorkItemTypes,
					fmt.Sprintf("could not create work item type %q: %v", c.create.Name, err))
				return false
			}
		case c.update != nil:
			if _, err := r.client.UpdateWorkItemType(ctx, *c.update); err != nil {
				diagnostics.AddError(errManagingWorkItemTypes,
					fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "work item type "+c.update.Name, err))
				return false
			}
		case c.deleteID != "":
			if err := r.client.DeleteWorkItemType(ctx, c.deleteID); err != nil {
				diagnostics.AddError(errManagingWorkItemTypes,
					fmt.Sprintf("could not delete work item type %q: %v", c.deleteID, err))
				return false
			}
		}
	}

	return true
}

// workItemTypeSyncResult carries the work item types syncWorkItemTypesIfConfigured has on hand.
// types is only meaningful when known is true; when known is false the caller must re-fetch,
// because the list either was never read or could not be confirmed to reflect the mutation.
// A known-good list may legitimately be nil (the API can return null for an empty list), which
// is why this is a separate flag rather than a nil check on types.
type workItemTypeSyncResult struct {
	types []youtrack.WorkItemType
	known bool
}

// syncWorkItemTypesIfConfigured reconciles work item types in YouTrack against the plan.
// It is a no-op when work_item_types is null or unknown in the plan.
//
// On success it also returns the most up-to-date work item types list it has on hand — the
// pre-mutation list when no changes were needed, or the freshly settled post-mutation list
// otherwise — so callers can reuse it instead of re-fetching.
func (r *globalTimeTrackingSettingsResource) syncWorkItemTypesIfConfigured(ctx context.Context, plan *globalTimeTrackingSettingsResourceModel, diagnostics *diag.Diagnostics) (workItemTypeSyncResult, bool) {
	if plan.WorkItemTypes.IsNull() || plan.WorkItemTypes.IsUnknown() {
		return workItemTypeSyncResult{}, true
	}

	var planTypes []globalWorkItemTypeModel
	if diags := plan.WorkItemTypes.ElementsAs(ctx, &planTypes, false); diags.HasError() {
		diagnostics.Append(diags...)
		return workItemTypeSyncResult{}, false
	}

	currentTypes, err := r.listWorkItemTypesWithRetry(ctx)
	if err != nil {
		diagnostics.AddError(errManagingWorkItemTypes, err.Error())
		return workItemTypeSyncResult{}, false
	}

	changes, err := planWorkItemTypeChanges(planTypes, currentTypes)
	if err != nil {
		diagnostics.AddError(errManagingWorkItemTypes, err.Error())
		return workItemTypeSyncResult{}, false
	}

	if !r.applyWorkItemTypeChanges(ctx, changes, diagnostics) {
		return workItemTypeSyncResult{}, false
	}

	settledTypes, settled := r.waitForWorkItemTypeChangesToSettle(ctx, currentTypes, changes, diagnostics)

	return workItemTypeSyncResult{types: settledTypes, known: settled}, true
}

// deletedWorkItemTypeIDs collects the IDs slated for deletion across the given changes.
func deletedWorkItemTypeIDs(changes []workItemTypeChange) map[string]struct{} {
	deletedIDs := make(map[string]struct{})
	for _, c := range changes {
		if c.deleteID != "" {
			deletedIDs[c.deleteID] = struct{}{}
		}
	}
	return deletedIDs
}

// expectedWorkItemTypesByName collects the name -> auto_attached values that created or
// updated work item types are expected to have once the API reflects the change.
func expectedWorkItemTypesByName(changes []workItemTypeChange) map[string]bool {
	expected := make(map[string]bool)
	for _, c := range changes {
		switch {
		case c.create != nil:
			expected[c.create.Name] = c.create.AutoAttached
		case c.update != nil:
			expected[c.update.Name] = c.update.AutoAttached
		}
	}
	return expected
}

// anyWorkItemTypeIDPresent reports whether any of the given types has an ID in ids.
func anyWorkItemTypeIDPresent(currentTypes []youtrack.WorkItemType, ids map[string]struct{}) bool {
	for _, ct := range currentTypes {
		if _, present := ids[ct.ID]; present {
			return true
		}
	}
	return false
}

// workItemTypeChangesSettled reports whether currentTypes reflects all of the given changes:
// deleted IDs are gone, and created/updated types are present with the expected auto_attached value.
func workItemTypeChangesSettled(currentTypes []youtrack.WorkItemType, deletedIDs map[string]struct{}, expectedByName map[string]bool) bool {
	if anyWorkItemTypeIDPresent(currentTypes, deletedIDs) {
		return false
	}

	currentByName := buildCurrentWorkItemTypesByName(currentTypes)
	for name, autoAttached := range expectedByName {
		existing, ok := currentByName[name]
		if !ok || existing.AutoAttached != autoAttached {
			return false
		}
	}

	return true
}

// waitForWorkItemTypeChangesToSettle polls the API until it reflects the given create/update/delete
// changes, guarding against the API briefly returning stale data right after a mutation. It keeps
// polling for any retryable read failure (see isRetryableWorkItemTypeListError) as well as the
// "listed fine but not settled yet" case, and fails fast only on a definitive client error, rather
// than burning the whole budget on an error that can never resolve.
//
// The changes were already applied successfully by the time this runs, so a failure to confirm them
// is reported as a warning, not an error: failing the apply here would report a false failure for a
// mutation that already succeeded, and leave YouTrack and Terraform state out of sync until the next
// refresh re-attempts a no-longer-needed create/delete.
//
// The returned bool — not the returned slice — says whether the list is usable: true with
// preChangeTypes when there was nothing to settle, true with the freshly confirmed list on success,
// and false when settling could not be confirmed and the caller must re-fetch instead. A successful
// list can itself be nil (the API may return null for an empty list), so callers must not infer
// "could not confirm" from a nil slice.
func (r *globalTimeTrackingSettingsResource) waitForWorkItemTypeChangesToSettle(ctx context.Context, preChangeTypes []youtrack.WorkItemType, changes []workItemTypeChange, diagnostics *diag.Diagnostics) ([]youtrack.WorkItemType, bool) {
	deletedIDs := deletedWorkItemTypeIDs(changes)
	expectedByName := expectedWorkItemTypesByName(changes)
	if len(deletedIDs) == 0 && len(expectedByName) == 0 {
		return preChangeTypes, true
	}

	var lastErr error
	for attempt := 1; attempt <= workItemTypeSettleMaxAttempts; attempt++ {
		currentTypes, err := r.client.ListWorkItemTypes(ctx)
		switch {
		case err == nil:
			if workItemTypeChangesSettled(currentTypes, deletedIDs, expectedByName) {
				return currentTypes, true
			}
			lastErr = nil
		case isRetryableWorkItemTypeListError(err):
			lastErr = err
		default:
			diagnostics.AddWarning(warnWorkItemTypesNotConfirmed,
				fmt.Sprintf("could not confirm work item type changes were reflected by the API; work_item_types may show drift until the next refresh: %v", err))
			return nil, false
		}

		if attempt == workItemTypeSettleMaxAttempts {
			break
		}

		if err := helpers.WaitOrContextDone(ctx, workItemTypePollRetryDelay); err != nil {
			diagnostics.AddWarning(warnWorkItemTypesNotConfirmed,
				fmt.Sprintf("stopped waiting for work item type changes to be reflected by the API: %v", err))
			return nil, false
		}
	}

	detail := "timed out waiting for work item type changes to be reflected by the API; work_item_types may show drift until the next refresh"
	if lastErr != nil {
		detail = fmt.Sprintf("%s (last read error: %v)", detail, lastErr)
	}
	diagnostics.AddWarning(warnWorkItemTypesNotConfirmed, detail)
	return nil, false
}
