package provider

import (
	"sort"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"
	youtrack "github.com/elcait/youtrack-api-client/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (m *stateBundleResourceModel) toAPIModel() youtrack.StateBundle {
	values := make([]youtrack.StateBundleElement, 0, len(m.Values))
	for _, value := range m.Values {
		values = append(values, value.toAPIModel())
	}

	return youtrack.StateBundle{
		Name:   m.Name.ValueString(),
		Values: values,
	}
}

func (m *stateBundleResourceModel) toAPIModelPreservingExisting(current *youtrack.StateBundle) youtrack.StateBundle {
	plannedByID := make(map[string]youtrack.StateBundleElement, len(m.Values))
	plannedWithoutIDByName := make(map[string]youtrack.StateBundleElement, len(m.Values))
	plannedWithoutID := make([]youtrack.StateBundleElement, 0, len(m.Values))

	for _, value := range m.Values {
		item := value.toAPIModel()
		if item.ID == "" {
			plannedWithoutIDByName[normalizeBundleValueName(item.Name)] = item
			plannedWithoutID = append(plannedWithoutID, item)
			continue
		}
		plannedByID[item.ID] = item
	}

	values := make([]youtrack.StateBundleElement, 0, len(current.Values)+len(plannedWithoutID))
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

	for _, value := range m.Values {
		plannedID := helpers.StringFromOptional(value.ID)
		if plannedID == "" {
			continue
		}
		planned, ok := plannedByID[plannedID]
		if !ok {
			continue
		}
		values = append(values, planned)
	}
	for _, planned := range plannedWithoutID {
		normalizedName := normalizeBundleValueName(planned.Name)
		if _, ok := plannedWithoutIDByName[normalizedName]; !ok {
			continue
		}
		values = append(values, planned)
		delete(plannedWithoutIDByName, normalizedName)
	}

	return youtrack.StateBundle{
		Name:   m.Name.ValueString(),
		Values: values,
	}
}

func (m *stateBundleValueModel) toAPIModel() youtrack.StateBundleElement {
	item := youtrack.StateBundleElement{
		Name:       m.Name.ValueString(),
		IsResolved: helpers.BoolFromOptional(m.IsResolved),
		Archived:   helpers.BoolFromOptional(m.Archived),
	}
	item.ID = helpers.StringFromOptional(m.ID)
	item.Description = helpers.StringFromOptional(m.Description)
	item.LocalizedName = helpers.StringFromOptional(m.LocalizedName)
	return item
}

func unexpectedStateValueNames(plan stateBundleResourceModel, updated *youtrack.StateBundle) []string {
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

func (m *stateBundleResourceModel) fromAPIModel(apiModel *youtrack.StateBundle) {
	m.ID = types.StringValue(apiModel.ID)
	m.Name = types.StringValue(apiModel.Name)
	m.IsUpdateable = types.BoolValue(apiModel.IsUpdateable)

	values := make([]stateBundleValueModel, 0, len(apiModel.Values))
	for _, value := range apiModel.Values {
		values = append(values, stateBundleValueModel{
			ID:            types.StringValue(value.ID),
			Name:          types.StringValue(value.Name),
			LocalizedName: helpers.StringOrNull(value.LocalizedName),
			Description:   helpers.StringOrNull(value.Description),
			IsResolved:    types.BoolValue(value.IsResolved),
			Archived:      types.BoolValue(value.Archived),
			Ordinal:       types.Int64Value(int64(value.Ordinal)),
		})
	}
	m.Values = values
}
