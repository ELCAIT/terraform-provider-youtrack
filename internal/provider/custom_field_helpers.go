package provider

import (
	"context"
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

	return defaults, hasValue
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
		"can_be_empty":     types.BoolValue(apiModel.FieldDefaults.CanBeEmpty),
		"empty_field_text": helpers.StringOrNull(apiModel.FieldDefaults.EmptyFieldText),
		"is_public":        types.BoolValue(apiModel.FieldDefaults.IsPublic),
		"bundle_id":        bundleID,
		"bundle_name":      bundleName,
	})
}

func customFieldDefaultsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"can_be_empty":     types.BoolType,
		"empty_field_text": types.StringType,
		"is_public":        types.BoolType,
		"bundle_id":        types.StringType,
		"bundle_name":      types.StringType,
	}
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
	model.FieldDefaults = &defaults

	return model
}
