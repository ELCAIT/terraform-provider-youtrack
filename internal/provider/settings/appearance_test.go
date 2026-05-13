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
	dateFormatID         = "youtrack.datefieldformat.iso8601"
	dateFormatPattern    = "yyyy-MM-dd"
	timeZoneID           = "Europe/Zurich"
	timeZonePresentation = "Central European Time"
	timeZoneOffset       = 3600000 // 1 hour in milliseconds
)

// Helper functions for test data creation
func makeModel(dateFormatID, timeZone string) appearanceSettingsResourceModel {
	return appearanceSettingsResourceModel{
		DateFormatID: types.StringValue(dateFormatID),
		TimeZoneID:   types.StringValue(timeZone),
	}
}

func makeAppearanceSettings(dateFormatID, timeZone string) youtrack.AppearanceSettings {
	return youtrack.AppearanceSettings{
		DateFormat: youtrack.DateFormatDescriptor{
			ID: dateFormatID,
		},
		TimeZone: youtrack.TimeZoneDescriptor{
			ID: timeZone,
		},
	}
}

func TestConvertModelToAppearanceSettings(t *testing.T) {
	tests := []struct {
		name         string
		dateFormatID string
		timeZone     string
	}{
		{"converts all required fields correctly", dateFormatID, timeZoneID},
		{"converts UTC timezone correctly", "youtrack.datefieldformat.medium_with_24h", "UTC"},
		{"converts American timezone correctly", "youtrack.datefieldformat.short", "America/New_York"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := makeModel(tt.dateFormatID, tt.timeZone)
			want := makeAppearanceSettings(tt.dateFormatID, tt.timeZone)
			got := convertModelToAppearanceSettings(model)

			helpers.AssertFieldEqual(t, "DateFormat.ID", got.DateFormat.ID, want.DateFormat.ID)
			helpers.AssertFieldEqual(t, "TimeZone.ID", got.TimeZone.ID, want.TimeZone.ID)
		})
	}
}

func TestConvertAppearanceSettingsToModel(t *testing.T) {
	tests := []struct {
		name             string
		id               string
		dateFormatID     string
		datePattern      string
		datePresentation string
		timeZoneID       string
		timeZoneLabel    string
		offset           int
	}{
		{
			name:             "converts all fields correctly",
			id:               "appearance-1",
			dateFormatID:     dateFormatID,
			datePattern:      dateFormatPattern,
			datePresentation: "2025-03-24",
			timeZoneID:       timeZoneID,
			timeZoneLabel:    timeZonePresentation,
			offset:           timeZoneOffset,
		},
		{
			name:             "converts UTC timezone correctly",
			id:               "appearance-2",
			dateFormatID:     "youtrack.datefieldformat.medium_with_24h",
			datePattern:      "d MMM yyyy HH:mm",
			datePresentation: "24 Mar 2025 10:00",
			timeZoneID:       "UTC",
			timeZoneLabel:    "Coordinated Universal Time",
			offset:           0,
		},
		{
			name:             "converts negative timezone offset correctly",
			id:               "appearance-3",
			dateFormatID:     "youtrack.datefieldformat.short",
			datePattern:      "MM/dd/yyyy",
			datePresentation: "03/24/2025",
			timeZoneID:       "America/New_York",
			timeZoneLabel:    "Eastern Standard Time",
			offset:           -18000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as := youtrack.AppearanceSettings{
				ID: tt.id,
				DateFormat: youtrack.DateFormatDescriptor{
					ID:           tt.dateFormatID,
					Presentation: tt.datePresentation,
					Pattern:      tt.datePattern,
					DatePattern:  tt.datePattern,
				},
				TimeZone: youtrack.TimeZoneDescriptor{
					ID:           tt.timeZoneID,
					Presentation: tt.timeZoneLabel,
					Offset:       tt.offset,
				},
			}

			got := convertAppearanceSettingsToModel(as)

			helpers.AssertFieldEqual(t, "ID", got.ID.ValueString(), tt.id)
			helpers.AssertFieldEqual(t, "DateFormatID", got.DateFormatID.ValueString(), tt.dateFormatID)
			helpers.AssertFieldEqual(t, "DateFormatPresentation", got.DateFormatPresentation.ValueString(), tt.datePresentation)
			helpers.AssertFieldEqual(t, "DateFormatPattern", got.DateFormatPattern.ValueString(), tt.datePattern)
			helpers.AssertFieldEqual(t, "DateFormatDatePattern", got.DateFormatDatePattern.ValueString(), tt.datePattern)
			helpers.AssertFieldEqual(t, "TimeZoneID", got.TimeZoneID.ValueString(), tt.timeZoneID)
			helpers.AssertFieldEqual(t, "TimeZonePresentation", got.TimeZonePresentation.ValueString(), tt.timeZoneLabel)
			helpers.AssertFieldEqual(t, "TimeZoneOffset", got.TimeZoneOffset.ValueInt64(), int64(tt.offset))
		})
	}
}
