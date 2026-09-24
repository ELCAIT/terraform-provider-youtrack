package provider

import (
	"reflect"
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testOwnedValueBackend  = "Backend"
	testOwnedValueFrontend = "Frontend"
	testOwnerLogin         = "jane.doe"
	testOwnerID            = "1-1"
)

func testOwnedValue(name, ownerLogin string) ownedBundleValueModel {
	login := types.StringNull()
	if ownerLogin != "" {
		login = types.StringValue(ownerLogin)
	}

	return ownedBundleValueModel{
		Name:        types.StringValue(name),
		Description: types.StringUnknown(),
		Archived:    types.BoolValue(false),
		OwnerLogin:  login,
	}
}

func TestOwnedBundleModelToAPIModel(t *testing.T) {
	t.Parallel()

	model := ownedBundleResourceModel{
		Name: types.StringValue("Subsystems"),
		Values: []ownedBundleValueModel{
			testOwnedValue(testOwnedValueBackend, "Jane.Doe"),
			testOwnedValue(testOwnedValueFrontend, ""),
		},
	}
	owners := ownerRefs{testOwnerLogin: {ID: testOwnerID}}

	apiModel := model.toAPIModel(owners)
	helpers.AssertFieldEqual(t, "Name", apiModel.Name, "Subsystems")
	helpers.AssertFieldEqual(t, "ValuesLength", len(apiModel.Values), 2)
	helpers.AssertFieldEqual(t, "FirstOwnerID", ownerID(apiModel.Values[0].Owner), testOwnerID)
	helpers.AssertFieldEqual(t, "SecondOwnerID", ownerID(apiModel.Values[1].Owner), "")
}

func TestOwnedValueUpdate(t *testing.T) {
	t.Parallel()

	current := youtrack.OwnedBundleElement{
		ID:          "49-1",
		Name:        testOwnedValueBackend,
		Description: "Server side",
		Ordinal:     1,
		Owner:       &youtrack.UserRef{ID: testOwnerID, Login: testOwnerLogin},
	}
	owners := ownerRefs{testOwnerLogin: {ID: testOwnerID}}

	tests := []struct {
		name            string
		value           ownedBundleValueModel
		ordinal         int
		wantDescription string
		wantOwnerID     string
		wantUpdate      bool
	}{
		{
			name:            "unchanged value needs no update",
			value:           testOwnedValue(testOwnedValueBackend, testOwnerLogin),
			ordinal:         1,
			wantDescription: "Server side",
			wantOwnerID:     testOwnerID,
		},
		{
			name:            "removed owner is cleared",
			value:           testOwnedValue(testOwnedValueBackend, ""),
			ordinal:         1,
			wantDescription: "Server side",
			wantUpdate:      true,
		},
		{
			name: "configured null description is cleared",
			value: func() ownedBundleValueModel {
				v := testOwnedValue(testOwnedValueBackend, testOwnerLogin)
				v.Description = types.StringNull()
				return v
			}(),
			ordinal:     1,
			wantOwnerID: testOwnerID,
			wantUpdate:  true,
		},
		{
			name:            "moved value gets its new ordinal",
			value:           testOwnedValue(testOwnedValueBackend, testOwnerLogin),
			ordinal:         2,
			wantDescription: "Server side",
			wantOwnerID:     testOwnerID,
			wantUpdate:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			update := tc.value.toValueUpdate(current, owners, tc.ordinal)
			description := ""
			if update.Description != nil {
				description = *update.Description
			}

			helpers.AssertFieldEqual(t, "Description", description, tc.wantDescription)
			helpers.AssertFieldEqual(t, "OwnerID", ownerID(update.Owner), tc.wantOwnerID)
			helpers.AssertFieldEqual(t, "Ordinal", *update.Ordinal, tc.ordinal)
			helpers.AssertFieldEqual(t, "NeedsUpdate", !reflect.DeepEqual(update, ownedValueAsUpdate(current)), tc.wantUpdate)
		})
	}
}

func TestOwnedBundleModelFromAPIModel(t *testing.T) {
	t.Parallel()

	apiModel := &youtrack.OwnedBundle{
		ID:           "48-1",
		Name:         "Subsystems",
		IsUpdateable: true,
		Values: []youtrack.OwnedBundleElement{
			{ID: "49-2", Name: testOwnedValueFrontend, Ordinal: 2},
			{ID: "49-1", Name: testOwnedValueBackend, Ordinal: 1, Owner: &youtrack.UserRef{ID: testOwnerID, Login: testOwnerLogin}},
		},
	}
	prior := []ownedBundleValueModel{testOwnedValue(testOwnedValueBackend, "Jane.Doe")}

	var model ownedBundleResourceModel
	model.fromAPIModel(apiModel, prior)

	helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), "48-1")
	helpers.AssertFieldEqual(t, "IsUpdateable", model.IsUpdateable.ValueBool(), true)
	helpers.AssertFieldEqual(t, "FirstValueByOrdinal", model.Values[0].Name.ValueString(), testOwnedValueBackend)
	helpers.AssertFieldEqual(t, "OwnerKeepsConfiguredSpelling", model.Values[0].OwnerLogin.ValueString(), "Jane.Doe")
	helpers.AssertFieldEqual(t, "UnownedValueIsNull", model.Values[1].OwnerLogin.IsNull(), true)
	helpers.AssertFieldEqual(t, "EmptyDescriptionIsNull", model.Values[1].Description.IsNull(), true)
}

func ownerID(owner *youtrack.UserRef) string {
	if owner == nil {
		return ""
	}
	return owner.ID
}
