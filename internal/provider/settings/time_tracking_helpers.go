package settings

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	errUnableToReadTimeTracking     = "Unable to Read YouTrack Global Time Tracking Settings"
	errUpdatingWorkTimeSettings     = "Error updating work time settings"
	errConvertingTimeTracking       = "Error converting time tracking settings"
	errConvertingWorkDays           = "Failed to convert work_days list"
	errConvertingNestedTimeTracking = "Failed to convert nested time tracking attributes"
	errManagingWorkItemTypes        = "Error managing work item types"
	errWorkItemTypeEmptyName        = "each work_item_types entry must have a non-empty name"
	globalTimeTrackingSingletonID   = "global"
	// workItemTypeBeingRemovedSuffix is appended by YouTrack when a work item type
	// is soft-deleted. The provider filters these out so they never appear in state.
	workItemTypeBeingRemovedSuffix = " (being removed)"
	defaultWorkMinutesADay         = 480
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
	settings, err := r.client.GetGlobalTimeTrackingSettings(ctx)
	if err != nil {
		diagnostics.AddError(
			errUnableToReadTimeTracking,
			err.Error(),
		)
		return youtrack.GlobalTimeTrackingSettings{}, false
	}

	return settings, true
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

	if !r.syncWorkItemTypesIfConfigured(ctx, plan, diagnostics) {
		return false
	}

	globalSettings, ok := r.getGlobalTimeTrackingSettingsAndHandleError(ctx, diagnostics)
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

// syncWorkItemTypesIfConfigured reconciles work item types in YouTrack against the plan.
// It is a no-op when work_item_types is null or unknown in the plan.
func (r *globalTimeTrackingSettingsResource) syncWorkItemTypesIfConfigured(ctx context.Context, plan *globalTimeTrackingSettingsResourceModel, diagnostics *diag.Diagnostics) bool {
	if plan.WorkItemTypes.IsNull() || plan.WorkItemTypes.IsUnknown() {
		return true
	}

	var planTypes []globalWorkItemTypeModel
	if diags := plan.WorkItemTypes.ElementsAs(ctx, &planTypes, false); diags.HasError() {
		diagnostics.Append(diags...)
		return false
	}

	currentTypes, err := r.client.ListWorkItemTypes(ctx)
	if err != nil {
		diagnostics.AddError(errManagingWorkItemTypes, err.Error())
		return false
	}

	changes, err := planWorkItemTypeChanges(planTypes, currentTypes)
	if err != nil {
		diagnostics.AddError(errManagingWorkItemTypes, err.Error())
		return false
	}

	return r.applyWorkItemTypeChanges(ctx, changes, diagnostics)
}
