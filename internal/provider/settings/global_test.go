// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package settings

import (
	"testing"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testLicenseKey = "test-license-key-123456789"
	testID         = "global-settings-id"
)

// Helper functions for test data creation
func makeGlobalModel(id, license string) globalSettingsResourceModel {
	model := globalSettingsResourceModel{
		ID: types.StringValue(id),
	}
	if license == "" {
		model.License = types.StringNull()
	} else {
		model.License = types.StringValue(license)
	}
	return model
}

func makeGlobalSettings(id, license string) youtrack.GlobalSettings {
	gs := youtrack.GlobalSettings{
		ID: id,
	}
	if license != "" {
		gs.License = &youtrack.License{
			License: license,
		}
	}
	return gs
}

func TestConvertModelToGlobalSettings(t *testing.T) {
	tests := []struct {
		name         string
		id           string
		license      string
		expectNilLic bool
	}{
		{"converts all fields correctly", testID, testLicenseKey, false},
		{"handles null license", testID, "", true},
		{"handles different values", "other-id", "another-license", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := makeGlobalModel(tt.id, tt.license)
			got := convertModelToGlobalSettings(model)

			helpers.AssertFieldEqual(t, "ID", got.ID, tt.id)

			if tt.expectNilLic {
				if got.License != nil {
					t.Errorf("License should be nil, got: %+v", got.License)
				}
			} else {
				if got.License == nil {
					t.Error("License should not be nil")
				} else {
					helpers.AssertFieldEqual(t, "License.License", got.License.License, tt.license)
				}
			}
		})
	}
}

func TestConvertGlobalSettingsToModel(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		license       string
		expectNullLic bool
	}{
		{"converts all fields correctly", testID, testLicenseKey, false},
		{"handles null license", testID, "", true},
		{"handles different values", "other-id", "another-license", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := makeGlobalSettings(tt.id, tt.license)
			got := convertGlobalSettingsToModel(gs)

			helpers.AssertFieldEqual(t, "ID", got.ID.ValueString(), tt.id)

			if tt.expectNullLic {
				if !got.License.IsNull() {
					t.Errorf("License should be null, got: %v", got.License.ValueString())
				}
			} else {
				helpers.AssertFieldEqual(t, "License", got.License.ValueString(), tt.license)
			}
		})
	}
}

func TestGetLicenseString(t *testing.T) {
	tests := []struct {
		name     string
		license  string
		expected string
	}{
		{"returns license string when set", testLicenseKey, testLicenseKey},
		{"returns empty string when license is nil", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := makeGlobalSettings(testID, tt.license)
			got := getLicenseString(gs)
			helpers.AssertFieldEqual(t, "License", got, tt.expected)
		})
	}
}

func TestUpdateGlobalSettingsModelWithTimestamp(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		license       string
		expectNullLic bool
	}{
		{"updates model and sets timestamp", testID, testLicenseKey, false},
		{"updates model with null license", testID, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalSettings := makeGlobalSettings(tt.id, tt.license)
			resourceModel := globalSettingsResourceModel{}
			updateGlobalSettingsModelWithTimestamp(globalSettings, &resourceModel)

			if resourceModel.LastUpdated.IsNull() {
				t.Error("LastUpdated should be set")
			}

			helpers.AssertFieldEqual(t, "ID", resourceModel.ID.ValueString(), tt.id)

			if tt.expectNullLic {
				if !resourceModel.License.IsNull() {
					t.Errorf("License should be null, got: %v", resourceModel.License.ValueString())
				}
			} else {
				helpers.AssertFieldEqual(t, "License", resourceModel.License.ValueString(), tt.license)
			}
		})
	}
}

func TestGlobalSettingsRoundTripConversion(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		license string
		isNull  bool
	}{
		{"full configuration round trip", testID, testLicenseKey, false},
		{"null license round trip", testID, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start with a model
			originalModel := makeGlobalModel(tt.id, tt.license)

			// Convert to GlobalSettings
			gs := convertModelToGlobalSettings(originalModel)

			// Convert back to model
			resultModel := convertGlobalSettingsToModel(gs)

			// Verify the round trip preserves all data
			helpers.AssertFieldEqual(t, "ID", resultModel.ID.ValueString(), tt.id)

			if tt.isNull {
				if !resultModel.License.IsNull() {
					t.Errorf("License should be null after round trip, got: %v", resultModel.License.ValueString())
				}
			} else {
				helpers.AssertFieldEqual(t, "License", resultModel.License.ValueString(), tt.license)
			}
		})
	}
}
