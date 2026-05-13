package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

const (
	nestedGroupHolderTypeUser  = "User"
	nestedGroupHolderTypeGroup = "UserGroup"
)

func (r *nestedGroupResource) resolveMembership(
	ctx context.Context,
	plan *nestedGroupResourceModel,
	groupID string,
	diagnostics *diag.Diagnostics,
) (*youtrack.NestedGroup, bool) {
	ownUserLogins, ok := helpers.SetToStringSlice(ctx, plan.OwnUserLogins, diagnostics)
	if !ok {
		return nil, false
	}

	users, ok := r.resolveUsersByLogin(ctx, ownUserLogins, diagnostics)
	if !ok {
		return nil, false
	}

	subGroupNames, ok := helpers.SetToStringSlice(ctx, plan.SubGroupNames, diagnostics)
	if !ok {
		return nil, false
	}

	subGroups, ok := r.resolveSubGroupsByName(ctx, subGroupNames, groupID, diagnostics)
	if !ok {
		return nil, false
	}

	viewers, ok := helpers.SetToStringSlice(ctx, plan.Viewers, diagnostics)
	if !ok {
		return nil, false
	}

	resolvedViewers, ok := r.resolveHoldersByLoginOrGroupName(ctx, viewers, diagnostics)
	if !ok {
		return nil, false
	}

	updaters, ok := helpers.SetToStringSlice(ctx, plan.Updaters, diagnostics)
	if !ok {
		return nil, false
	}

	resolvedUpdaters, ok := r.resolveHoldersByLoginOrGroupName(ctx, updaters, diagnostics)
	if !ok {
		return nil, false
	}

	requireTwoFactorAuth := helpers.BoolFromOptional(plan.RequireTwoFactorAuthentication)
	autoJoin := helpers.BoolFromOptional(plan.AutoJoin)
	autoJoinDomain := helpers.StringFromOptional(plan.AutoJoinDomain)
	description := helpers.StringFromOptional(plan.Description)

	return &youtrack.NestedGroup{
		Name:                 plan.Name.ValueString(),
		Description:          description,
		OwnUsers:             users,
		SubGroups:            subGroups,
		RequireTwoFactorAuth: requireTwoFactorAuth,
		Viewers:              resolvedViewers,
		Updaters:             resolvedUpdaters,
		AutoJoin:             autoJoin,
		AutoJoinDomain:       autoJoinDomain,
		Users:                make([]youtrack.User, 0),
	}, true
}

func (r *nestedGroupResource) resolveUsersByLogin(ctx context.Context, logins []string, diagnostics *diag.Diagnostics) ([]youtrack.User, bool) {
	users := make([]youtrack.User, 0, len(logins))
	for _, login := range logins {
		holder, err := r.client.GetUserByLogin(ctx, login)
		if err != nil {
			diagnostics.AddError(errResolvingUser, fmt.Sprintf("Could not resolve user login '%s': %v", login, err))
			return nil, false
		}

		users = append(users, youtrack.User{ID: holder.Id})
	}

	return users, true
}

func (r *nestedGroupResource) resolveSubGroupsByName(
	ctx context.Context,
	names []string,
	groupID string,
	diagnostics *diag.Diagnostics,
) ([]youtrack.NestedGroup, bool) {
	subGroups := make([]youtrack.NestedGroup, 0, len(names))
	for _, name := range names {
		holder, err := r.resolveSubGroupByNameWithRetry(ctx, name)
		if err != nil {
			diagnostics.AddError(errResolvingGroup, fmt.Sprintf("Could not resolve subgroup '%s': %v", name, err))
			return nil, false
		}

		if holder.Id == groupID {
			diagnostics.AddError(errSelfSubgroup, "A group cannot contain itself as a subgroup")
			return nil, false
		}

		subGroups = append(subGroups, youtrack.NestedGroup{ID: holder.Id})
	}

	return subGroups, true
}

func (r *nestedGroupResource) resolveHoldersByLoginOrGroupName(ctx context.Context, values []string, diagnostics *diag.Diagnostics) ([]youtrack.Holder, bool) {
	holders := make([]youtrack.Holder, 0, len(values))
	for _, value := range values {
		holder, err := r.client.GetUserByLogin(ctx, value)
		if err == nil {
			holders = append(holders, youtrack.Holder{Id: holder.Id, Type: helpers.HolderTypeOrDefault(holder.Type, nestedGroupHolderTypeUser)})
			continue
		}

		holder, err = r.client.GetUserGroupByName(ctx, value)
		if err == nil {
			holders = append(holders, youtrack.Holder{Id: holder.Id, Type: helpers.HolderTypeOrDefault(holder.Type, nestedGroupHolderTypeGroup)})
			continue
		}

		diagnostics.AddError(
			errResolvingGroup,
			fmt.Sprintf("Could not resolve user login or group name '%s'", value),
		)
		return nil, false
	}

	return holders, true
}

func (r *nestedGroupResource) resolveSubGroupByNameWithRetry(ctx context.Context, name string) (*youtrack.Holder, error) {
	var lastErr error

	for attempt := 0; attempt < subGroupResolveMaxAttempts; attempt++ {
		holder, err := r.client.GetUserGroupByName(ctx, name)
		if err == nil {
			return holder, nil
		}

		lastErr = err
		if !shouldRetrySubGroupResolution(err) || attempt == subGroupResolveMaxAttempts-1 {
			return nil, err
		}

		if !waitForSubGroupRetry(ctx, subGroupResolveRetryDelay) {
			return nil, ctx.Err()
		}
	}

	return nil, lastErr
}

func shouldRetrySubGroupResolution(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(strings.ToLower(err.Error()), errNotFoundFragment)
}

func waitForSubGroupRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
