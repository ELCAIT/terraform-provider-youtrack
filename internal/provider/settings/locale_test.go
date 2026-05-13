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
	localeID       = "en_US"
	localeName     = "English"
	localeLanguage = "en"
)

// Helper functions for test data creation
func makeLocaleModel(id, locale, language, name string, community bool) localeSettingsResourceModel {
	return localeSettingsResourceModel{
		ID:        types.StringValue(id),
		Locale:    types.StringValue(locale),
		Language:  types.StringValue(language),
		Community: types.BoolValue(community),
		Name:      types.StringValue(name),
	}
}

func makeLocaleSettings(id, locale, language, name string, community bool) youtrack.LocaleSettings {
	return youtrack.LocaleSettings{
		Locale: youtrack.LocaleDescriptor{
			ID:        id,
			Locale:    locale,
			Language:  language,
			Community: community,
			Name:      name,
		},
	}
}

func TestConvertModelToLocaleSettings(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		locale    string
		language  string
		localName string
		community bool
	}{
		{"converts all fields correctly for English locale", localeID, localeID, localeLanguage, localeName, false},
		{"converts all fields correctly for French community locale", "fr_FR", "fr_FR", "fr", "French", true},
		{"converts all fields correctly for German locale", "de_DE", "de_DE", "de", "German", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := makeLocaleModel(tt.id, tt.locale, tt.language, tt.localName, tt.community)
			want := makeLocaleSettings(tt.id, tt.locale, tt.language, tt.localName, tt.community)
			got := convertModelToLocaleSettings(model)

			helpers.AssertFieldEqual(t, "Locale.ID", got.Locale.ID, want.Locale.ID)
			helpers.AssertFieldEqual(t, "Locale.Locale", got.Locale.Locale, want.Locale.Locale)
			helpers.AssertFieldEqual(t, "Locale.Language", got.Locale.Language, want.Locale.Language)
			helpers.AssertFieldEqual(t, "Locale.Community", got.Locale.Community, want.Locale.Community)
			helpers.AssertFieldEqual(t, "Locale.Name", got.Locale.Name, want.Locale.Name)
		})
	}
}

func TestConvertLocaleSettingsToModel(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		locale    string
		language  string
		localName string
		community bool
	}{
		{"converts all fields correctly for English locale", localeID, localeID, localeLanguage, localeName, false},
		{"converts all fields correctly for French community locale", "fr_FR", "fr_FR", "fr", "French", true},
		{"converts all fields correctly for Spanish locale", "es_ES", "es_ES", "es", "Spanish", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ls := makeLocaleSettings(tt.id, tt.locale, tt.language, tt.localName, tt.community)
			got := convertLocaleSettingsToModel(ls)

			helpers.AssertFieldEqual(t, "ID", got.ID.ValueString(), tt.id)
			helpers.AssertFieldEqual(t, "Locale", got.Locale.ValueString(), tt.locale)
			helpers.AssertFieldEqual(t, "Language", got.Language.ValueString(), tt.language)
			helpers.AssertFieldEqual(t, "Community", got.Community.ValueBool(), tt.community)
			helpers.AssertFieldEqual(t, "Name", got.Name.ValueString(), tt.localName)
		})
	}
}

func TestUpdateLocaleSettingsModelWithTimestamp(t *testing.T) {
	tests := []struct {
		name           string
		localeSettings youtrack.LocaleSettings
		wantNil        bool
	}{
		{
			name: "updates model when response has data",
			localeSettings: youtrack.LocaleSettings{
				Locale: youtrack.LocaleDescriptor{
					ID:        localeID,
					Locale:    localeID,
					Language:  localeLanguage,
					Community: false,
					Name:      localeName,
				},
			},
			wantNil: false,
		},
		{
			name: "updates model with community locale",
			localeSettings: youtrack.LocaleSettings{
				Locale: youtrack.LocaleDescriptor{
					ID:        "ja_JP",
					Locale:    "ja_JP",
					Language:  "ja",
					Community: true,
					Name:      "Japanese",
				},
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resourceModel localeSettingsResourceModel
			updateLocaleSettingsModelWithTimestamp(tt.localeSettings, &resourceModel)

			// Verify the model was updated with the correct values
			helpers.AssertFieldEqual(t, "ID", resourceModel.ID.ValueString(), tt.localeSettings.Locale.ID)
			helpers.AssertFieldEqual(t, "Locale", resourceModel.Locale.ValueString(), tt.localeSettings.Locale.Locale)
			helpers.AssertFieldEqual(t, "Language", resourceModel.Language.ValueString(), tt.localeSettings.Locale.Language)
			helpers.AssertFieldEqual(t, "Community", resourceModel.Community.ValueBool(), tt.localeSettings.Locale.Community)
			helpers.AssertFieldEqual(t, "Name", resourceModel.Name.ValueString(), tt.localeSettings.Locale.Name)

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

func TestConvertLocaleSettingsRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		original youtrack.LocaleSettings
	}{
		{
			name: "round trip for English locale",
			original: youtrack.LocaleSettings{
				Locale: youtrack.LocaleDescriptor{
					ID:        localeID,
					Locale:    localeID,
					Language:  localeLanguage,
					Community: false,
					Name:      localeName,
				},
			},
		},
		{
			name: "round trip for Italian community locale",
			original: youtrack.LocaleSettings{
				Locale: youtrack.LocaleDescriptor{
					ID:        "it_IT",
					Locale:    "it_IT",
					Language:  "it",
					Community: true,
					Name:      "Italian",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to model and back
			model := convertLocaleSettingsToModel(tt.original)
			result := convertModelToLocaleSettings(model)

			// Verify all fields match original
			helpers.AssertFieldEqual(t, "Locale.ID", result.Locale.ID, tt.original.Locale.ID)
			helpers.AssertFieldEqual(t, "Locale.Locale", result.Locale.Locale, tt.original.Locale.Locale)
			helpers.AssertFieldEqual(t, "Locale.Language", result.Locale.Language, tt.original.Locale.Language)
			helpers.AssertFieldEqual(t, "Locale.Community", result.Locale.Community, tt.original.Locale.Community)
			helpers.AssertFieldEqual(t, "Locale.Name", result.Locale.Name, tt.original.Locale.Name)
		})
	}
}
