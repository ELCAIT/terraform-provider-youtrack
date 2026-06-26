package provider

import (
	"strings"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (m *enumBundleResourceModel) toAPIModel() youtrack.EnumBundle {
	values := mapBundleValues(m.Values, func(value enumBundleValueModel) youtrack.EnumBundleElement {
		return value.toAPIModel()
	})

	return youtrack.EnumBundle{
		Name:   m.Name.ValueString(),
		Values: values,
	}
}

func (m *enumBundleResourceModel) toAPIModelPreservingExisting(current *youtrack.EnumBundle) youtrack.EnumBundle {
	values := mergeBundleValuesPreservingExisting(m.Values, current.Values, bundleValueMergeOps[enumBundleValueModel, youtrack.EnumBundleElement]{
		toAPI: func(value enumBundleValueModel) youtrack.EnumBundleElement {
			return value.toAPIModel()
		},
		modelID: func(value enumBundleValueModel) string {
			return helpers.StringFromOptional(value.ID)
		},
		apiID: func(item youtrack.EnumBundleElement) string {
			return item.ID
		},
		setAPIID: func(item *youtrack.EnumBundleElement, id string) {
			item.ID = id
		},
		apiName: func(item youtrack.EnumBundleElement) string {
			return item.Name
		},
	})

	return youtrack.EnumBundle{
		Name:   m.Name.ValueString(),
		Values: values,
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
	return unexpectedBundleValueNames(
		plan.Values,
		updated.Values,
		func(value enumBundleValueModel) string { return value.Name.ValueString() },
		func(value youtrack.EnumBundleElement) string { return value.Name },
	)
}

func (m *enumBundleResourceModel) fromAPIModel(apiModel *youtrack.EnumBundle) {
	m.ID = types.StringValue(apiModel.ID)
	m.Name = types.StringValue(apiModel.Name)
	m.IsUpdateable = types.BoolValue(apiModel.IsUpdateable)

	values := mapBundleValues(apiModel.Values, func(value youtrack.EnumBundleElement) enumBundleValueModel {
		return enumBundleValueModel{
			ID:            types.StringValue(value.ID),
			Name:          types.StringValue(value.Name),
			LocalizedName: helpers.StringOrNull(value.LocalizedName),
			Description:   helpers.StringOrNull(value.Description),
			Archived:      types.BoolValue(value.Archived),
			Ordinal:       types.Int64Value(int64(value.Ordinal)),
		}
	})
	m.Values = values
}
