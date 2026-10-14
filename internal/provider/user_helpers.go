// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	errCreatingUser        = "Error creating user"
	errReadingUserResource = "Error reading user"
	errUpdatingUser        = "Error updating user"
	errDeletingUser        = "Error deleting user"
	errMissingUserID       = "Missing user ID"
	errUserIDRequired      = "User ID is required to read the user"
)

// applyUserUpdate writes the account fields and, when it changed, the banned flag.
// The flag needs its own call: User.Banned is omitempty, so UpdateUser cannot unban.
func (r *userResource) applyUserUpdate(ctx context.Context, plan, prior userResourceModel) (*youtrack.User, error) {
	updated, err := r.client.UpdateUser(ctx, plan.ID.ValueString(), plan.toUpdateAPIModel(prior))
	if err != nil {
		return nil, err
	}

	if plan.Banned.Equal(prior.Banned) {
		return updated, nil
	}

	return r.client.SetUserBanned(ctx, plan.ID.ValueString(), plan.Banned.ValueBool())
}

func (m *userResourceModel) toAPIModel() youtrack.User {
	password, _ := helpers.OptionalStringValue(m.Password)

	return youtrack.User{
		Login:    m.Login.ValueString(),
		FullName: m.FullName.ValueString(),
		Email:    helpers.StringFromOptional(m.Email),
		Password: password,
		Banned:   helpers.BoolFromOptional(m.Banned),
	}
}

// toUpdateAPIModel builds the update payload. The password is sent only when it
// changed, so an unrelated edit does not reset it; banned is left to SetUserBanned.
func (m *userResourceModel) toUpdateAPIModel(prior userResourceModel) youtrack.User {
	apiModel := m.toAPIModel()
	apiModel.Banned = false
	if m.Password.Equal(prior.Password) {
		apiModel.Password = ""
	}

	return apiModel
}

// fromAPIModel copies what YouTrack returns. The password is never returned, so it
// keeps whatever the model already holds (the plan, or the prior state on Read).
func (m *userResourceModel) fromAPIModel(user *youtrack.User) {
	m.ID = types.StringValue(user.ID)
	m.Login = types.StringValue(user.Login)
	m.FullName = types.StringValue(user.FullName)
	m.Email = helpers.StringOrNull(user.Email)
	m.Banned = types.BoolValue(user.Banned)
}
