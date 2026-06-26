package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testNestedGroupID   = "group-1"
	testNestedGroupName = "Main Group"
	testNestedGroupDesc = "Main group description"
	testSubGroupA       = "Team A"
	testSubGroupB       = "Team B"
	testUserLoginA      = "alice"
	testUserLoginB      = "bob"
	testViewerLogin     = "viewer-user"
	testUpdaterGroup    = "Editors"
	testUserIDA         = "user-1"
	testUserIDB         = "user-2"
	testViewerIDA       = "viewer-1"
	testUpdaterGroupID  = "group-updater-1"
	testSubGroupIDA     = "sub-1"
	testSubGroupIDB     = "sub-2"

	apiPathUsers  = "/api/users"
	apiPathGroups = "/api/groups"

	errUnexpectedDiagnosticsFmt = "unexpected diagnostics errors: %v"
	testAutoJoinDomain          = "example.com"
)

func TestSetToStrings(t *testing.T) {
	tests := []struct {
		name   string
		input  types.Set
		want   []string
		wantOK bool
	}{
		{
			name:   "null set returns empty",
			input:  types.SetNull(types.StringType),
			want:   []string{},
			wantOK: true,
		},
		{
			name:   "unknown set returns empty",
			input:  types.SetUnknown(types.StringType),
			want:   []string{},
			wantOK: true,
		},
		{
			name:   "trims deduplicates and sorts",
			input:  mustStringSet(t, []string{" " + testUserLoginB, testUserLoginA + " ", " " + testUserLoginA + " ", ""}),
			want:   []string{testUserLoginA, testUserLoginB},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diagnostics diag.Diagnostics

			got, ok := helpers.SetToStringSlice(context.Background(), tt.input, &diagnostics)

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if diagnostics.HasError() {
				t.Fatalf(errUnexpectedDiagnosticsFmt, diagnostics)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("setToStrings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapAPIToModel(t *testing.T) {
	resource := &nestedGroupResource{}
	model := nestedGroupResourceModel{}
	var diagnostics diag.Diagnostics

	apiGroup := &youtrack.NestedGroup{
		ID:                   testNestedGroupID,
		Name:                 testNestedGroupName,
		Description:          testNestedGroupDesc,
		RequireTwoFactorAuth: true,
		AutoJoin:             true,
		AutoJoinDomain:       testAutoJoinDomain,
		OwnUsers: []youtrack.User{
			{Login: testUserLoginB},
			{Login: ""},
			{Login: testUserLoginA},
		},
		SubGroups: []youtrack.NestedGroup{
			{Name: testSubGroupB},
			{Name: ""},
			{Name: testSubGroupA},
		},
		Viewers: []youtrack.Holder{
			{Login: testViewerLogin},
		},
		Updaters: []youtrack.Holder{
			{Name: testUpdaterGroup},
		},
	}

	resource.mapAPIToModel(context.Background(), apiGroup, &model, &diagnostics)

	if diagnostics.HasError() {
		t.Fatalf(errUnexpectedDiagnosticsFmt, diagnostics)
	}
	if model.ID.ValueString() != testNestedGroupID {
		t.Fatalf("model ID = %q, want %q", model.ID.ValueString(), testNestedGroupID)
	}
	if model.Name.ValueString() != testNestedGroupName {
		t.Fatalf("model Name = %q, want %q", model.Name.ValueString(), testNestedGroupName)
	}
	if model.Description.ValueString() != testNestedGroupDesc {
		t.Fatalf("model Description = %q, want %q", model.Description.ValueString(), testNestedGroupDesc)
	}

	ownUsers := mustStringsFromSet(t, model.OwnUserLogins)
	if !slices.Equal(ownUsers, []string{testUserLoginA, testUserLoginB}) {
		t.Fatalf("own_user_logins = %v, want %v", ownUsers, []string{testUserLoginA, testUserLoginB})
	}

	subGroups := mustStringsFromSet(t, model.SubGroupNames)
	if !slices.Equal(subGroups, []string{testSubGroupA, testSubGroupB}) {
		t.Fatalf("sub_group_names = %v, want %v", subGroups, []string{testSubGroupA, testSubGroupB})
	}

	if !model.RequireTwoFactorAuthentication.ValueBool() {
		t.Fatal("require_two_factor_authentication = false, want true")
	}

	if !model.AutoJoin.ValueBool() {
		t.Fatal("auto_join = false, want true")
	}

	if model.AutoJoinDomain.ValueString() != testAutoJoinDomain {
		t.Fatalf("auto_join_domain = %q, want %q", model.AutoJoinDomain.ValueString(), testAutoJoinDomain)
	}

	viewers := mustStringsFromSet(t, model.Viewers)
	if !slices.Equal(viewers, []string{testViewerLogin}) {
		t.Fatalf("viewers = %v, want %v", viewers, []string{testViewerLogin})
	}

	updaters := mustStringsFromSet(t, model.Updaters)
	if !slices.Equal(updaters, []string{testUpdaterGroup}) {
		t.Fatalf("updaters = %v, want %v", updaters, []string{testUpdaterGroup})
	}
}

type resolveMembershipTestCase struct {
	name            string
	ownUserLogins   []string
	subGroupNames   []string
	groupID         string
	viewers         []string
	updaters        []string
	autoJoin        bool
	autoJoinDomain  string
	require2FA      bool
	description     string
	transientLookup bool
	wantOK          bool
	wantDiagnostics bool
	wantUserIDs     []string
	wantSubGroupIDs []string
	wantViewerIDs   []string
	wantUpdaterIDs  []string
}

func TestResolveMembership(t *testing.T) {
	tests := []resolveMembershipTestCase{
		{
			name:            "resolves users and subgroups",
			ownUserLogins:   []string{" " + testUserLoginB + " ", testUserLoginA},
			subGroupNames:   []string{testSubGroupB, " " + testSubGroupA + " "},
			groupID:         testNestedGroupID,
			viewers:         []string{testViewerLogin},
			updaters:        []string{testUpdaterGroup},
			autoJoin:        true,
			autoJoinDomain:  testAutoJoinDomain,
			require2FA:      true,
			description:     testNestedGroupDesc,
			transientLookup: false,
			wantOK:          true,
			wantDiagnostics: false,
			wantUserIDs:     []string{testUserIDA, testUserIDB},
			wantSubGroupIDs: []string{testSubGroupIDA, testSubGroupIDB},
			wantViewerIDs:   []string{testViewerIDA},
			wantUpdaterIDs:  []string{testUpdaterGroupID},
		},
		{
			name:            "retries subgroup lookup until available",
			ownUserLogins:   []string{},
			subGroupNames:   []string{testSubGroupA},
			groupID:         testNestedGroupID,
			viewers:         []string{},
			updaters:        []string{},
			autoJoin:        false,
			autoJoinDomain:  "",
			require2FA:      false,
			description:     "",
			transientLookup: true,
			wantOK:          true,
			wantDiagnostics: false,
			wantUserIDs:     []string{},
			wantSubGroupIDs: []string{testSubGroupIDA},
			wantViewerIDs:   []string{},
			wantUpdaterIDs:  []string{},
		},
		{
			name:            "self subgroup is rejected",
			ownUserLogins:   []string{},
			subGroupNames:   []string{testNestedGroupName},
			groupID:         testNestedGroupID,
			viewers:         []string{},
			updaters:        []string{},
			autoJoin:        false,
			autoJoinDomain:  "",
			require2FA:      false,
			description:     "",
			transientLookup: false,
			wantOK:          false,
			wantDiagnostics: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runResolveMembershipCase(t, tt)
		})
	}
}

func runResolveMembershipCase(t *testing.T, tt resolveMembershipTestCase) {
	t.Helper()

	server := newResolveMembershipServer(t, tt)
	defer server.Close()

	client, err := youtrack.NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	resource := &nestedGroupResource{client: client}
	plan := buildResolveMembershipPlan(t, tt)

	var diagnostics diag.Diagnostics
	resolved, ok := resource.resolveMembership(context.Background(), plan, tt.groupID, &diagnostics)

	if ok != tt.wantOK {
		t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
	}

	if diagnostics.HasError() != tt.wantDiagnostics {
		t.Fatalf("diagnostics.HasError() = %v, want %v", diagnostics.HasError(), tt.wantDiagnostics)
	}

	if !tt.wantOK {
		if resolved != nil {
			t.Fatalf("resolved should be nil on failure, got: %+v", resolved)
		}
		return
	}

	assertResolvedMembership(t, resolved, tt)
}

func newResolveMembershipServer(t *testing.T, tt resolveMembershipTestCase) *httptest.Server {
	t.Helper()

	groupsRequests := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case apiPathUsers:
			_, _ = w.Write([]byte(`[
				{"id":"` + testUserIDA + `","login":"` + testUserLoginA + `","name":"Alice"},
				{"id":"` + testUserIDB + `","login":"` + testUserLoginB + `","name":"Bob"},
				{"id":"` + testViewerIDA + `","login":"` + testViewerLogin + `","name":"Viewer"}
			]`))
		case apiPathGroups:
			groupsRequests++
			if tt.transientLookup && groupsRequests == 1 {
				_, _ = w.Write([]byte(`[
					{"id":"` + testNestedGroupID + `","name":"` + testNestedGroupName + `"},
					{"id":"` + testSubGroupIDB + `","name":"` + testSubGroupB + `"}
				]`))
				return
			}
			_, _ = w.Write([]byte(`[
				{"id":"` + testNestedGroupID + `","name":"` + testNestedGroupName + `"},
				{"id":"` + testSubGroupIDA + `","name":"` + testSubGroupA + `"},
				{"id":"` + testSubGroupIDB + `","name":"` + testSubGroupB + `"},
				{"id":"` + testUpdaterGroupID + `","name":"` + testUpdaterGroup + `"}
			]`))
		default:
			http.NotFound(w, req)
		}
	}))
}

func buildResolveMembershipPlan(t *testing.T, tt resolveMembershipTestCase) *nestedGroupResourceModel {
	t.Helper()

	return &nestedGroupResourceModel{
		Name:                           types.StringValue(testNestedGroupName),
		Description:                    types.StringValue(tt.description),
		OwnUserLogins:                  mustStringSet(t, tt.ownUserLogins),
		SubGroupNames:                  mustStringSet(t, tt.subGroupNames),
		Viewers:                        mustStringSet(t, tt.viewers),
		Updaters:                       mustStringSet(t, tt.updaters),
		AutoJoin:                       types.BoolValue(tt.autoJoin),
		AutoJoinDomain:                 types.StringValue(tt.autoJoinDomain),
		RequireTwoFactorAuthentication: types.BoolValue(tt.require2FA),
	}
}

func assertResolvedMembership(t *testing.T, resolved *youtrack.NestedGroup, tt resolveMembershipTestCase) {
	t.Helper()

	if resolved == nil {
		t.Fatal("resolved result is nil")
		return
	}
	if resolved.Name != testNestedGroupName {
		t.Fatalf("resolved.Name = %q, want %q", resolved.Name, testNestedGroupName)
	}
	if resolved.Description != tt.description {
		t.Fatalf("resolved.Description = %q, want %q", resolved.Description, tt.description)
	}
	if resolved.Viewers == nil {
		t.Fatal("resolved.Viewers should not be nil")
	}
	if resolved.Updaters == nil {
		t.Fatal("resolved.Updaters should not be nil")
	}
	if resolved.Users == nil {
		t.Fatal("resolved.Users should not be nil")
	}
	if resolved.RequireTwoFactorAuth != tt.require2FA {
		t.Fatalf("resolved.RequireTwoFactorAuth = %v, want %v", resolved.RequireTwoFactorAuth, tt.require2FA)
	}
	if resolved.AutoJoin != tt.autoJoin {
		t.Fatalf("resolved.AutoJoin = %v, want %v", resolved.AutoJoin, tt.autoJoin)
	}
	if resolved.AutoJoinDomain != tt.autoJoinDomain {
		t.Fatalf("resolved.AutoJoinDomain = %q, want %q", resolved.AutoJoinDomain, tt.autoJoinDomain)
	}

	assertResolvedUserIDs(t, resolved, tt.wantUserIDs)
	assertResolvedSubGroupIDs(t, resolved, tt.wantSubGroupIDs)
	assertResolvedViewerIDs(t, resolved, tt.wantViewerIDs)
	assertResolvedUpdaterIDs(t, resolved, tt.wantUpdaterIDs)
}

func assertResolvedUserIDs(t *testing.T, resolved *youtrack.NestedGroup, want []string) {
	t.Helper()

	userIDs := make([]string, 0, len(resolved.OwnUsers))
	for _, user := range resolved.OwnUsers {
		userIDs = append(userIDs, user.ID)
	}

	if !slices.Equal(userIDs, want) {
		t.Fatalf("resolved user IDs = %v, want %v", userIDs, want)
	}
}

func assertResolvedSubGroupIDs(t *testing.T, resolved *youtrack.NestedGroup, want []string) {
	t.Helper()

	subGroupIDs := make([]string, 0, len(resolved.SubGroups))
	for _, group := range resolved.SubGroups {
		subGroupIDs = append(subGroupIDs, group.ID)
	}

	if !slices.Equal(subGroupIDs, want) {
		t.Fatalf("resolved subgroup IDs = %v, want %v", subGroupIDs, want)
	}
}

func assertResolvedViewerIDs(t *testing.T, resolved *youtrack.NestedGroup, want []string) {
	t.Helper()

	viewerIDs := make([]string, 0, len(resolved.Viewers))
	for _, viewer := range resolved.Viewers {
		viewerIDs = append(viewerIDs, viewer.Id)
		if strings.TrimSpace(viewer.Type) == "" {
			t.Fatal("resolved viewer type should not be empty")
		}
	}

	if !slices.Equal(viewerIDs, want) {
		t.Fatalf("resolved viewer IDs = %v, want %v", viewerIDs, want)
	}
}

func assertResolvedUpdaterIDs(t *testing.T, resolved *youtrack.NestedGroup, want []string) {
	t.Helper()

	updaterIDs := make([]string, 0, len(resolved.Updaters))
	for _, updater := range resolved.Updaters {
		updaterIDs = append(updaterIDs, updater.Id)
		if strings.TrimSpace(updater.Type) == "" {
			t.Fatal("resolved updater type should not be empty")
		}
	}

	if !slices.Equal(updaterIDs, want) {
		t.Fatalf("resolved updater IDs = %v, want %v", updaterIDs, want)
	}
}

func TestNewNestedGroupPayload(t *testing.T) {
	payload := newNestedGroupPayload(testNestedGroupName)

	if payload.Name != testNestedGroupName {
		t.Fatalf("payload.Name = %q, want %q", payload.Name, testNestedGroupName)
	}
	if payload.OwnUsers == nil {
		t.Fatal("payload.OwnUsers should not be nil")
	}
	if payload.SubGroups == nil {
		t.Fatal("payload.SubGroups should not be nil")
	}
	if payload.Viewers == nil {
		t.Fatal("payload.Viewers should not be nil")
	}
	if payload.Updaters == nil {
		t.Fatal("payload.Updaters should not be nil")
	}
	if payload.Users == nil {
		t.Fatal("payload.Users should not be nil")
	}
}

func mustStringSet(t *testing.T, values []string) types.Set {
	t.Helper()

	setValue, diagnostics := types.SetValueFrom(context.Background(), types.StringType, values)
	if diagnostics.HasError() {
		t.Fatalf("failed to construct set from values %v: %v", values, diagnostics)
	}

	return setValue
}

func mustStringsFromSet(t *testing.T, setValue types.Set) []string {
	t.Helper()

	var diagnostics diag.Diagnostics
	values, ok := helpers.SetToStringSlice(context.Background(), setValue, &diagnostics)
	if !ok {
		t.Fatalf("setToStrings returned false with diagnostics: %v", diagnostics)
	}
	if diagnostics.HasError() {
		t.Fatalf(errUnexpectedDiagnosticsFmt, diagnostics)
	}

	return values
}
