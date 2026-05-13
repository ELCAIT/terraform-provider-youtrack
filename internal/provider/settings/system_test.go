// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package settings

import (
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	adminEmail     = "admin@example.com"
	baseURL        = "https://youtrack.example.com"
	maxExportItems = 1000
	maxUploadSize  = 10485760 // 10MB in bytes

	// Test values for various configurations
	smallMaxExportItems = 500
	smallMaxUploadSize  = 5242880 // 5MB in bytes
)

// Helper functions for test data creation
func makeSystemModel(email, url string, maxExport, maxUpload int64, allowStats, readOnly bool) systemSettingsResourceModel {
	return systemSettingsResourceModel{
		AdministratorEmail:        types.StringValue(email),
		MaxExportItems:            types.Int64Value(maxExport),
		MaxUploadFileSize:         types.Int64Value(maxUpload),
		AllowStatisticsCollection: types.BoolValue(allowStats),
		IsApplicationReadOnly:     types.BoolValue(readOnly),
		BaseURL:                   types.StringValue(url),
	}
}

func makeSystemSettings(email, url string, maxExport, maxUpload int, allowStats, readOnly bool) youtrack.SystemSettings {
	return youtrack.SystemSettings{
		AdministratorEmail:        email,
		MaxExportItems:            maxExport,
		MaxUploadFileSize:         maxUpload,
		AllowStatisticsCollection: allowStats,
		IsApplicationReadOnly:     readOnly,
		BaseUrl:                   url,
	}
}

func TestConvertModelToSystemSettings(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		url        string
		maxExport  int64
		maxUpload  int64
		allowStats bool
		readOnly   bool
	}{
		{"converts all fields correctly", adminEmail, baseURL, maxExportItems, maxUploadSize, true, false},
		{"handles read-only mode enabled", "", baseURL, smallMaxExportItems, smallMaxUploadSize, false, true},
		{"handles default values", "", "", smallMaxExportItems, maxUploadSize, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := makeSystemModel(tt.email, tt.url, tt.maxExport, tt.maxUpload, tt.allowStats, tt.readOnly)
			want := makeSystemSettings(tt.email, tt.url, int(tt.maxExport), int(tt.maxUpload), tt.allowStats, tt.readOnly)
			got := convertModelToSystemSettings(model)

			helpers.AssertFieldEqual(t, "AdministratorEmail", got.AdministratorEmail, want.AdministratorEmail)
			helpers.AssertFieldEqual(t, "MaxExportItems", got.MaxExportItems, want.MaxExportItems)
			helpers.AssertFieldEqual(t, "MaxUploadFileSize", got.MaxUploadFileSize, want.MaxUploadFileSize)
			helpers.AssertFieldEqual(t, "AllowStatisticsCollection", got.AllowStatisticsCollection, want.AllowStatisticsCollection)
			helpers.AssertFieldEqual(t, "IsApplicationReadOnly", got.IsApplicationReadOnly, want.IsApplicationReadOnly)
			helpers.AssertFieldEqual(t, "BaseUrl", got.BaseUrl, want.BaseUrl)
		})
	}
}

func TestConvertSystemSettingsToModel(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		url        string
		maxExport  int
		maxUpload  int
		allowStats bool
		readOnly   bool
	}{
		{"converts all fields correctly", adminEmail, baseURL, maxExportItems, maxUploadSize, true, false},
		{"handles read-only mode enabled", "", baseURL, smallMaxExportItems, smallMaxUploadSize, false, true},
		{"handles empty settings", "", "", 0, 0, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := makeSystemSettings(tt.email, tt.url, tt.maxExport, tt.maxUpload, tt.allowStats, tt.readOnly)
			got := convertSystemSettingsToModel(ss)

			helpers.AssertFieldEqual(t, "AdministratorEmail", got.AdministratorEmail.ValueString(), tt.email)
			helpers.AssertFieldEqual(t, "MaxExportItems", got.MaxExportItems.ValueInt64(), int64(tt.maxExport))
			helpers.AssertFieldEqual(t, "MaxUploadFileSize", got.MaxUploadFileSize.ValueInt64(), int64(tt.maxUpload))
			helpers.AssertFieldEqual(t, "AllowStatisticsCollection", got.AllowStatisticsCollection.ValueBool(), tt.allowStats)
			helpers.AssertFieldEqual(t, "IsApplicationReadOnly", got.IsApplicationReadOnly.ValueBool(), tt.readOnly)
			helpers.AssertFieldEqual(t, "BaseURL", got.BaseURL.ValueString(), tt.url)
		})
	}
}

func TestUpdateSystemSettingsModelWithTimestamp(t *testing.T) {
	tests := []struct {
		name           string
		systemSettings youtrack.SystemSettings
	}{
		{
			name: "updates model and sets timestamp",
			systemSettings: youtrack.SystemSettings{
				AdministratorEmail:        adminEmail,
				MaxExportItems:            maxExportItems,
				MaxUploadFileSize:         maxUploadSize,
				AllowStatisticsCollection: true,
				IsApplicationReadOnly:     false,
				BaseUrl:                   baseURL,
			},
		},
		{
			name: "updates model with minimal settings",
			systemSettings: youtrack.SystemSettings{
				AdministratorEmail:        "",
				MaxExportItems:            500,
				MaxUploadFileSize:         10485760,
				AllowStatisticsCollection: false,
				IsApplicationReadOnly:     false,
				BaseUrl:                   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceModel := systemSettingsResourceModel{}
			updateSystemSettingsModelWithTimestamp(tt.systemSettings, &resourceModel)

			if resourceModel.LastUpdated.IsNull() {
				t.Error("LastUpdated should be set")
			}

			helpers.AssertFieldEqual(t, "AdministratorEmail", resourceModel.AdministratorEmail.ValueString(), tt.systemSettings.AdministratorEmail)
			helpers.AssertFieldEqual(t, "MaxExportItems", int(resourceModel.MaxExportItems.ValueInt64()), tt.systemSettings.MaxExportItems)
			helpers.AssertFieldEqual(t, "MaxUploadFileSize", int(resourceModel.MaxUploadFileSize.ValueInt64()), tt.systemSettings.MaxUploadFileSize)
			helpers.AssertFieldEqual(t, "AllowStatisticsCollection", resourceModel.AllowStatisticsCollection.ValueBool(), tt.systemSettings.AllowStatisticsCollection)
			helpers.AssertFieldEqual(t, "IsApplicationReadOnly", resourceModel.IsApplicationReadOnly.ValueBool(), tt.systemSettings.IsApplicationReadOnly)
			helpers.AssertFieldEqual(t, "BaseURL", resourceModel.BaseURL.ValueString(), tt.systemSettings.BaseUrl)
		})
	}
}

func TestRoundTripConversion(t *testing.T) {
	tests := []struct {
		name     string
		original youtrack.SystemSettings
	}{
		{
			name: "full configuration round trip",
			original: youtrack.SystemSettings{
				AdministratorEmail:        adminEmail,
				MaxExportItems:            maxExportItems,
				MaxUploadFileSize:         maxUploadSize,
				AllowStatisticsCollection: true,
				IsApplicationReadOnly:     false,
				BaseUrl:                   baseURL,
			},
		},
		{
			name: "minimal configuration round trip",
			original: youtrack.SystemSettings{
				AdministratorEmail:        "",
				MaxExportItems:            500,
				MaxUploadFileSize:         10485760,
				AllowStatisticsCollection: false,
				IsApplicationReadOnly:     false,
				BaseUrl:                   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to model
			model := convertSystemSettingsToModel(tt.original)

			// Convert back to system settings
			result := convertModelToSystemSettings(model)

			// Verify the round trip preserves all data
			helpers.AssertFieldEqual(t, "AdministratorEmail", result.AdministratorEmail, tt.original.AdministratorEmail)
			helpers.AssertFieldEqual(t, "MaxExportItems", result.MaxExportItems, tt.original.MaxExportItems)
			helpers.AssertFieldEqual(t, "MaxUploadFileSize", result.MaxUploadFileSize, tt.original.MaxUploadFileSize)
			helpers.AssertFieldEqual(t, "AllowStatisticsCollection", result.AllowStatisticsCollection, tt.original.AllowStatisticsCollection)
			helpers.AssertFieldEqual(t, "IsApplicationReadOnly", result.IsApplicationReadOnly, tt.original.IsApplicationReadOnly)
			helpers.AssertFieldEqual(t, "BaseUrl", result.BaseUrl, tt.original.BaseUrl)
		})
	}
}
