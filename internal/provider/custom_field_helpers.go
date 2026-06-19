package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func (m *customFieldResourceModel) toAPIModel() youtrack.CustomFieldUpsertRequest {
	apiModel := youtrack.CustomFieldUpsertRequest{
		Name:      m.Name.ValueString(),
		FieldType: &youtrack.FieldType{ID: m.FieldTypeID.ValueString()},
	}

	if value, ok := helpers.OptionalStringValue(m.LocalizedName); ok {
		apiModel.LocalizedName = &value
	}

	if value, ok := helpers.OptionalStringValue(m.Aliases); ok {
		apiModel.Aliases = &value
	}

	if value, ok := helpers.OptionalBoolValue(m.IsAutoAttached); ok {
		apiModel.IsAutoAttached = &value
	}

	if value, ok := helpers.OptionalBoolValue(m.IsDisplayedInIssueList); ok {
		apiModel.IsDisplayedInIssueList = &value
	}

	if defaults, ok := m.toAPIDefaults(); ok {
		apiModel.FieldDefaults = defaults
	}

	return apiModel
}

func (m *customFieldResourceModel) toAPIDefaults() (*youtrack.CustomFieldDefaultsUpsertModel, bool) {
	if m.FieldDefaults.IsNull() || m.FieldDefaults.IsUnknown() {
		return nil, false
	}

	defaultsModel := customFieldDefaultsModel{}
	if diags := m.FieldDefaults.As(context.Background(), &defaultsModel, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil, false
	}

	defaults := &youtrack.CustomFieldDefaultsUpsertModel{}
	hasValue := false
	if defaultsType, ok := defaultsTypeForFieldTypeID(m.FieldTypeID.ValueString()); ok {
		defaults.Type = defaultsType
	}

	if value, ok := helpers.OptionalBoolValue(defaultsModel.CanBeEmpty); ok {
		defaults.CanBeEmpty = &value
		hasValue = true
	}

	if value, ok := helpers.OptionalStringValue(defaultsModel.EmptyFieldText); ok {
		defaults.EmptyFieldText = &value
		hasValue = true
	}

	if value, ok := helpers.OptionalBoolValue(defaultsModel.IsPublic); ok {
		defaults.IsPublic = &value
		hasValue = true
	}

	if value, ok := helpers.OptionalStringValue(defaultsModel.BundleID); ok {
		bundle := &youtrack.BundleRef{ID: value}
		if bundleType, hasType := bundleTypeForFieldTypeID(m.FieldTypeID.ValueString()); hasType {
			bundle.Type = bundleType
		}

		defaults.Bundle = bundle
		hasValue = true
	}

	if defaultValues, ok := m.defaultValueRefsForDefaults(defaultsModel); ok {
		defaults.DefaultValues = defaultValues
		hasValue = true
	}

	return defaults, hasValue
}

func (m *customFieldResourceModel) defaultValueRefsForDefaults(defaultsModel customFieldDefaultsModel) ([]youtrack.ProjectCustomFieldValueRef, bool) {
	if defaultsModel.DefaultValueNames.IsNull() || defaultsModel.DefaultValueNames.IsUnknown() {
		return nil, false
	}

	names, ok := helpers.ListToStringSlice(context.Background(), defaultsModel.DefaultValueNames)
	if !ok {
		return nil, false
	}

	refs := make([]youtrack.ProjectCustomFieldValueRef, 0, len(names))
	for _, name := range names {
		normalized := strings.TrimSpace(name)
		if normalized == "" {
			continue
		}
		refs = append(refs, youtrack.ProjectCustomFieldValueRef{Name: normalized})
	}

	if len(refs) == 0 {
		return nil, false
	}

	return refs, true
}

func (m *customFieldResourceModel) fromAPIModel(apiModel *youtrack.CustomField) {
	m.ID = types.StringValue(apiModel.ID)
	m.Name = types.StringValue(apiModel.Name)
	m.LocalizedName = helpers.StringOrNull(apiModel.LocalizedName)
	m.Aliases = helpers.StringOrNull(apiModel.Aliases)
	m.FieldTypeID = types.StringValue(apiModel.FieldType.ID)
	m.FieldTypePresentation = helpers.StringOrNull(apiModel.FieldType.Presentation)
	m.IsAutoAttached = types.BoolValue(apiModel.IsAutoAttached)
	m.IsDisplayedInIssueList = types.BoolValue(apiModel.IsDisplayedInIssueList)
	m.Ordinal = types.Int64Value(int64(apiModel.Ordinal))
	m.IsUpdateable = types.BoolValue(apiModel.IsUpdateable)
	m.HasRunningJob = types.BoolValue(apiModel.HasRunningJob)

	if apiModel.FieldDefaults == nil {
		m.FieldDefaults = types.ObjectNull(customFieldDefaultsAttrTypes())
		return
	}

	bundleID := types.StringNull()
	bundleName := types.StringNull()
	if apiModel.FieldDefaults.Bundle != nil {
		bundleID = helpers.StringOrNull(apiModel.FieldDefaults.Bundle.ID)
		bundleName = helpers.StringOrNull(apiModel.FieldDefaults.Bundle.Name)
	}

	m.FieldDefaults = types.ObjectValueMust(customFieldDefaultsAttrTypes(), map[string]attr.Value{
		"can_be_empty":        types.BoolValue(apiModel.FieldDefaults.CanBeEmpty),
		"empty_field_text":    helpers.StringOrEmpty(apiModel.FieldDefaults.EmptyFieldText),
		"is_public":           types.BoolValue(apiModel.FieldDefaults.IsPublic),
		"bundle_id":           bundleID,
		"bundle_name":         bundleName,
		"default_value_names": defaultValueNamesFromCustomFieldDefaults(apiModel.FieldDefaults),
	})
}

func customFieldDefaultsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"can_be_empty":        types.BoolType,
		"empty_field_text":    types.StringType,
		"is_public":           types.BoolType,
		"bundle_id":           types.StringType,
		"bundle_name":         types.StringType,
		"default_value_names": types.ListType{ElemType: types.StringType},
	}
}

func defaultValueNamesFromCustomFieldDefaults(defaults *youtrack.CustomFieldDefaults) attr.Value {
	if defaults == nil || len(defaults.DefaultValues) == 0 {
		return types.ListNull(types.StringType)
	}

	names := make([]attr.Value, 0, len(defaults.DefaultValues))
	for _, value := range defaults.DefaultValues {
		if strings.TrimSpace(value.Name) == "" {
			continue
		}
		names = append(names, types.StringValue(value.Name))
	}

	if len(names) == 0 {
		return types.ListNull(types.StringType)
	}

	return types.ListValueMust(types.StringType, names)
}

func bundleTypeForFieldTypeID(fieldTypeID string) (string, bool) {
	fieldTypePrefix := extractFieldTypePrefix(fieldTypeID)

	switch fieldTypePrefix {
	case fieldTypePrefixEnum:
		return bundleTypeEnum, true
	case fieldTypePrefixState:
		return bundleTypeState, true
	case fieldTypePrefixOwnedField:
		return bundleTypeOwned, true
	case fieldTypePrefixBuild:
		return bundleTypeBuild, true
	case fieldTypePrefixVersion:
		return bundleTypeVer, true
	default:
		return "", false
	}
}

func defaultsTypeForFieldTypeID(fieldTypeID string) (string, bool) {
	fieldTypePrefix := extractFieldTypePrefix(fieldTypeID)

	switch fieldTypePrefix {
	case fieldTypePrefixEnum:
		return defaultsTypeEnum, true
	case fieldTypePrefixState:
		return defaultsTypeState, true
	case fieldTypePrefixOwnedField:
		return defaultsTypeOwned, true
	case fieldTypePrefixBuild:
		return defaultsTypeBuild, true
	case fieldTypePrefixVersion:
		return defaultsTypeVer, true
	default:
		return "", false
	}
}

func extractFieldTypePrefix(fieldTypeID string) string {
	fieldTypePrefix := fieldTypeID
	if idx := strings.Index(fieldTypeID, "["); idx > 0 {
		fieldTypePrefix = fieldTypeID[:idx]
	}

	return fieldTypePrefix
}

func (r *customFieldResource) resolveCustomFieldDefaultValues(
	ctx context.Context,
	apiModel *youtrack.CustomFieldUpsertRequest,
	plan customFieldResourceModel,
) error {
	if apiModel == nil || apiModel.FieldDefaults == nil {
		return nil
	}

	defaultsModel, err := decodeCustomFieldDefaultsModel(ctx, plan)
	if err != nil {
		return err
	}

	if err := r.ensureDefaultValuesBundle(ctx, apiModel, plan.FieldTypeID.ValueString(), defaultsModel); err != nil {
		return err
	}

	if len(apiModel.FieldDefaults.DefaultValues) == 0 {
		return nil
	}

	if apiModel.FieldDefaults.Bundle == nil || strings.TrimSpace(apiModel.FieldDefaults.Bundle.ID) == "" {
		return fmt.Errorf("default_value_names requires field_defaults.bundle_id or field_defaults.bundle_name to be set")
	}

	names := defaultValueNames(apiModel.FieldDefaults.DefaultValues)

	if len(names) == 0 {
		apiModel.FieldDefaults.DefaultValues = nil
		return nil
	}

	refs, err := resolveCustomFieldDefaultValuesByType(ctx, r.client, plan.FieldTypeID.ValueString(), apiModel.FieldDefaults.Bundle.ID, names)
	if err != nil {
		return err
	}

	apiModel.FieldDefaults.DefaultValues = refs
	return nil
}

func decodeCustomFieldDefaultsModel(ctx context.Context, plan customFieldResourceModel) (customFieldDefaultsModel, error) {
	defaultsModel := customFieldDefaultsModel{}
	if plan.FieldDefaults.IsNull() || plan.FieldDefaults.IsUnknown() {
		return defaultsModel, nil
	}

	if diags := plan.FieldDefaults.As(ctx, &defaultsModel, basetypes.ObjectAsOptions{}); diags.HasError() {
		return customFieldDefaultsModel{}, fmt.Errorf("failed to decode field_defaults")
	}

	return defaultsModel, nil
}

func (r *customFieldResource) ensureDefaultValuesBundle(
	ctx context.Context,
	apiModel *youtrack.CustomFieldUpsertRequest,
	fieldTypeID string,
	defaultsModel customFieldDefaultsModel,
) error {
	if apiModel.FieldDefaults.Bundle != nil && strings.TrimSpace(apiModel.FieldDefaults.Bundle.ID) != "" {
		return nil
	}

	if defaultsModel.BundleName.IsNull() || defaultsModel.BundleName.IsUnknown() {
		return nil
	}

	bundleName := strings.TrimSpace(defaultsModel.BundleName.ValueString())
	if bundleName == "" {
		return nil
	}

	bundle, err := r.lookupCustomFieldBundleByName(ctx, fieldTypeID, bundleName)
	if err != nil {
		return err
	}

	apiModel.FieldDefaults.Bundle = bundle
	return nil
}

func defaultValueNames(values []youtrack.ProjectCustomFieldValueRef) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}

	return names
}

func resolveCustomFieldDefaultValuesByType(
	ctx context.Context,
	client *youtrack.Client,
	fieldTypeID string,
	bundleID string,
	names []string,
) ([]youtrack.ProjectCustomFieldValueRef, error) {
	switch extractFieldTypePrefix(fieldTypeID) {
	case fieldTypePrefixEnum:
		return resolveEnumCustomFieldDefaultValues(ctx, client, bundleID, names)
	case fieldTypePrefixState:
		return resolveStateCustomFieldDefaultValues(ctx, client, bundleID, names)
	default:
		return nil, errors.New(errDefaultValuesTypeUnsupported)
	}
}

func (r *customFieldResource) lookupCustomFieldBundleByName(ctx context.Context, fieldTypeID, bundleName string) (*youtrack.BundleRef, error) {
	switch extractFieldTypePrefix(fieldTypeID) {
	case fieldTypePrefixEnum:
		bundle, err := r.client.GetEnumBundleByName(ctx, bundleName)
		if err != nil {
			return nil, fmt.Errorf("could not find enum bundle with name %q: %w", bundleName, err)
		}
		return &youtrack.BundleRef{ID: bundle.ID, Type: bundleTypeEnum}, nil
	case fieldTypePrefixState:
		bundle, err := r.client.GetStateBundleByName(ctx, bundleName)
		if err != nil {
			return nil, fmt.Errorf("could not find state bundle with name %q: %w", bundleName, err)
		}
		return &youtrack.BundleRef{ID: bundle.ID, Type: bundleTypeState}, nil
	default:
		return nil, errors.New(errBundleNameTypeUnsupported)
	}
}

func resolveEnumCustomFieldDefaultValues(
	ctx context.Context,
	client *youtrack.Client,
	bundleID string,
	names []string,
) ([]youtrack.ProjectCustomFieldValueRef, error) {
	bundle, err := client.GetEnumBundleByID(ctx, bundleID)
	if err != nil {
		return nil, fmt.Errorf("could not load enum bundle %q for default_value_names: %w", bundleID, err)
	}

	byName := make(map[string]youtrack.EnumBundleElement, len(bundle.Values))
	for _, value := range bundle.Values {
		byName[value.Name] = value
	}

	refs := make([]youtrack.ProjectCustomFieldValueRef, 0, len(names))
	for _, name := range names {
		value, exists := byName[name]
		if !exists {
			return nil, fmt.Errorf("default value %q not found in enum bundle %q", name, bundle.Name)
		}
		refs = append(refs, youtrack.ProjectCustomFieldValueRef{ID: value.ID, Name: value.Name, Type: value.Type})
	}

	return refs, nil
}

func resolveStateCustomFieldDefaultValues(
	ctx context.Context,
	client *youtrack.Client,
	bundleID string,
	names []string,
) ([]youtrack.ProjectCustomFieldValueRef, error) {
	bundle, err := client.GetStateBundleByID(ctx, bundleID)
	if err != nil {
		return nil, fmt.Errorf("could not load state bundle %q for default_value_names: %w", bundleID, err)
	}

	byName := make(map[string]youtrack.StateBundleElement, len(bundle.Values))
	for _, value := range bundle.Values {
		byName[value.Name] = value
	}

	refs := make([]youtrack.ProjectCustomFieldValueRef, 0, len(names))
	for _, name := range names {
		value, exists := byName[name]
		if !exists {
			return nil, fmt.Errorf("default value %q not found in state bundle %q", name, bundle.Name)
		}
		refs = append(refs, youtrack.ProjectCustomFieldValueRef{ID: value.ID, Name: value.Name, Type: value.Type})
	}

	return refs, nil
}

func (r *customFieldResource) createWithBundleFallback(
	ctx context.Context,
	apiModel youtrack.CustomFieldUpsertRequest,
) (*youtrack.CustomField, error) {
	initialModel := customFieldRequestWithoutBundle(apiModel)

	created, err := r.client.CreateCustomField(ctx, initialModel)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errBundleFallbackCreate, err)
	}

	if strings.TrimSpace(created.ID) == "" {
		return nil, fmt.Errorf("%s: custom field ID is empty", errBundleFallbackUpdate)
	}

	updated, err := r.client.UpdateCustomField(ctx, created.ID, apiModel)
	if err == nil {
		return updated, nil
	}

	deleteErr := r.client.DeleteCustomField(ctx, created.ID)
	if deleteErr != nil {
		return nil, fmt.Errorf("%s: %v; %s: %w", errBundleFallbackUpdate, err, errBundleFallbackDelete, deleteErr)
	}

	return nil, fmt.Errorf("%s: %w", errBundleFallbackUpdate, err)
}

func customFieldRequestHasBundle(model youtrack.CustomFieldUpsertRequest) bool {
	return model.FieldDefaults != nil && model.FieldDefaults.Bundle != nil && strings.TrimSpace(model.FieldDefaults.Bundle.ID) != ""
}

func customFieldRequestWithoutBundle(model youtrack.CustomFieldUpsertRequest) youtrack.CustomFieldUpsertRequest {
	if model.FieldDefaults == nil {
		return model
	}

	defaults := *model.FieldDefaults
	defaults.Bundle = nil
	defaults.DefaultValues = nil
	model.FieldDefaults = &defaults

	return model
}
