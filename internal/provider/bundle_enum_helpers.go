package provider

import (
	"sort"
	"strings"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (m *enumBundleResourceModel) toAPIModel() youtrack.EnumBundle {
	values := make([]youtrack.EnumBundleElement, 0, len(m.Values))
	for _, value := range m.Values {
		values = append(values, value.toAPIModel())
	}

	return youtrack.EnumBundle{
		Name:   m.Name.ValueString(),
		Values: values,
	}
}

func (m *enumBundleResourceModel) toAPIModelPreservingExisting(current *youtrack.EnumBundle) youtrack.EnumBundle {
	plannedByID := make(map[string]youtrack.EnumBundleElement, len(m.Values))
	plannedWithoutIDByName := make(map[string]youtrack.EnumBundleElement, len(m.Values))
	plannedWithoutID := make([]youtrack.EnumBundleElement, 0, len(m.Values))

	collectPlannedEnumValues(m.Values, plannedByID, plannedWithoutIDByName, &plannedWithoutID)

	values := make([]youtrack.EnumBundleElement, 0, len(current.Values)+len(plannedWithoutID))
	for _, existing := range current.Values {
		if planned, ok := plannedByID[existing.ID]; ok {
			values = append(values, planned)
			delete(plannedByID, existing.ID)
			continue
		}

		normalizedExistingName := normalizeBundleValueName(existing.Name)
		if planned, ok := plannedWithoutIDByName[normalizedExistingName]; ok {
			planned.ID = existing.ID
			values = append(values, planned)
			delete(plannedWithoutIDByName, normalizedExistingName)
			continue
		}

		values = append(values, existing)
	}

	appendRemainingPlannedEnumValuesByID(m.Values, plannedByID, &values)

	for _, planned := range plannedWithoutID {
		normalizedName := normalizeBundleValueName(planned.Name)
		if _, ok := plannedWithoutIDByName[normalizedName]; !ok {
			continue
		}
		values = append(values, planned)
		delete(plannedWithoutIDByName, normalizedName)
	}

	return youtrack.EnumBundle{
		Name:   m.Name.ValueString(),
		Values: values,
	}
}

func collectPlannedEnumValues(
	modelValues []enumBundleValueModel,
	plannedByID map[string]youtrack.EnumBundleElement,
	plannedWithoutIDByName map[string]youtrack.EnumBundleElement,
	plannedWithoutID *[]youtrack.EnumBundleElement,
) {
	for _, value := range modelValues {
		item := value.toAPIModel()
		if item.ID == "" {
			plannedWithoutIDByName[normalizeBundleValueName(item.Name)] = item
			*plannedWithoutID = append(*plannedWithoutID, item)
			continue
		}
		plannedByID[item.ID] = item
	}
}

func appendRemainingPlannedEnumValuesByID(
	modelValues []enumBundleValueModel,
	plannedByID map[string]youtrack.EnumBundleElement,
	values *[]youtrack.EnumBundleElement,
) {
	for _, value := range modelValues {
		plannedID := helpers.StringFromOptional(value.ID)
		if plannedID == "" {
			continue
		}
		planned, ok := plannedByID[plannedID]
		if !ok {
			continue
		}
		*values = append(*values, planned)
	}
}

func (m *enumBundleValueModel) toAPIModel() youtrack.EnumBundleElement {
	item := youtrack.EnumBundleElement{
		Name:     m.Name.ValueString(),
		Archived: helpers.BoolFromOptional(m.Archived),
	}
	item.ID = helpers.StringFromOptional(m.ID)
	item.Description = helpers.StringFromOptional(m.Description)
	item.LocalizedName = helpers.StringFromOptional(m.LocalizedName)
	return item
}

func isRequiredCustomFieldWorkflowError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	hasRule := strings.Contains(errMsg, "@jetbrains/required-custom-fields-feature")
	hasFieldRequired := strings.Contains(errMsg, "field required") || strings.Contains(errMsg, " is required")
	hasWorkflowType := strings.Contains(errMsg, "\"error_type\":\"workflow\"")

	return hasRule || (hasFieldRequired && hasWorkflowType)
}

func normalizeBundleValueName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func unexpectedEnumValueNames(plan enumBundleResourceModel, updated *youtrack.EnumBundle) []string {
	plannedByName := make(map[string]struct{}, len(plan.Values))
	for _, value := range plan.Values {
		plannedByName[normalizeBundleValueName(value.Name.ValueString())] = struct{}{}
	}

	unexpected := make([]string, 0)
	for _, value := range updated.Values {
		normalizedName := normalizeBundleValueName(value.Name)
		if _, ok := plannedByName[normalizedName]; ok {
			continue
		}
		unexpected = append(unexpected, value.Name)
	}

	sort.Strings(unexpected)
	return unexpected
}

func (m *enumBundleResourceModel) fromAPIModel(apiModel *youtrack.EnumBundle) {
	m.ID = types.StringValue(apiModel.ID)
	m.Name = types.StringValue(apiModel.Name)
	m.IsUpdateable = types.BoolValue(apiModel.IsUpdateable)

	values := make([]enumBundleValueModel, 0, len(apiModel.Values))
	for _, value := range apiModel.Values {
		values = append(values, enumBundleValueModel{
			ID:            types.StringValue(value.ID),
			Name:          types.StringValue(value.Name),
			LocalizedName: helpers.StringOrNull(value.LocalizedName),
			Description:   helpers.StringOrNull(value.Description),
			Archived:      types.BoolValue(value.Archived),
			Ordinal:       types.Int64Value(int64(value.Ordinal)),
		})
	}
	m.Values = values
}
