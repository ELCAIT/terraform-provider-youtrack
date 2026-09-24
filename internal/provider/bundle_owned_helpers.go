package provider

import (
	"context"
	"fmt"
	"strings"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ownerRefs maps a normalized owner login to the YouTrack user it resolved to.
type ownerRefs map[string]*youtrack.UserRef

func normalizeOwnerLogin(login string) string {
	return strings.ToLower(strings.TrimSpace(login))
}

// resolveOwners looks up every distinct owner_login in values once, so a user
// who owns several values costs a single lookup.
func (r *ownedBundleResource) resolveOwners(ctx context.Context, values []ownedBundleValueModel) (ownerRefs, error) {
	owners := make(ownerRefs, len(values))
	for _, value := range values {
		login := helpers.StringFromOptional(value.OwnerLogin)
		key := normalizeOwnerLogin(login)
		if key == "" {
			continue
		}
		if _, done := owners[key]; done {
			continue
		}

		user, err := r.client.GetUserByLogin(ctx, login)
		if err != nil {
			return nil, fmt.Errorf("could not find owner with login %q: %w", login, err)
		}
		owners[key] = &youtrack.UserRef{ID: user.Id}
	}

	return owners, nil
}

func (o ownerRefs) forLogin(login types.String) *youtrack.UserRef {
	return o[normalizeOwnerLogin(helpers.StringFromOptional(login))]
}

func (m *ownedBundleResourceModel) toAPIModel(owners ownerRefs) youtrack.OwnedBundle {
	return youtrack.OwnedBundle{
		Name: m.Name.ValueString(),
		Values: mapBundleValues(m.Values, func(value ownedBundleValueModel) youtrack.OwnedBundleElement {
			return value.toAPIModel(owners)
		}),
	}
}

func (m *ownedBundleValueModel) toAPIModel(owners ownerRefs) youtrack.OwnedBundleElement {
	return youtrack.OwnedBundleElement{
		Name:        m.Name.ValueString(),
		Description: helpers.StringFromOptional(m.Description),
		Archived:    helpers.BoolFromOptional(m.Archived),
		Owner:       owners.forLogin(m.OwnerLogin),
	}
}

// toValueUpdate builds the full replacement for an existing value. A
// description left unknown by the plan (unset in config) keeps the one YouTrack
// has, since the attribute is Optional+Computed.
func (m *ownedBundleValueModel) toValueUpdate(current youtrack.OwnedBundleElement, owners ownerRefs, ordinal int) youtrack.OwnedBundleValueUpdate {
	return youtrack.OwnedBundleValueUpdate{
		Name:        m.Name.ValueString(),
		Description: optionalStringPointer(m.Description, current.Description),
		Archived:    helpers.BoolFromOptional(m.Archived),
		Ordinal:     &ordinal,
		Owner:       owners.forLogin(m.OwnerLogin),
	}
}

func ownedValueAsUpdate(current youtrack.OwnedBundleElement) youtrack.OwnedBundleValueUpdate {
	ordinal := current.Ordinal
	update := youtrack.OwnedBundleValueUpdate{
		Name:        current.Name,
		Description: stringPointerOrNil(current.Description),
		Archived:    current.Archived,
		Ordinal:     &ordinal,
	}
	if current.Owner != nil {
		update.Owner = &youtrack.UserRef{ID: current.Owner.ID}
	}

	return update
}

func (r *ownedBundleResource) valueReconciler(bundleID string, owners ownerRefs) bundleValueReconciler[ownedBundleValueModel, youtrack.OwnedBundleElement, youtrack.OwnedBundleValueUpdate] {
	return bundleValueReconciler[ownedBundleValueModel, youtrack.OwnedBundleElement, youtrack.OwnedBundleValueUpdate]{
		plannedName: func(value ownedBundleValueModel) string { return value.Name.ValueString() },
		apiID:       func(value youtrack.OwnedBundleElement) string { return value.ID },
		apiName:     func(value youtrack.OwnedBundleElement) string { return value.Name },
		toUpdate: func(value ownedBundleValueModel, current youtrack.OwnedBundleElement, ordinal int) youtrack.OwnedBundleValueUpdate {
			return value.toValueUpdate(current, owners, ordinal)
		},
		currentAsUpdate: ownedValueAsUpdate,
		add: func(ctx context.Context, value ownedBundleValueModel, ordinal int) error {
			element := value.toAPIModel(owners)
			element.Ordinal = ordinal
			_, err := r.client.AddOwnedBundleValue(ctx, bundleID, element)
			return err
		},
		replace: func(ctx context.Context, valueID string, update youtrack.OwnedBundleValueUpdate) error {
			_, err := r.client.ReplaceOwnedBundleValue(ctx, bundleID, valueID, update)
			return err
		},
		remove: func(ctx context.Context, valueID string) error {
			return r.client.DeleteOwnedBundleValue(ctx, bundleID, valueID)
		},
	}
}

// reconcile brings an existing owned bundle in line with the plan, editing
// values one by one so that each keeps its ID (see bundleValueReconciler).
func (r *ownedBundleResource) reconcile(ctx context.Context, plan ownedBundleResourceModel) (*youtrack.OwnedBundle, error) {
	bundleID := plan.ID.ValueString()

	current, err := r.client.GetOwnedBundleByID(ctx, bundleID)
	if err != nil {
		return nil, err
	}

	owners, err := r.resolveOwners(ctx, plan.Values)
	if err != nil {
		return nil, err
	}

	if current.Name != plan.Name.ValueString() {
		if _, err := r.client.UpdateOwnedBundle(ctx, bundleID, youtrack.OwnedBundle{Name: plan.Name.ValueString()}); err != nil {
			return nil, err
		}
	}

	if err := r.valueReconciler(bundleID, owners).reconcile(ctx, plan.Values, current.Values); err != nil {
		return nil, err
	}

	return r.client.GetOwnedBundleByID(ctx, bundleID)
}

func unexpectedOwnedValueNames(plan ownedBundleResourceModel, updated *youtrack.OwnedBundle) []string {
	return unexpectedBundleValueNames(
		plan.Values,
		updated.Values,
		func(value ownedBundleValueModel) string { return value.Name.ValueString() },
		func(value youtrack.OwnedBundleElement) string { return value.Name },
	)
}

// fromAPIModel maps the bundle into state, ordered by ordinal. prior supplies
// the owner_login spelling from configuration: YouTrack matches logins
// case-insensitively and returns its own spelling, which would otherwise read
// as a change.
func (m *ownedBundleResourceModel) fromAPIModel(apiModel *youtrack.OwnedBundle, prior []ownedBundleValueModel) {
	m.ID = types.StringValue(apiModel.ID)
	m.Name = types.StringValue(apiModel.Name)
	m.IsUpdateable = types.BoolValue(apiModel.IsUpdateable)

	priorLogins := make(map[string]string, len(prior))
	for _, value := range prior {
		priorLogins[normalizeBundleValueName(value.Name.ValueString())] = helpers.StringFromOptional(value.OwnerLogin)
	}

	values := sortedByOrdinal(apiModel.Values, func(value youtrack.OwnedBundleElement) int { return value.Ordinal })

	m.Values = mapBundleValues(values, func(value youtrack.OwnedBundleElement) ownedBundleValueModel {
		return ownedBundleValueModel{
			ID:          types.StringValue(value.ID),
			Name:        types.StringValue(value.Name),
			Description: helpers.StringOrNull(value.Description),
			Archived:    types.BoolValue(value.Archived),
			Ordinal:     types.Int64Value(int64(value.Ordinal)),
			OwnerLogin:  ownerLoginValue(value.Owner, priorLogins[normalizeBundleValueName(value.Name)]),
		}
	})
}

func ownerLoginValue(owner *youtrack.UserRef, priorLogin string) types.String {
	if owner == nil || owner.Login == "" {
		return types.StringNull()
	}
	if strings.EqualFold(owner.Login, priorLogin) {
		return types.StringValue(priorLogin)
	}
	return types.StringValue(owner.Login)
}
