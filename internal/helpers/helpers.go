package helpers

import (
	"context"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"testing"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	ErrUnexpectedResourceConfigType = "Unexpected Resource Configure Type"
	ErrUnexpectedConfigureType      = "Expected *youtrack.Client, got: %T. Please report this issue to the provider developers."
	ErrCouldNotUpdateFmt            = "Could not update %s, unexpected error: %v"
	ErrInvalidEmailAddress          = "Invalid Email Address"
	ErrInvalidPortNumber            = "Invalid Port Number"
	ErrInvalidURL                   = "Invalid URL"
)

// AssertFieldEqual is a test helper that compares two values and reports an error if they differ.
func AssertFieldEqual(t *testing.T, fieldName string, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", fieldName, got, want)
	}
}

// GetClientFromConfigure extracts and validates the YouTrack client from a ConfigureRequest.
func GetClientFromConfigure(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*youtrack.Client, bool) {
	if req.ProviderData == nil {
		return nil, false
	}

	client, ok := req.ProviderData.(*youtrack.Client)
	if !ok {
		resp.Diagnostics.AddError(
			ErrUnexpectedResourceConfigType,
			fmt.Sprintf(ErrUnexpectedConfigureType, req.ProviderData),
		)
		return nil, false
	}

	return client, true
}

// GetPlanAndCheckError retrieves the plan and checks for diagnostics errors.
func GetPlanAndCheckError(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse, plan interface{}) bool {
	diags := req.Plan.Get(ctx, plan)
	resp.Diagnostics.Append(diags...)
	return !resp.Diagnostics.HasError()
}

// GetPlanAndCheckErrorUpdate retrieves the plan in update context and checks for diagnostics errors.
func GetPlanAndCheckErrorUpdate(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse, plan interface{}) bool {
	diags := req.Plan.Get(ctx, plan)
	resp.Diagnostics.Append(diags...)
	return !resp.Diagnostics.HasError()
}

// GetStateAndCheckError retrieves the state and checks for diagnostics errors.
func GetStateAndCheckError(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse, state interface{}) bool {
	diags := req.State.Get(ctx, state)
	resp.Diagnostics.Append(diags...)
	return !resp.Diagnostics.HasError()
}

// GetStateAndCheckErrorDelete retrieves the state in delete context and checks for diagnostics errors.
func GetStateAndCheckErrorDelete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse, state interface{}) bool {
	diags := req.State.Get(ctx, state)
	resp.Diagnostics.Append(diags...)
	return !resp.Diagnostics.HasError()
}

// SetStateAndCheckError sets the state and appends any diagnostics errors.
func SetStateAndCheckError(ctx context.Context, resp interface{}, state interface{}) {
	var diags diag.Diagnostics

	switch r := resp.(type) {
	case *resource.CreateResponse:
		diags = r.State.Set(ctx, state)
		r.Diagnostics.Append(diags...)
	case *resource.ReadResponse:
		diags = r.State.Set(ctx, state)
		r.Diagnostics.Append(diags...)
	case *resource.UpdateResponse:
		diags = r.State.Set(ctx, state)
		r.Diagnostics.Append(diags...)
	case *resource.DeleteResponse:
		diags = r.State.Set(ctx, state)
		r.Diagnostics.Append(diags...)
	case *resource.ImportStateResponse:
		diags = r.State.Set(ctx, state)
		r.Diagnostics.Append(diags...)
	}
}

// ValidateEmailField validates that a string field contains a valid email address if not empty.
func ValidateEmailField(value types.String, fieldPath path.Path, fieldDescription string, diagnostics *diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return
	}

	email := value.ValueString()
	if email == "" {
		return
	}

	if _, err := mail.ParseAddress(email); err != nil {
		diagnostics.AddAttributeError(
			fieldPath,
			ErrInvalidEmailAddress,
			fmt.Sprintf("The %s must be a valid email address: %s", fieldDescription, err.Error()),
		)
	}
}

// ListToStringSlice converts a Terraform string list into a Go string slice.
func ListToStringSlice(ctx context.Context, list types.List) ([]string, bool) {
	var out []string
	diags := list.ElementsAs(ctx, &out, false)
	if diags.HasError() {
		return nil, false
	}

	return out, true
}

// SetToStringSlice converts a Terraform string set into a deduplicated, trimmed, sorted Go string slice.
func SetToStringSlice(ctx context.Context, input types.Set, diagnostics *diag.Diagnostics) ([]string, bool) {
	if input.IsNull() || input.IsUnknown() {
		return []string{}, true
	}

	var values []string
	diags := input.ElementsAs(ctx, &values, false)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return nil, false
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	sort.Strings(result)
	return result, true
}

// BoolFromOptional returns the bool value of a types.Bool, or false if null or unknown.
func BoolFromOptional(value types.Bool) bool {
	if value.IsNull() || value.IsUnknown() {
		return false
	}

	return value.ValueBool()
}

// OptionalBoolValue returns the bool value and whether it is known (not null/unknown).
func OptionalBoolValue(value types.Bool) (bool, bool) {
	if value.IsNull() || value.IsUnknown() {
		return false, false
	}

	return BoolFromOptional(value), true
}

// StringFromOptional returns the trimmed string value of a types.String, or empty string if null or unknown.
func StringFromOptional(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}

	return strings.TrimSpace(value.ValueString())
}

// OptionalStringValue returns the raw string value and whether it is known (not null/unknown).
func OptionalStringValue(value types.String) (string, bool) {
	if value.IsNull() || value.IsUnknown() {
		return "", false
	}

	return value.ValueString(), true
}

// StringOrNull returns a null types.String when the value is empty, otherwise a types.StringValue.
func StringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}

	return types.StringValue(s)
}

// HolderTypeOrDefault returns the trimmed value if non-empty, otherwise the fallback.
func HolderTypeOrDefault(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}

	return trimmed
}

// HasResourceID checks whether a Terraform string ID is known and non-empty.
func HasResourceID(id types.String) bool {
	return !id.IsNull() && !id.IsUnknown() && strings.TrimSpace(id.ValueString()) != ""
}

// ValidateResourceID validates that a Terraform string ID is known and non-empty.
func ValidateResourceID(id types.String, diagnostics *diag.Diagnostics, summary, detail string) bool {
	if !HasResourceID(id) {
		diagnostics.AddError(summary, detail)
		return false
	}

	return true
}
