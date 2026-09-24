package provider

import (
	"context"
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

func (m *enumBundleValueModel) toAPIModel() youtrack.EnumBundleElement {
	item := youtrack.EnumBundleElement{
		Name:     m.Name.ValueString(),
		Archived: helpers.BoolFromOptional(m.Archived),
	}
	item.Description = helpers.StringFromOptional(m.Description)
	item.LocalizedName = helpers.StringFromOptional(m.LocalizedName)
	return item
}

// toValueUpdate builds the full replacement for an existing value. Optional
// attributes the plan leaves unknown (unset in configuration) keep what
// YouTrack has.
func (m *enumBundleValueModel) toValueUpdate(current youtrack.EnumBundleElement, ordinal int) youtrack.EnumBundleValueUpdate {
	return youtrack.EnumBundleValueUpdate{
		Name:          m.Name.ValueString(),
		LocalizedName: optionalStringPointer(m.LocalizedName, current.LocalizedName),
		Description:   optionalStringPointer(m.Description, current.Description),
		Archived:      helpers.BoolFromOptional(m.Archived),
		Ordinal:       &ordinal,
	}
}

func enumValueAsUpdate(current youtrack.EnumBundleElement) youtrack.EnumBundleValueUpdate {
	ordinal := current.Ordinal
	return youtrack.EnumBundleValueUpdate{
		Name:          current.Name,
		LocalizedName: stringPointerOrNil(current.LocalizedName),
		Description:   stringPointerOrNil(current.Description),
		Archived:      current.Archived,
		Ordinal:       &ordinal,
	}
}

func (r *enumBundleResource) valueReconciler(bundleID string) bundleValueReconciler[enumBundleValueModel, youtrack.EnumBundleElement, youtrack.EnumBundleValueUpdate] {
	return bundleValueReconciler[enumBundleValueModel, youtrack.EnumBundleElement, youtrack.EnumBundleValueUpdate]{
		plannedName: func(value enumBundleValueModel) string { return value.Name.ValueString() },
		apiID:       func(value youtrack.EnumBundleElement) string { return value.ID },
		apiName:     func(value youtrack.EnumBundleElement) string { return value.Name },
		toUpdate: func(value enumBundleValueModel, current youtrack.EnumBundleElement, ordinal int) youtrack.EnumBundleValueUpdate {
			return value.toValueUpdate(current, ordinal)
		},
		currentAsUpdate: enumValueAsUpdate,
		add: func(ctx context.Context, value enumBundleValueModel, ordinal int) error {
			element := value.toAPIModel()
			element.Ordinal = ordinal
			_, err := r.client.AddEnumBundleValue(ctx, bundleID, element)
			return err
		},
		replace: func(ctx context.Context, valueID string, update youtrack.EnumBundleValueUpdate) error {
			_, err := r.client.ReplaceEnumBundleValue(ctx, bundleID, valueID, update)
			return err
		},
		remove: func(ctx context.Context, valueID string) error {
			return r.client.DeleteEnumBundleValue(ctx, bundleID, valueID)
		},
	}
}

// reconcile brings an existing enum bundle in line with the plan, editing
// values one by one so that each keeps its ID (see bundleValueReconciler).
func (r *enumBundleResource) reconcile(ctx context.Context, plan enumBundleResourceModel) (*youtrack.EnumBundle, error) {
	bundleID := plan.ID.ValueString()

	current, err := r.client.GetEnumBundleByID(ctx, bundleID)
	if err != nil {
		return nil, err
	}

	if current.Name != plan.Name.ValueString() {
		if _, err := r.client.UpdateEnumBundle(ctx, bundleID, youtrack.EnumBundle{Name: plan.Name.ValueString()}); err != nil {
			return nil, err
		}
	}

	if err := r.valueReconciler(bundleID).reconcile(ctx, plan.Values, current.Values); err != nil {
		return nil, err
	}

	return r.client.GetEnumBundleByID(ctx, bundleID)
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

	values := sortedByOrdinal(apiModel.Values, func(value youtrack.EnumBundleElement) int { return value.Ordinal })
	m.Values = mapBundleValues(values, func(value youtrack.EnumBundleElement) enumBundleValueModel {
		return enumBundleValueModel{
			ID:            types.StringValue(value.ID),
			Name:          types.StringValue(value.Name),
			LocalizedName: helpers.StringOrNull(value.LocalizedName),
			Description:   helpers.StringOrNull(value.Description),
			Archived:      types.BoolValue(value.Archived),
			Ordinal:       types.Int64Value(int64(value.Ordinal)),
		}
	})
}
