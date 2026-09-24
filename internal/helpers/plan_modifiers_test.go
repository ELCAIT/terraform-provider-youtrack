package helpers

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestPreserveStateWhenUnconfiguredBool(t *testing.T) {
	t.Parallel()

	existingResource := tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})}
	newResource := tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{}, nil)}

	tests := []struct {
		name   string
		state  tfsdk.State
		config types.Bool
		prior  types.Bool
		want   types.Bool
	}{
		{name: "create keeps the default", state: newResource, config: types.BoolNull(), prior: types.BoolNull(), want: types.BoolValue(false)},
		{name: "unset keeps the prior value", state: existingResource, config: types.BoolNull(), prior: types.BoolValue(true), want: types.BoolValue(true)},
		{name: "configured value wins", state: existingResource, config: types.BoolValue(false), prior: types.BoolValue(true), want: types.BoolValue(false)},
		{name: "new nested element keeps the default", state: existingResource, config: types.BoolNull(), prior: types.BoolNull(), want: types.BoolValue(false)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.BoolRequest{State: tc.state, ConfigValue: tc.config, StateValue: tc.prior, PlanValue: types.BoolValue(false)}
			resp := &planmodifier.BoolResponse{PlanValue: req.PlanValue}
			PreserveStateWhenUnconfiguredBool.PlanModifyBool(context.Background(), req, resp)

			AssertFieldEqual(t, "PlanValue", resp.PlanValue, tc.want)
		})
	}
}
