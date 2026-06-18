// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &projectCustomFieldResource{}
	_ resource.ResourceWithConfigure   = &projectCustomFieldResource{}
	_ resource.ResourceWithImportState = &projectCustomFieldResource{}
)

const (
	errCreatingProjectCustomField  = "Error attaching project custom field"
	errReadingProjectCustomField   = "Error reading project custom field"
	errUpdatingProjectCustomField  = "Error updating project custom field"
	errDeletingProjectCustomField  = "Error removing project custom field"
	errMissingProjectCustomFieldID = "Missing project custom field ID"
	errResolvingProjectCustomField = "Error resolving project custom field type"

	errProjectCustomFieldIDRequired      = "Project custom field ID is required"
	errProjectCustomFieldImportIDInvalid = "Invalid import ID format. Expected: {project_id}/{field_id}"
	errBundleTypeNotSupported            = "bundle lookup not supported for field type %q; supported: EnumProjectCustomField, StateProjectCustomField"
	errDefaultValuesTypeNotSupported     = "default_value_names is only supported for field type %q; supported: EnumProjectCustomField, StateProjectCustomField"
	errCannotDeriveProjectType           = "could not derive project custom field type from global custom field type %q; set field_type explicitly"
	errCannotDeriveProjectTypeFromID     = "could not derive project custom field type from global custom field fieldType.id %q; set field_type explicitly"

	projectCustomFieldImportSeparator = "/"
	projectCustomFieldImportParts     = 2
)

// NewProjectCustomFieldResource is a helper function to simplify the provider implementation.
func NewProjectCustomFieldResource() resource.Resource {
	return &projectCustomFieldResource{}
}

type projectCustomFieldResource struct {
	client *youtrack.Client
}

type projectCustomFieldResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectID         types.String `tfsdk:"project_id"`
	FieldName         types.String `tfsdk:"field_name"`
	FieldType         types.String `tfsdk:"field_type"`
	BundleName        types.String `tfsdk:"bundle_name"`
	CanBeEmpty        types.Bool   `tfsdk:"can_be_empty"`
	EmptyFieldText    types.String `tfsdk:"empty_field_text"`
	IsPublic          types.Bool   `tfsdk:"is_public"`
	DefaultValueNames types.List   `tfsdk:"default_value_names"`
}

func (r *projectCustomFieldResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_custom_field"
}

func (r *projectCustomFieldResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack project custom field resource. Attaches a global custom field to a project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The entity ID of the project custom field attachment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The entity ID of the project to attach the custom field to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"field_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the global custom field to attach to the project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"field_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The $type of the ProjectCustomField to create (e.g., EnumProjectCustomField, StateProjectCustomField). If omitted, it is derived from the global custom field type.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"can_be_empty": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the custom field can have an empty value.",
			},
			"empty_field_text": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The placeholder text shown when the field has an empty value.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_public": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether basic Read/Update Issue permissions are sufficient to access this field.",
			},
			"bundle_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the bundle to use for this project custom field. Overrides the default bundle from the global custom field. Supported for EnumProjectCustomField and StateProjectCustomField.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_value_names": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Default values set for new issues. Values are resolved by name from the effective bundle. Supported for EnumProjectCustomField and StateProjectCustomField.",
			},
		},
	}
}

func (r *projectCustomFieldResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

func (r *projectCustomFieldResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectCustomFieldResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	field, err := r.client.GetCustomFieldByName(ctx, plan.FieldName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			errCreatingProjectCustomField,
			fmt.Sprintf("Could not find custom field with name %q: %v", plan.FieldName.ValueString(), err),
		)
		return
	}

	if err := r.ensureFieldType(ctx, &plan, field); err != nil {
		resp.Diagnostics.AddError(errResolvingProjectCustomField, err.Error())
		return
	}

	bundle, err := r.lookupBundleByName(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(errCreatingProjectCustomField, err.Error())
		return
	}

	defaultValues, err := r.lookupDefaultValues(ctx, plan, field, bundle)
	if err != nil {
		resp.Diagnostics.AddError(errCreatingProjectCustomField, err.Error())
		return
	}

	payload := plan.toUpsertPayload(field.ID, bundle, defaultValues)
	created, err := r.client.AddProjectCustomField(ctx, plan.ProjectID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			errCreatingProjectCustomField,
			fmt.Sprintf("Could not attach custom field to project: %v", err),
		)
		return
	}

	plan.fromAPIModel(created)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *projectCustomFieldResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectCustomFieldResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingProjectCustomFieldID, errProjectCustomFieldIDRequired) {
		return
	}

	field, err := r.client.GetProjectCustomField(ctx, state.ProjectID.ValueString(), state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			errReadingProjectCustomField,
			fmt.Sprintf("Could not read project custom field: %v", err),
		)
		return
	}

	state.fromAPIModel(field)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *projectCustomFieldResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectCustomFieldResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	if !helpers.ValidateResourceID(plan.ID, &resp.Diagnostics, errMissingProjectCustomFieldID, errProjectCustomFieldIDRequired) {
		return
	}

	field, err := r.client.GetCustomFieldByName(ctx, plan.FieldName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			errUpdatingProjectCustomField,
			fmt.Sprintf("Could not find custom field with name %q: %v", plan.FieldName.ValueString(), err),
		)
		return
	}

	if err := r.ensureFieldType(ctx, &plan, field); err != nil {
		resp.Diagnostics.AddError(errResolvingProjectCustomField, err.Error())
		return
	}

	bundle, err := r.lookupBundleByName(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingProjectCustomField, err.Error())
		return
	}

	defaultValues, err := r.lookupDefaultValues(ctx, plan, field, bundle)
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingProjectCustomField, err.Error())
		return
	}

	payload := plan.toUpsertPayload(field.ID, bundle, defaultValues)
	updated, err := r.client.UpdateProjectCustomField(ctx, plan.ProjectID.ValueString(), plan.ID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			errUpdatingProjectCustomField,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "project custom field", err),
		)
		return
	}

	plan.fromAPIModel(updated)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *projectCustomFieldResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectCustomFieldResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.ID) {
		return
	}

	err := r.client.RemoveProjectCustomField(ctx, state.ProjectID.ValueString(), state.ID.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			errDeletingProjectCustomField,
			fmt.Sprintf("Could not remove custom field from project: %v", err),
		)
	}
}

func (r *projectCustomFieldResource) ensureFieldType(ctx context.Context, m *projectCustomFieldResourceModel, field *youtrack.CustomField) error {
	if m == nil {
		return fmt.Errorf("missing project custom field model")
	}

	if !m.FieldType.IsNull() && !m.FieldType.IsUnknown() && strings.TrimSpace(m.FieldType.ValueString()) != "" {
		return nil
	}

	if field == nil || strings.TrimSpace(field.ID) == "" {
		return fmt.Errorf("cannot derive field_type because global custom field ID is missing; set field_type explicitly")
	}

	fullField, err := r.client.GetCustomFieldByID(ctx, field.ID)
	if err != nil {
		return fmt.Errorf("could not read global custom field %q for type derivation: %w", field.ID, err)
	}

	derivedType, err := deriveProjectCustomFieldTypeFromFieldTypeID(fullField.FieldType.ID)
	if err != nil {
		// Fallback to legacy behavior for compatibility with older payload shapes.
		derivedType, err = deriveProjectCustomFieldType(fullField.Type)
	}
	if err != nil {
		return err
	}

	m.FieldType = types.StringValue(derivedType)
	return nil
}

func deriveProjectCustomFieldType(globalType string) (string, error) {
	t := strings.TrimSpace(globalType)
	if t == "" {
		return "", fmt.Errorf(errCannotDeriveProjectType, globalType)
	}

	if strings.HasSuffix(t, "ProjectCustomField") {
		return t, nil
	}

	if strings.HasSuffix(t, "IssueCustomField") {
		return strings.TrimSuffix(t, "IssueCustomField") + "ProjectCustomField", nil
	}

	return "", fmt.Errorf(errCannotDeriveProjectType, globalType)
}

func deriveProjectCustomFieldTypeFromFieldTypeID(fieldTypeID string) (string, error) {
	t := strings.ToLower(strings.TrimSpace(fieldTypeID))
	if t == "" {
		return "", fmt.Errorf(errCannotDeriveProjectTypeFromID, fieldTypeID)
	}

	if i := strings.Index(t, "["); i > 0 {
		t = t[:i]
	}

	switch t {
	case "enum":
		return "EnumProjectCustomField", nil
	case "state":
		return "StateProjectCustomField", nil
	case "period":
		return "PeriodProjectCustomField", nil
	case "build":
		return "BuildProjectCustomField", nil
	case "version":
		return "VersionProjectCustomField", nil
	case "ownedfield":
		return "OwnedProjectCustomField", nil
	case "user":
		return "UserProjectCustomField", nil
	case "group":
		return "GroupProjectCustomField", nil
	case "date":
		return "DateProjectCustomField", nil
	case "datetime":
		return "DateTimeProjectCustomField", nil
	case "simple", "string", "text", "integer", "float":
		return "SimpleProjectCustomField", nil
	default:
		return "", fmt.Errorf(errCannotDeriveProjectTypeFromID, fieldTypeID)
	}
}

// lookupBundleByName resolves a bundle name to a BundleRef for the given field type.
// Returns nil if bundle_name is not set. Only EnumProjectCustomField and StateProjectCustomField are supported.
func (r *projectCustomFieldResource) lookupBundleByName(ctx context.Context, m projectCustomFieldResourceModel) (*youtrack.BundleRef, error) {
	if m.BundleName.IsNull() || m.BundleName.IsUnknown() || m.BundleName.ValueString() == "" {
		return nil, nil
	}

	bundleName := m.BundleName.ValueString()
	fieldType := m.FieldType.ValueString()

	switch fieldType {
	case "EnumProjectCustomField":
		b, err := r.client.GetEnumBundleByName(ctx, bundleName)
		if err != nil {
			return nil, fmt.Errorf("could not find enum bundle with name %q: %w", bundleName, err)
		}
		return &youtrack.BundleRef{ID: b.ID, Type: bundleTypeEnum}, nil
	case "StateProjectCustomField":
		b, err := r.client.GetStateBundleByName(ctx, bundleName)
		if err != nil {
			return nil, fmt.Errorf("could not find state bundle with name %q: %w", bundleName, err)
		}
		return &youtrack.BundleRef{ID: b.ID, Type: bundleTypeState}, nil
	default:
		return nil, fmt.Errorf(errBundleTypeNotSupported, fieldType)
	}
}

func (r *projectCustomFieldResource) lookupDefaultValues(
	ctx context.Context,
	m projectCustomFieldResourceModel,
	field *youtrack.CustomField,
	bundle *youtrack.BundleRef,
) ([]youtrack.ProjectCustomFieldValueRef, error) {
	if m.DefaultValueNames.IsNull() || m.DefaultValueNames.IsUnknown() {
		return nil, nil
	}

	names, ok := helpers.ListToStringSlice(ctx, m.DefaultValueNames)
	if !ok {
		return nil, fmt.Errorf("failed to decode default_value_names")
	}

	trimmed := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		normalized := strings.TrimSpace(name)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		trimmed = append(trimmed, normalized)
	}

	if len(trimmed) == 0 {
		return nil, nil
	}

	bundleID := ""
	if bundle != nil {
		bundleID = strings.TrimSpace(bundle.ID)
	}
	if bundleID == "" && field != nil && field.FieldDefaults != nil && field.FieldDefaults.Bundle != nil {
		bundleID = strings.TrimSpace(field.FieldDefaults.Bundle.ID)
	}
	if bundleID == "" {
		return nil, fmt.Errorf("cannot resolve default_value_names because no bundle is configured; set bundle_name or configure a default bundle on the global custom field")
	}

	fieldType := m.FieldType.ValueString()
	refs := make([]youtrack.ProjectCustomFieldValueRef, 0, len(trimmed))

	switch fieldType {
	case "EnumProjectCustomField":
		enumBundle, err := r.client.GetEnumBundleByID(ctx, bundleID)
		if err != nil {
			return nil, fmt.Errorf("could not load enum bundle %q for default_value_names: %w", bundleID, err)
		}

		byName := make(map[string]youtrack.EnumBundleElement, len(enumBundle.Values))
		for _, v := range enumBundle.Values {
			byName[v.Name] = v
		}

		for _, name := range trimmed {
			v, exists := byName[name]
			if !exists {
				return nil, fmt.Errorf("default value %q not found in enum bundle %q", name, enumBundle.Name)
			}
			refs = append(refs, youtrack.ProjectCustomFieldValueRef{ID: v.ID, Name: v.Name, Type: v.Type})
		}
	case "StateProjectCustomField":
		stateBundle, err := r.client.GetStateBundleByID(ctx, bundleID)
		if err != nil {
			return nil, fmt.Errorf("could not load state bundle %q for default_value_names: %w", bundleID, err)
		}

		byName := make(map[string]youtrack.StateBundleElement, len(stateBundle.Values))
		for _, v := range stateBundle.Values {
			byName[v.Name] = v
		}

		for _, name := range trimmed {
			v, exists := byName[name]
			if !exists {
				return nil, fmt.Errorf("default value %q not found in state bundle %q", name, stateBundle.Name)
			}
			refs = append(refs, youtrack.ProjectCustomFieldValueRef{ID: v.ID, Name: v.Name, Type: v.Type})
		}
	default:
		return nil, fmt.Errorf(errDefaultValuesTypeNotSupported, fieldType)
	}

	return refs, nil
}

func (r *projectCustomFieldResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, projectCustomFieldImportSeparator, projectCustomFieldImportParts)
	if len(parts) != projectCustomFieldImportParts || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			errMissingProjectCustomFieldID,
			errProjectCustomFieldImportIDInvalid,
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (m *projectCustomFieldResourceModel) toUpsertPayload(
	fieldID string,
	bundle *youtrack.BundleRef,
	defaultValues []youtrack.ProjectCustomFieldValueRef,
) youtrack.ProjectCustomFieldUpsertPayload {
	payload := youtrack.ProjectCustomFieldUpsertPayload{
		Field:         &youtrack.CustomFieldIDRef{ID: fieldID},
		Bundle:        bundle,
		DefaultValues: defaultValues,
		Type:          m.FieldType.ValueString(),
	}

	if !m.CanBeEmpty.IsNull() && !m.CanBeEmpty.IsUnknown() {
		v := m.CanBeEmpty.ValueBool()
		payload.CanBeEmpty = &v
	}

	if !m.EmptyFieldText.IsNull() && !m.EmptyFieldText.IsUnknown() {
		payload.EmptyFieldText = m.EmptyFieldText.ValueString()
	}

	if !m.IsPublic.IsNull() && !m.IsPublic.IsUnknown() {
		v := m.IsPublic.ValueBool()
		payload.IsPublic = &v
	}

	return payload
}

func (m *projectCustomFieldResourceModel) fromAPIModel(f *youtrack.ProjectCustomField) {
	m.ID = types.StringValue(f.ID)
	m.CanBeEmpty = types.BoolValue(f.CanBeEmpty)
	m.EmptyFieldText = helpers.StringOrNull(f.EmptyFieldText)
	m.IsPublic = types.BoolValue(f.IsPublic)
	m.FieldType = types.StringValue(f.Type)

	if f.Field != nil {
		m.FieldName = types.StringValue(f.Field.Name)
	}

	if f.Bundle != nil {
		m.BundleName = types.StringValue(f.Bundle.Name)
	} else {
		m.BundleName = types.StringNull()
	}

	if len(f.DefaultValues) == 0 {
		m.DefaultValueNames = types.ListNull(types.StringType)
		return
	}

	names := make([]string, 0, len(f.DefaultValues))
	for _, value := range f.DefaultValues {
		if strings.TrimSpace(value.Name) == "" {
			continue
		}
		names = append(names, value.Name)
	}

	if len(names) == 0 {
		m.DefaultValueNames = types.ListNull(types.StringType)
		return
	}

	list, diags := types.ListValueFrom(context.Background(), types.StringType, names)
	if diags.HasError() {
		m.DefaultValueNames = types.ListNull(types.StringType)
		return
	}

	m.DefaultValueNames = list
}
