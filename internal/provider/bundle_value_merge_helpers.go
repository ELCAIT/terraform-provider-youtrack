package provider

type bundleValueMergeOps[Model any, API any] struct {
	toAPI    func(Model) API
	modelID  func(Model) string
	apiID    func(API) string
	setAPIID func(*API, string)
	apiName  func(API) string
}

type plannedBundleValues[API any] struct {
	byID             map[string]API
	withoutIDByName  map[string]API
	withoutIDInOrder []API
}

func mergeBundleValuesPreservingExisting[Model any, API any](modelValues []Model, currentValues []API, ops bundleValueMergeOps[Model, API]) []API {
	planned := partitionPlannedBundleValues(modelValues, ops)
	values := mergeCurrentBundleValues(currentValues, planned, ops)
	appendRemainingPlannedByID(modelValues, planned.byID, &values, ops)
	appendRemainingPlannedWithoutID(planned.withoutIDInOrder, planned.withoutIDByName, &values, ops)
	return values
}

func partitionPlannedBundleValues[Model any, API any](modelValues []Model, ops bundleValueMergeOps[Model, API]) plannedBundleValues[API] {
	planned := plannedBundleValues[API]{
		byID:             make(map[string]API, len(modelValues)),
		withoutIDByName:  make(map[string]API, len(modelValues)),
		withoutIDInOrder: make([]API, 0, len(modelValues)),
	}

	for _, value := range modelValues {
		item := ops.toAPI(value)
		itemID := ops.apiID(item)
		if itemID == "" {
			planned.withoutIDByName[normalizeBundleValueName(ops.apiName(item))] = item
			planned.withoutIDInOrder = append(planned.withoutIDInOrder, item)
			continue
		}
		planned.byID[itemID] = item
	}

	return planned
}

func mergeCurrentBundleValues[Model any, API any](currentValues []API, planned plannedBundleValues[API], ops bundleValueMergeOps[Model, API]) []API {
	values := make([]API, 0, len(currentValues)+len(planned.withoutIDInOrder))

	for _, existing := range currentValues {
		existingID := ops.apiID(existing)
		if plannedValue, ok := planned.byID[existingID]; ok {
			values = append(values, plannedValue)
			delete(planned.byID, existingID)
			continue
		}

		normalizedExistingName := normalizeBundleValueName(ops.apiName(existing))
		if plannedValue, ok := planned.withoutIDByName[normalizedExistingName]; ok {
			if existingID != "" {
				ops.setAPIID(&plannedValue, existingID)
			}
			values = append(values, plannedValue)
			delete(planned.withoutIDByName, normalizedExistingName)
			continue
		}

		values = append(values, existing)
	}

	return values
}

func appendRemainingPlannedByID[Model any, API any](modelValues []Model, plannedByID map[string]API, values *[]API, ops bundleValueMergeOps[Model, API]) {
	for _, value := range modelValues {
		plannedID := ops.modelID(value)
		if plannedID == "" {
			continue
		}
		plannedValue, ok := plannedByID[plannedID]
		if !ok {
			continue
		}
		*values = append(*values, plannedValue)
	}
}

func appendRemainingPlannedWithoutID[Model any, API any](plannedWithoutID []API, plannedWithoutIDByName map[string]API, values *[]API, ops bundleValueMergeOps[Model, API]) {
	for _, plannedValue := range plannedWithoutID {
		normalizedName := normalizeBundleValueName(ops.apiName(plannedValue))
		if _, ok := plannedWithoutIDByName[normalizedName]; !ok {
			continue
		}
		*values = append(*values, plannedValue)
		delete(plannedWithoutIDByName, normalizedName)
	}
}
