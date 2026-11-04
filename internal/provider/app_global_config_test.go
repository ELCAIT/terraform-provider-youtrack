// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testSettingsWritten = `{"valueManagers":[{"id":"4-1"}],"apiToken":"perm-secret","youtrackUrl":"http://youtrack:8080","archiveDirectly":true}`

	// testSettingsRead is what YouTrack answers for testSettingsWritten: the group
	// is expanded and the secret is masked.
	testSettingsRead = `{"valueManagers":[{"$type":"NestedGroup","id":"4-1","ringId":"abc","name":"admins","icon":null,"allUsersGroup":false}],` +
		`"apiToken":"<***>","youtrackUrl":"http://youtrack:8080","archiveDirectly":true}`
)

func TestReconcileSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		written string
		read    string
		want    map[string]any // nil means the written document is kept as it is
		wantErr bool
	}{
		{name: "expanded entity and masked secret hold", written: testSettingsWritten, read: testSettingsRead},
		{
			name:    "changed scalar is reported",
			written: testSettingsWritten,
			read:    `{"valueManagers":[{"id":"4-1"}],"apiToken":"<***>","youtrackUrl":"http://other:8080","archiveDirectly":true}`,
			want: map[string]any{
				"valueManagers":   []any{map[string]any{"id": "4-1"}},
				"apiToken":        "perm-secret",
				"youtrackUrl":     "http://other:8080",
				"archiveDirectly": true,
			},
		},
		{
			name:    "changed entity is reported",
			written: testSettingsWritten,
			read:    `{"valueManagers":[{"id":"4-2","name":"other"}],"apiToken":"<***>","youtrackUrl":"http://youtrack:8080","archiveDirectly":true}`,
			want: map[string]any{
				"valueManagers":   []any{map[string]any{"id": "4-2", "name": "other"}},
				"apiToken":        "perm-secret",
				"youtrackUrl":     "http://youtrack:8080",
				"archiveDirectly": true,
			},
		},
		{
			name:    "key missing in YouTrack is dropped from state",
			written: `{"youtrackUrl":"http://youtrack:8080","archiveDirectly":true}`,
			read:    `{"youtrackUrl":"http://youtrack:8080"}`,
			want:    map[string]any{"youtrackUrl": "http://youtrack:8080"},
		},
		{name: "null left out by YouTrack holds", written: `{"youtrackUrl":"http://youtrack:8080","webhookUrl":null}`, read: `{"youtrackUrl":"http://youtrack:8080"}`},
		{name: "empty string left out by YouTrack holds", written: `{"youtrackUrl":"http://youtrack:8080","webhookUrl":""}`, read: `{"youtrackUrl":"http://youtrack:8080"}`},
		{
			name:    "empty string set in YouTrack is reported",
			written: `{"webhookUrl":""}`,
			read:    `{"webhookUrl":"http://other"}`,
			want:    map[string]any{"webhookUrl": "http://other"},
		},
		{name: "entities in another order hold", written: `{"groups":[{"id":"4-1"},{"id":"4-0"}]}`, read: `{"groups":[{"id":"4-0","name":"a"},{"id":"4-1","name":"b"}]}`},
		{name: "duplicate entity merged by YouTrack holds", written: `{"groups":[{"id":"4-0"},{"id":"4-1"},{"id":"4-1"}]}`, read: `{"groups":[{"id":"4-0"},{"id":"4-1"}]}`},
		{
			name:    "missing entity is reported",
			written: `{"groups":[{"id":"4-0"},{"id":"4-1"}]}`,
			read:    `{"groups":[{"id":"4-0"}]}`,
			want:    map[string]any{"groups": []any{map[string]any{"id": "4-0"}}},
		},
		{
			name:    "entity list emptied in YouTrack is reported",
			written: `{"groups":[{"id":"4-0"}]}`,
			read:    `{"groups":[]}`,
			want:    map[string]any{"groups": []any{}},
		},
		{
			name:    "plain values keep their order",
			written: `{"tags":["a","b"]}`,
			read:    `{"tags":["b","a"]}`,
			want:    map[string]any{"tags": []any{"b", "a"}},
		},
		{name: "key not written is ignored", written: `{"youtrackUrl":"http://youtrack:8080"}`, read: `{"youtrackUrl":"http://youtrack:8080","extra":1}`},
		{name: "settings cleared in YouTrack", written: `{"youtrackUrl":"http://youtrack:8080"}`, read: ``, want: map[string]any{}},
		{name: "invalid state", written: `not json`, read: testSettingsRead, wantErr: true},
		{name: "invalid answer", written: testSettingsWritten, read: `not json`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := reconcileSettings(tc.written, tc.read)
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected error state: %v", err)
			}
			if tc.wantErr {
				return
			}

			if tc.want == nil {
				if got != tc.written {
					t.Fatalf("expected the written settings to be kept, got %s", got)
				}
				return
			}

			var doc map[string]any
			if err = json.Unmarshal([]byte(got), &doc); err != nil {
				t.Fatalf("result is not JSON: %v", err)
			}
			if !reflect.DeepEqual(doc, tc.want) {
				t.Fatalf("unexpected settings: got %v, want %v", doc, tc.want)
			}
		})
	}
}

func TestAppGlobalConfigFromAPIModel(t *testing.T) {
	t.Parallel()

	config := &youtrack.AppGlobalConfig{
		Enabled:        false,
		GlobalSettings: json.RawMessage(testSettingsRead),
		VisibilitySettings: &youtrack.AppVisibilitySettings{
			PermittedGroups: []youtrack.AppVisibilityGroup{{ID: "6-1"}, {ID: "6-0"}},
		},
	}

	t.Run("manages what the configuration manages", func(t *testing.T) {
		t.Parallel()

		set, _ := types.SetValueFrom(context.Background(), types.StringType, []string{"6-0"})
		m := appGlobalConfigResourceModel{
			AppName:           types.StringValue("my-app"),
			Enabled:           types.BoolValue(true),
			Settings:          types.StringValue(testSettingsWritten),
			PermittedGroupIDs: set,
		}

		if err := m.fromAPIModel(context.Background(), "151-1", config); err != nil {
			t.Fatal(err)
		}

		if m.Enabled.ValueBool() || m.ID.ValueString() != "151-1" {
			t.Fatalf("unexpected model: %+v", m)
		}
		if m.Settings.ValueString() != testSettingsWritten {
			t.Fatalf("settings drifted without a difference: %s", m.Settings.ValueString())
		}

		var ids []string
		m.PermittedGroupIDs.ElementsAs(context.Background(), &ids, false)
		if !reflect.DeepEqual(ids, []string{"6-0", "6-1"}) {
			t.Fatalf("unexpected permitted groups: %v", ids)
		}
	})

	t.Run("no permitted groups is an empty set, not null", func(t *testing.T) {
		t.Parallel()

		empty, _ := types.SetValueFrom(context.Background(), types.StringType, []string{})
		m := appGlobalConfigResourceModel{
			AppName:           types.StringValue("my-app"),
			Settings:          types.StringNull(),
			PermittedGroupIDs: empty,
		}

		if err := m.fromAPIModel(context.Background(), "151-1", &youtrack.AppGlobalConfig{}); err != nil {
			t.Fatal(err)
		}

		if m.PermittedGroupIDs.IsNull() || len(m.PermittedGroupIDs.Elements()) != 0 {
			t.Fatalf("expected an empty set, got %v", m.PermittedGroupIDs)
		}
	})

	t.Run("leaves unmanaged attributes null", func(t *testing.T) {
		t.Parallel()

		m := appGlobalConfigResourceModel{
			AppName:           types.StringValue("my-app"),
			Settings:          types.StringNull(),
			PermittedGroupIDs: types.SetNull(types.StringType),
		}

		if err := m.fromAPIModel(context.Background(), "151-1", config); err != nil {
			t.Fatal(err)
		}

		if !m.Settings.IsNull() || !m.PermittedGroupIDs.IsNull() {
			t.Fatalf("unmanaged attributes were filled in: %+v", m)
		}
	})
}

func TestAppGlobalConfigWritesOnlyWhatChanged(t *testing.T) {
	t.Parallel()

	groups := func(ids ...string) types.Set {
		set, _ := types.SetValueFrom(context.Background(), types.StringType, ids)
		return set
	}
	model := func(settings types.String, groupIDs types.Set) *appGlobalConfigResourceModel {
		return &appGlobalConfigResourceModel{Settings: settings, PermittedGroupIDs: groupIDs}
	}
	unmanagedGroups := types.SetNull(types.StringType)

	tests := []struct {
		name         string
		plan         *appGlobalConfigResourceModel
		prior        *appGlobalConfigResourceModel
		wantSettings bool
		wantGroups   bool
	}{
		{name: "create writes what is managed", plan: model(types.StringValue(`{"a":1}`), groups("6-0")), wantSettings: true, wantGroups: true},
		{name: "create writes nothing unmanaged", plan: model(types.StringNull(), unmanagedGroups)},
		{
			name:  "an unrelated change writes neither",
			plan:  model(types.StringValue(`{"a":1}`), groups("6-0")),
			prior: model(types.StringValue(`{"a":1}`), groups("6-0")),
		},
		{
			name:         "changed settings are written",
			plan:         model(types.StringValue(`{"a":2}`), groups("6-0")),
			prior:        model(types.StringValue(`{"a":1}`), groups("6-0")),
			wantSettings: true,
		},
		{
			name:       "changed groups are written",
			plan:       model(types.StringValue(`{"a":1}`), groups("6-0", "4-1")),
			prior:      model(types.StringValue(`{"a":1}`), groups("6-0")),
			wantGroups: true,
		},
		{
			name:  "unmanaged settings are never written",
			plan:  model(types.StringNull(), unmanagedGroups),
			prior: model(types.StringValue(`{"a":1}`), groups("6-0")),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			helpers.AssertFieldEqual(t, "settingsChanged", tc.plan.settingsChanged(tc.prior), tc.wantSettings)
			helpers.AssertFieldEqual(t, "groupsChanged", tc.plan.groupsChanged(tc.prior), tc.wantGroups)
		})
	}
}

func TestAppGlobalConfigCheckNotReinstalled(t *testing.T) {
	t.Parallel()

	state := func(appID types.String) *appGlobalConfigResourceModel {
		return &appGlobalConfigResourceModel{ID: appID}
	}

	tests := []struct {
		name    string
		prior   *appGlobalConfigResourceModel
		wantErr bool
	}{
		{name: "create has no prior state", prior: nil},
		{name: "same app", prior: state(types.StringValue("151-1"))},
		{name: "no app ID in state", prior: state(types.StringNull())},
		{name: "app reinstalled under a new ID", prior: state(types.StringValue("151-0")), wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := appGlobalConfigResourceModel{AppName: types.StringValue("my-app")}
			err := m.checkNotReinstalled(tc.prior, "151-1")
			helpers.AssertFieldEqual(t, "error", err != nil, tc.wantErr)
		})
	}
}
