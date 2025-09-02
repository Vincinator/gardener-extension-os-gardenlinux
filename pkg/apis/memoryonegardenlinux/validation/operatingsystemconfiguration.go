// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	memoryonev1alpha1 "github.com/gardener/gardener-extension-os-gardenlinux/pkg/apis/memoryonegardenlinux/v1alpha1"
)

var (
	supportedInjectedFeatures = []string{"opt_enable", "feature_enable", "debug_features_enable"}
)

func ValidateOperatingSystemConfig(osconfig *memoryonev1alpha1.OperatingSystemConfiguration, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if osconfig.VsmpConfiguration != nil {
		allErrs = append(allErrs, validateVsmpConfig(osconfig.VsmpConfiguration, fldPath.Child("vsmpConfiguration"))...)
	}

	return allErrs
}

func validateVsmpConfig(config map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for k, v := range config {
		if validationerrors := validation.IsQualifiedName(k); len(validationerrors) != 0 {
			for _, e := range validationerrors {
				allErrs = append(allErrs, field.Invalid(fldPath.Key(k), k, e))
			}
		}
		if strings.Contains(v, ";") {
			allErrs = append(allErrs, field.Forbidden(fldPath.Key(k), "vSMP configuration values must not contain semicola"))
		}
	}

	return allErrs
}
