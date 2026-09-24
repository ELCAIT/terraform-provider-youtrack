package provider

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// errKeptBundleValuesFmt reports values YouTrack still lists after an update
// removed them.
const errKeptBundleValuesFmt = "YouTrack kept values that are not present in configuration: %s. This usually happens when values are still required by workflows or existing issues. Keep these values in configuration or set archived = true instead of removing them."

// bundleValueReconciler edits the values of an existing bundle one by one.
//
// A bundle-level update cannot do it: YouTrack ignores changes to existing
// values sent in its values list, and deletes and recreates, under a new ID,
// any value sent there without its ID, which clears that value on every issue
// that used it. Model is a planned value, API a value as YouTrack returns it
// and Update the full replacement sent for an existing value.
type bundleValueReconciler[Model any, API any, Update any] struct {
	plannedName func(Model) string
	apiID       func(API) string
	apiName     func(API) string
	// toUpdate builds the replacement for an existing value from the plan.
	toUpdate func(value Model, current API, ordinal int) Update
	// currentAsUpdate expresses an existing value as the replacement that would
	// leave it unchanged, so that a value only gets written when it differs.
	currentAsUpdate func(current API) Update
	add             func(ctx context.Context, value Model, ordinal int) error
	replace         func(ctx context.Context, valueID string, update Update) error
	remove          func(ctx context.Context, valueID string) error
}

// reconcile deletes the existing values nothing in planned claims, replaces
// the ones that changed and adds the new ones. Each planned value is matched to
// an existing one by name, and its ordinal is set to its planned position so
// that YouTrack orders the values as configured.
func (r bundleValueReconciler[Model, API, Update]) reconcile(ctx context.Context, planned []Model, existing []API) error {
	matched, unmatched := matchBundleValues(planned, existing, r.plannedName, r.apiName)

	for _, stale := range unmatched {
		if err := r.remove(ctx, r.apiID(stale)); err != nil {
			return bundleValueRemovalError(r.apiName(stale), err)
		}
	}

	for i, value := range planned {
		if err := r.apply(ctx, value, matched[i], i+1); err != nil {
			return err
		}
	}

	return nil
}

func (r bundleValueReconciler[Model, API, Update]) apply(ctx context.Context, value Model, current *API, ordinal int) error {
	name := r.plannedName(value)
	if current == nil {
		if err := r.add(ctx, value, ordinal); err != nil {
			return fmt.Errorf("could not add value %q: %w", name, err)
		}
		return nil
	}

	update := r.toUpdate(value, *current, ordinal)
	if reflect.DeepEqual(update, r.currentAsUpdate(*current)) {
		return nil
	}
	if err := r.replace(ctx, r.apiID(*current), update); err != nil {
		return fmt.Errorf("could not update value %q: %w", name, err)
	}

	return nil
}

func bundleValueRemovalError(name string, err error) error {
	if isRequiredCustomFieldWorkflowError(err) {
		return fmt.Errorf(
			"could not remove value %q because a workflow still requires it: %w. Keep the value in configuration or set archived = true instead of removing it",
			name, err,
		)
	}
	return fmt.Errorf("could not remove value %q: %w", name, err)
}

// matchBundleValues pairs each planned value with the existing value of the
// same name, ignoring case and surrounding space. It returns the match per
// planned index (nil when the value is new) and the existing values nothing in
// the plan claimed.
func matchBundleValues[Model any, API any](
	planned []Model,
	existing []API,
	plannedName func(Model) string,
	apiName func(API) string,
) ([]*API, []API) {
	byName := make(map[string]int, len(existing))
	for i := range existing {
		key := normalizeBundleValueName(apiName(existing[i]))
		if _, taken := byName[key]; !taken {
			byName[key] = i
		}
	}

	matched := make([]*API, len(planned))
	claimed := make(map[int]struct{}, len(planned))
	for i, value := range planned {
		key := normalizeBundleValueName(plannedName(value))
		if index, ok := byName[key]; ok {
			matched[i] = &existing[index]
			claimed[index] = struct{}{}
			delete(byName, key)
		}
	}

	unmatched := make([]API, 0, len(existing))
	for i, value := range existing {
		if _, ok := claimed[i]; !ok {
			unmatched = append(unmatched, value)
		}
	}

	return matched, unmatched
}

// sortedByOrdinal returns values ordered by their ordinal. YouTrack lists
// bundle values in insertion order, which stops matching the configured order
// once values have been moved.
func sortedByOrdinal[T any](values []T, ordinal func(T) int) []T {
	sorted := append([]T(nil), values...)
	sort.SliceStable(sorted, func(i, j int) bool { return ordinal(sorted[i]) < ordinal(sorted[j]) })
	return sorted
}

// optionalStringPointer returns the configured value of an Optional+Computed
// string, or fallback when the plan leaves it unknown (unset in configuration),
// as a pointer that is nil for an empty value so that it is sent as null.
func optionalStringPointer(planned types.String, fallback string) *string {
	if planned.IsUnknown() {
		return stringPointerOrNil(fallback)
	}
	return stringPointerOrNil(planned.ValueString())
}

func stringPointerOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func mapBundleValues[In any, Out any](values []In, mapper func(In) Out) []Out {
	mapped := make([]Out, 0, len(values))
	for _, value := range values {
		mapped = append(mapped, mapper(value))
	}

	return mapped
}

func unexpectedBundleValueNames[Plan any, API any](planValues []Plan, updatedValues []API, planName func(Plan) string, apiName func(API) string) []string {
	plannedByName := make(map[string]struct{}, len(planValues))
	for _, value := range planValues {
		plannedByName[normalizeBundleValueName(planName(value))] = struct{}{}
	}

	unexpected := make([]string, 0)
	for _, value := range updatedValues {
		normalizedName := normalizeBundleValueName(apiName(value))
		if _, ok := plannedByName[normalizedName]; ok {
			continue
		}
		unexpected = append(unexpected, apiName(value))
	}

	sort.Strings(unexpected)
	return unexpected
}
