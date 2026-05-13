// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package settings

import (
	"context"
	"errors"
	"fmt"
	"testing"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Test case structure for REST settings tests
type restSettingsTestCase struct {
	name            string
	id              string
	allowAllOrigins bool
	allowedOrigins  []string
}

// Common test cases used across multiple tests
var commonRestSettingsTestCases = []restSettingsTestCase{
	{
		name:            "converts all fields correctly with allow all origins disabled",
		id:              "rest-1",
		allowAllOrigins: false,
		allowedOrigins:  []string{"https://example.com", "https://api.example.com"},
	},
	{
		name:            "converts all fields correctly with allow all origins enabled",
		id:              "rest-2",
		allowAllOrigins: true,
		allowedOrigins:  []string{},
	},
	{
		name:            "converts all fields correctly with empty origins list",
		id:              "rest-3",
		allowAllOrigins: false,
		allowedOrigins:  []string{},
	},
}

// Helper functions for test data creation
func makeRestModel(id string, allowAllOrigins bool, allowedOrigins []string) (restSettingsResourceModel, error) {
	ctx := context.Background()
	allowedOriginsList, diags := types.ListValueFrom(ctx, types.StringType, allowedOrigins)
	if diags.HasError() {
		return restSettingsResourceModel{}, errors.New(diags.Errors()[0].Detail())
	}

	return restSettingsResourceModel{
		ID:              types.StringValue(id),
		AllowAllOrigins: types.BoolValue(allowAllOrigins),
		AllowedOrigins:  allowedOriginsList,
	}, nil
}

func makeRestSettings(id string, allowAllOrigins bool, allowedOrigins []string) youtrack.RestSettings {
	return youtrack.RestSettings{
		ID:              id,
		AllowAllOrigins: allowAllOrigins,
		AllowedOrigins:  allowedOrigins,
	}
}

// assertAllowedOrigins verifies that two allowed origins slices are equal
func assertAllowedOrigins(t *testing.T, got, want []string) {
	t.Helper()
	helpers.AssertFieldEqual(t, "AllowedOrigins length", len(got), len(want))
	for i, origin := range got {
		helpers.AssertFieldEqual(t, fmt.Sprintf("AllowedOrigins[%d]", i), origin, want[i])
	}
}

func TestConvertModelToRestSettings(t *testing.T) {
	// Add specific test case for this function
	tests := append(commonRestSettingsTestCases, restSettingsTestCase{
		name:            "converts all fields correctly with single origin",
		id:              "rest-4",
		allowAllOrigins: false,
		allowedOrigins:  []string{"https://trusted.example.com"},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := makeRestModel(tt.id, tt.allowAllOrigins, tt.allowedOrigins)
			if err != nil {
				t.Fatalf("failed to create REST model: %v", err)
			}
			want := makeRestSettings(tt.id, tt.allowAllOrigins, tt.allowedOrigins)
			got, ok := convertModelToRestSettings(model)

			if !ok {
				t.Fatal("convertModelToRestSettings failed")
			}

			helpers.AssertFieldEqual(t, "ID", got.ID, want.ID)
			helpers.AssertFieldEqual(t, "AllowAllOrigins", got.AllowAllOrigins, want.AllowAllOrigins)
			assertAllowedOrigins(t, got.AllowedOrigins, want.AllowedOrigins)
		})
	}
}

func TestConvertRestSettingsToModel(t *testing.T) {
	// Add specific test case for this function
	tests := append(commonRestSettingsTestCases, restSettingsTestCase{
		name:            "converts all fields correctly with single origin",
		id:              "rest-4",
		allowAllOrigins: false,
		allowedOrigins:  []string{"https://single.example.com"},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			rs := makeRestSettings(tt.id, tt.allowAllOrigins, tt.allowedOrigins)
			got := convertRestSettingsToModel(ctx, rs)

			if got == nil {
				t.Fatal("convertRestSettingsToModel returned nil")
			}

			helpers.AssertFieldEqual(t, "ID", got.ID.ValueString(), tt.id)
			helpers.AssertFieldEqual(t, "AllowAllOrigins", got.AllowAllOrigins.ValueBool(), tt.allowAllOrigins)

			var gotOrigins []string
			if diags := got.AllowedOrigins.ElementsAs(ctx, &gotOrigins, false); diags.HasError() {
				t.Fatalf("failed to convert allowed origins list: %v", diags)
			}
			assertAllowedOrigins(t, gotOrigins, tt.allowedOrigins)
		})
	}
}

func TestUpdateRestSettingsModelWithTimestamp(t *testing.T) {
	tests := []struct {
		name         string
		restSettings youtrack.RestSettings
	}{
		{
			name: "updates model when response has data with multiple origins",
			restSettings: youtrack.RestSettings{
				ID:              "rest-1",
				AllowAllOrigins: false,
				AllowedOrigins:  []string{"https://example.com", "https://api.example.com"},
			},
		},
		{
			name: "updates model with allow all origins enabled",
			restSettings: youtrack.RestSettings{
				ID:              "rest-2",
				AllowAllOrigins: true,
				AllowedOrigins:  []string{},
			},
		},
		{
			name: "updates model with empty origins list",
			restSettings: youtrack.RestSettings{
				ID:              "rest-3",
				AllowAllOrigins: false,
				AllowedOrigins:  []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var resourceModel restSettingsResourceModel
			ok := updateRestSettingsModelWithTimestamp(ctx, tt.restSettings, &resourceModel)

			if !ok {
				t.Fatal("updateRestSettingsModelWithTimestamp failed")
			}

			// Verify the model was updated with the correct values
			helpers.AssertFieldEqual(t, "ID", resourceModel.ID.ValueString(), tt.restSettings.ID)
			helpers.AssertFieldEqual(t, "AllowAllOrigins", resourceModel.AllowAllOrigins.ValueBool(), tt.restSettings.AllowAllOrigins)

			var gotOrigins []string
			if diags := resourceModel.AllowedOrigins.ElementsAs(ctx, &gotOrigins, false); diags.HasError() {
				t.Fatalf("failed to convert allowed origins list: %v", diags)
			}
			assertAllowedOrigins(t, gotOrigins, tt.restSettings.AllowedOrigins)

			// Verify timestamp was set
			if resourceModel.LastUpdated.IsNull() {
				t.Error("LastUpdated should not be null after update")
			}
			if resourceModel.LastUpdated.ValueString() == "" {
				t.Error("LastUpdated should not be empty after update")
			}
		})
	}
}
