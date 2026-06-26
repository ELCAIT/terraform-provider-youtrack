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
	values := mergeBundleValuesPreservingExisting(m.Values, current.Values, bundleValueMergeOps[stateBundleValueModel, youtrack.StateBundleElement]{
		toAPI: func(value stateBundleValueModel) youtrack.StateBundleElement {
			return value.toAPIModel()
		},
		modelID: func(value stateBundleValueModel) string {
			return helpers.StringFromOptional(value.ID)
		},
		apiID: func(item youtrack.StateBundleElement) string {
			return item.ID
		},
		setAPIID: func(item *youtrack.StateBundleElement, id string) {
			item.ID = id
		},
		apiName: func(item youtrack.StateBundleElement) string {
			return item.Name
		},
	})

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
