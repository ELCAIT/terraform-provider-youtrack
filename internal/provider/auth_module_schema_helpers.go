// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

const (
	// Identity provider names used in the shared attribute descriptions.
	azureProviderName  = "Entra ID"
	oauth2ProviderName = "the identity provider"

	// attrAllowedCreateNewUsers is the attribute Hub forces to true when an auth
	// module is created, whatever the request asked for.
	attrAllowedCreateNewUsers = "allowed_create_new_users"
)

// authModuleCommonAttributes returns the schema attributes every Hub auth module
// resource shares. providerName names the identity provider as it reads mid
// sentence ("Entra ID", "the identity provider") in the descriptions that mention
// it; the remaining descriptions are provider-agnostic.
//
// The returned map is freshly built on every call, so callers own it and may add
// their own module-specific attributes to it.
func authModuleCommonAttributes(providerName string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "Hub ID of the auth module (computed).",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required:    true,
			Description: "Display name of the auth module.",
		},
		"disabled": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether the auth module is disabled. Defaults to false.",
		},
		"redirect_uri": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "OAuth 2.0 redirect URI. When omitted, Hub sets this automatically.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"allowed_create_new_users": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether new users may be created on first login via this module. Defaults to false.",
		},
		"background_sync_enabled": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: fmt.Sprintf("Whether background synchronisation with %s is enabled. Defaults to false.", providerName),
		},
		"icon_url": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "URL of a custom icon to display for this auth module.",
		},
		"extension_grant_type": schema.StringAttribute{
			Optional:    true,
			Description: "Custom OAuth 2.0 extension grant type.",
		},
		"connection_timeout": schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: fmt.Sprintf("Connection timeout in milliseconds when contacting %s.", providerName),
		},
		"read_timeout": schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: fmt.Sprintf("Read timeout in milliseconds when contacting %s.", providerName),
		},
		"sync_interval": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Cron expression that controls background synchronisation frequency.",
		},
		"is_default": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether this auth module is the default login method. Defaults to false.",
		},
	}
}
