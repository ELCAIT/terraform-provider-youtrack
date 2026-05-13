package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

const (
	bundleNameDescriptionFmt             = "The %s name."
	bundleIDDescriptionFmt               = "The ID of the %s bundle."
	bundleIsUpdateableDescription        = "Whether current user can update this bundle."
	bundleValuesDescriptionFmt           = "Ordered list of %s values."
	bundleValueIDDescriptionFmt          = "%s value ID."
	bundleValueNameDescriptionFmt        = "%s value name."
	bundleValueLocalizedNameDescFmt      = "Localized %s value name."
	bundleValueDescriptionDescriptionFmt = "%s value description."
	bundleValueArchivedDescriptionFmt    = "Whether this %s value is archived."
	bundleValueOrdinalDescription        = "Position in the bundle."
)

func bundleCommonAttributes(bundleKind string, valueAttributes map[string]schema.Attribute) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: fmt.Sprintf(bundleIDDescriptionFmt, bundleKind),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required:    true,
			Description: fmt.Sprintf(bundleNameDescriptionFmt, bundleKind),
		},
		"is_updateable": schema.BoolAttribute{
			Computed:    true,
			Description: bundleIsUpdateableDescription,
		},
		"values": schema.ListNestedAttribute{
			Required:    true,
			Description: fmt.Sprintf(bundleValuesDescriptionFmt, bundleKind),
			NestedObject: schema.NestedAttributeObject{
				Attributes: valueAttributes,
			},
		},
	}
}

func bundleCommonValueAttributes(valueKind string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: fmt.Sprintf(bundleValueIDDescriptionFmt, valueKind),
		},
		"name": schema.StringAttribute{
			Required:    true,
			Description: fmt.Sprintf(bundleValueNameDescriptionFmt, valueKind),
		},
		"localized_name": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: fmt.Sprintf(bundleValueLocalizedNameDescFmt, valueKind),
		},
		"description": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: fmt.Sprintf(bundleValueDescriptionDescriptionFmt, valueKind),
		},
		"archived": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: fmt.Sprintf(bundleValueArchivedDescriptionFmt, valueKind),
		},
		"ordinal": schema.Int64Attribute{
			Computed:    true,
			Description: bundleValueOrdinalDescription,
		},
	}
}
