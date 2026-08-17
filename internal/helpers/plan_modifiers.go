package helpers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// PreserveStateWhenUnconfiguredDescription is the description shared by every
// preserve-state-when-unconfigured plan modifier below.
const PreserveStateWhenUnconfiguredDescription = "Once set, preserves the prior state value when this attribute is left unset in " +
	"configuration, instead of resetting it to the default on every apply."

// PreserveStateWhenUnconfiguredBool is a planmodifier.Bool for an Optional+Computed+Default bool
// attribute. It keeps the prior state value when the attribute is left unset in configuration,
// instead of resending the schema Default on every apply. Without this, an unrelated apply
// silently reverts a value that was changed out-of-band (e.g. directly in Hub or the YouTrack UI),
// because the framework resolves Default from the raw config value alone, before any plan modifier
// runs, regardless of what is already in state.
var PreserveStateWhenUnconfiguredBool planmodifier.Bool = preserveStateWhenUnconfiguredBoolModifier{}

type preserveStateWhenUnconfiguredBoolModifier struct{}

func (m preserveStateWhenUnconfiguredBoolModifier) Description(_ context.Context) string {
	return PreserveStateWhenUnconfiguredDescription
}

func (m preserveStateWhenUnconfiguredBoolModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m preserveStateWhenUnconfiguredBoolModifier) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.State.Raw.IsNull() {
		// Resource creation: let the schema Default apply.
		return
	}

	if req.ConfigValue.IsNull() {
		resp.PlanValue = req.StateValue
	}
}

// PreserveStateWhenUnconfiguredString is the string counterpart of
// PreserveStateWhenUnconfiguredBool, for Optional+Computed+Default string attributes.
var PreserveStateWhenUnconfiguredString planmodifier.String = preserveStateWhenUnconfiguredStringModifier{}

type preserveStateWhenUnconfiguredStringModifier struct{}

func (m preserveStateWhenUnconfiguredStringModifier) Description(_ context.Context) string {
	return PreserveStateWhenUnconfiguredDescription
}

func (m preserveStateWhenUnconfiguredStringModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m preserveStateWhenUnconfiguredStringModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() {
		// Resource creation: let the schema Default apply.
		return
	}

	if req.ConfigValue.IsNull() {
		resp.PlanValue = req.StateValue
	}
}

// PreserveStateWhenUnconfiguredInt64 is the int64 counterpart of
// PreserveStateWhenUnconfiguredBool, for Optional+Computed+Default int64 attributes.
var PreserveStateWhenUnconfiguredInt64 planmodifier.Int64 = preserveStateWhenUnconfiguredInt64Modifier{}

type preserveStateWhenUnconfiguredInt64Modifier struct{}

func (m preserveStateWhenUnconfiguredInt64Modifier) Description(_ context.Context) string {
	return PreserveStateWhenUnconfiguredDescription
}

func (m preserveStateWhenUnconfiguredInt64Modifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m preserveStateWhenUnconfiguredInt64Modifier) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.State.Raw.IsNull() {
		// Resource creation: let the schema Default apply.
		return
	}

	if req.ConfigValue.IsNull() {
		resp.PlanValue = req.StateValue
	}
}
