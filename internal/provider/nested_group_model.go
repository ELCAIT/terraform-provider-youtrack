package provider

import (
	"context"
	"sort"
	"strings"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func newNestedGroupPayload(name string) youtrack.NestedGroup {
	return youtrack.NestedGroup{
		Name:                 name,
		OwnUsers:             make([]youtrack.User, 0),
		SubGroups:            make([]youtrack.NestedGroup, 0),
		RequireTwoFactorAuth: false,
		Viewers:              make([]youtrack.Holder, 0),
		Updaters:             make([]youtrack.Holder, 0),
		AutoJoin:             false,
		AutoJoinDomain:       "",
		Users:                make([]youtrack.User, 0),
	}
}

func (r *nestedGroupResource) mapAPIToModel(
	ctx context.Context,
	apiGroup *youtrack.NestedGroup,
	model *nestedGroupResourceModel,
	diagnostics *diag.Diagnostics,
) {
	model.ID = types.StringValue(apiGroup.ID)
	model.Name = types.StringValue(apiGroup.Name)
	model.Description = types.StringValue(apiGroup.Description)
	model.RequireTwoFactorAuthentication = types.BoolValue(apiGroup.RequireTwoFactorAuth)
	model.AutoJoin = types.BoolValue(apiGroup.AutoJoin)
	model.AutoJoinDomain = types.StringValue(apiGroup.AutoJoinDomain)

	ownUserLogins := extractUserLogins(apiGroup.OwnUsers)
	subGroupNames := extractSubGroupNames(apiGroup.SubGroups)

	ownUsersSet, diags := types.SetValueFrom(ctx, types.StringType, ownUserLogins)
	diagnostics.Append(diags...)
	subGroupsSet, diags := types.SetValueFrom(ctx, types.StringType, subGroupNames)
	diagnostics.Append(diags...)

	viewersSet, diags := types.SetValueFrom(ctx, types.StringType, holdersToNames(apiGroup.Viewers))
	diagnostics.Append(diags...)
	updatersSet, diags := types.SetValueFrom(ctx, types.StringType, holdersToNames(apiGroup.Updaters))
	diagnostics.Append(diags...)

	if diagnostics.HasError() {
		return
	}

	model.OwnUserLogins = ownUsersSet
	model.SubGroupNames = subGroupsSet
	model.Viewers = viewersSet
	model.Updaters = updatersSet
}

func extractUserLogins(users []youtrack.User) []string {
	logins := make([]string, 0, len(users))
	for _, user := range users {
		if user.Login != "" {
			logins = append(logins, user.Login)
		}
	}
	sort.Strings(logins)
	return logins
}

func extractSubGroupNames(groups []youtrack.NestedGroup) []string {
	names := make([]string, 0, len(groups))
	for _, group := range groups {
		if group.Name != "" {
			names = append(names, group.Name)
		}
	}
	sort.Strings(names)
	return names
}

func holdersToNames(holders []youtrack.Holder) []string {
	values := make([]string, 0, len(holders))
	seen := make(map[string]struct{}, len(holders))
	for _, holder := range holders {
		value := strings.TrimSpace(holder.Login)
		if value == "" {
			value = strings.TrimSpace(holder.Name)
		}
		if value == "" {
			value = strings.TrimSpace(holder.Id)
		}
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}
