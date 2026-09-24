package provider

import (
	"context"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"
	youtrack "github.com/elcait/youtrack-api-client/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (m *stateBundleResourceModel) toAPIModel() youtrack.StateBundle {
	values := mapBundleValues(m.Values, func(value stateBundleValueModel) youtrack.StateBundleElement {
		return value.toAPIModel()
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
	item.Description = helpers.StringFromOptional(m.Description)
	item.LocalizedName = helpers.StringFromOptional(m.LocalizedName)
	return item
}

// toValueUpdate builds the full replacement for an existing value. Optional
// attributes the plan leaves unknown (unset in configuration) keep what
// YouTrack has.
func (m *stateBundleValueModel) toValueUpdate(current youtrack.StateBundleElement, ordinal int) youtrack.StateBundleValueUpdate {
	return youtrack.StateBundleValueUpdate{
		Name:          m.Name.ValueString(),
		LocalizedName: optionalStringPointer(m.LocalizedName, current.LocalizedName),
		Description:   optionalStringPointer(m.Description, current.Description),
		IsResolved:    helpers.BoolFromOptional(m.IsResolved),
		Archived:      helpers.BoolFromOptional(m.Archived),
		Ordinal:       &ordinal,
	}
}

func stateValueAsUpdate(current youtrack.StateBundleElement) youtrack.StateBundleValueUpdate {
	ordinal := current.Ordinal
	return youtrack.StateBundleValueUpdate{
		Name:          current.Name,
		LocalizedName: stringPointerOrNil(current.LocalizedName),
		Description:   stringPointerOrNil(current.Description),
		IsResolved:    current.IsResolved,
		Archived:      current.Archived,
		Ordinal:       &ordinal,
	}
}

func (r *stateBundleResource) valueReconciler(bundleID string) bundleValueReconciler[stateBundleValueModel, youtrack.StateBundleElement, youtrack.StateBundleValueUpdate] {
	return bundleValueReconciler[stateBundleValueModel, youtrack.StateBundleElement, youtrack.StateBundleValueUpdate]{
		plannedName: func(value stateBundleValueModel) string { return value.Name.ValueString() },
		apiID:       func(value youtrack.StateBundleElement) string { return value.ID },
		apiName:     func(value youtrack.StateBundleElement) string { return value.Name },
		toUpdate: func(value stateBundleValueModel, current youtrack.StateBundleElement, ordinal int) youtrack.StateBundleValueUpdate {
			return value.toValueUpdate(current, ordinal)
		},
		currentAsUpdate: stateValueAsUpdate,
		add: func(ctx context.Context, value stateBundleValueModel, ordinal int) error {
			element := value.toAPIModel()
			element.Ordinal = ordinal
			_, err := r.client.AddStateBundleValue(ctx, bundleID, element)
			return err
		},
		replace: func(ctx context.Context, valueID string, update youtrack.StateBundleValueUpdate) error {
			_, err := r.client.ReplaceStateBundleValue(ctx, bundleID, valueID, update)
			return err
		},
		remove: func(ctx context.Context, valueID string) error {
			return r.client.DeleteStateBundleValue(ctx, bundleID, valueID)
		},
	}
}

// reconcile brings an existing state bundle in line with the plan, editing
// values one by one so that each keeps its ID (see bundleValueReconciler).
func (r *stateBundleResource) reconcile(ctx context.Context, plan stateBundleResourceModel) (*youtrack.StateBundle, error) {
	bundleID := plan.ID.ValueString()

	current, err := r.client.GetStateBundleByID(ctx, bundleID)
	if err != nil {
		return nil, err
	}

	if current.Name != plan.Name.ValueString() {
		if _, err := r.client.UpdateStateBundle(ctx, bundleID, youtrack.StateBundle{Name: plan.Name.ValueString()}); err != nil {
			return nil, err
		}
	}

	if err := r.valueReconciler(bundleID).reconcile(ctx, plan.Values, current.Values); err != nil {
		return nil, err
	}

	return r.client.GetStateBundleByID(ctx, bundleID)
}

func unexpectedStateValueNames(plan stateBundleResourceModel, updated *youtrack.StateBundle) []string {
	return unexpectedBundleValueNames(
		plan.Values,
		updated.Values,
		func(value stateBundleValueModel) string { return value.Name.ValueString() },
		func(value youtrack.StateBundleElement) string { return value.Name },
	)
}

func (m *stateBundleResourceModel) fromAPIModel(apiModel *youtrack.StateBundle) {
	m.ID = types.StringValue(apiModel.ID)
	m.Name = types.StringValue(apiModel.Name)
	m.IsUpdateable = types.BoolValue(apiModel.IsUpdateable)

	values := sortedByOrdinal(apiModel.Values, func(value youtrack.StateBundleElement) int { return value.Ordinal })
	m.Values = mapBundleValues(values, func(value youtrack.StateBundleElement) stateBundleValueModel {
		return stateBundleValueModel{
			ID:            types.StringValue(value.ID),
			Name:          types.StringValue(value.Name),
			LocalizedName: helpers.StringOrNull(value.LocalizedName),
			Description:   helpers.StringOrNull(value.Description),
			IsResolved:    types.BoolValue(value.IsResolved),
			Archived:      types.BoolValue(value.Archived),
			Ordinal:       types.Int64Value(int64(value.Ordinal)),
		}
	})
}
