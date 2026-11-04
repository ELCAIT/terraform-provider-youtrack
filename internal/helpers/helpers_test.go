package helpers

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateNotBlank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		value      types.String
		wantErrors int
	}{
		{name: "set", value: types.StringValue("x")},
		{name: "null", value: types.StringNull()},
		{name: "unknown", value: types.StringUnknown()},
		{name: "empty", value: types.StringValue(""), wantErrors: 1},
		{name: "blank", value: types.StringValue("  "), wantErrors: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var diags diag.Diagnostics
			ValidateNotBlank(map[string]types.String{"app_name": tc.value}, "Invalid", &diags)
			AssertFieldEqual(t, "errors", diags.ErrorsCount(), tc.wantErrors)
		})
	}
}
