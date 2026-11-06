package helpers

import (
	"strings"
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

func TestSplitImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id         string
		wantOK     bool
		wantParent string
		wantChild  string
	}{
		{id: "0-1/my-workflow", wantOK: true, wantParent: "0-1", wantChild: "my-workflow"},
		{id: "0-1/@jetbrains/youtrack-workflow-due-date", wantOK: true, wantParent: "0-1", wantChild: "@jetbrains/youtrack-workflow-due-date"},
		{id: " 0-1 / my-workflow ", wantOK: true, wantParent: "0-1", wantChild: "my-workflow"},
		{id: "0-1"},
		{id: "0-1/"},
		{id: "/my-workflow"},
		{id: " / "},
		{id: ""},
	}

	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()

			parent, child, ok := SplitImportID(tc.id)
			AssertFieldEqual(t, "ok", ok, tc.wantOK)
			AssertFieldEqual(t, "parent", parent, tc.wantParent)
			AssertFieldEqual(t, "child", child, tc.wantChild)
		})
	}
}

func TestValidateNotBlankOrder(t *testing.T) {
	t.Parallel()

	var diags diag.Diagnostics
	blank := types.StringValue("")
	ValidateNotBlank(map[string]types.String{"project_id": blank, "app_name": blank, "b": blank}, "Invalid", &diags)

	var got []string
	for _, d := range diags {
		if withPath, ok := d.(diag.DiagnosticWithPath); ok {
			got = append(got, withPath.Path().String())
		}
	}
	AssertFieldEqual(t, "order", strings.Join(got, ","), "app_name,b,project_id")
}
