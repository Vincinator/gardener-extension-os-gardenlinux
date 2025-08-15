// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"regexp"
	"strings"

	memoryonev1alpha1 "github.com/gardener/gardener-extension-os-gardenlinux/pkg/apis/memoryonegardenlinux/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var (
	supportedInjectedFeatures = []string{"opt_enable", "feature_enable", "debug_features_enable"}
)

func ValidateOperatingSystemConfig(osconfig *memoryonev1alpha1.OperatingSystemConfiguration, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if osconfig.MemoryTopology != nil && len(*osconfig.MemoryTopology) > 0 {
		allErrs = append(allErrs, validateMemoryTopology(*osconfig.MemoryTopology, fldPath.Child("memoryTopology"))...)
	}

	if osconfig.SystemMemory != nil && len(*osconfig.SystemMemory) > 0 {
		if err := validateSystemMemory(*osconfig.SystemMemory, fldPath.Child("systemMemory")); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	return allErrs
}

func validateMemoryTopology(memoryTopology string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	elements := strings.Split(memoryTopology, ";")

	for i, v := range elements {
		featureKV := strings.Split(v, "=")
		if len(featureKV) > 1 {
			allErrs = append(allErrs, validateInjectedFeatures(featureKV, fldPath.Index(i))...)
		} else {
			regex := regexp.MustCompile("^\\d+x?$")
			if !regex.MatchString(featureKV[0]) {
				allErrs = append(allErrs, field.Invalid(fldPath.Index(i), elements[0], "memoryTopology must only contain digits with an optional trailing x"))
			}
		}
	}

	return allErrs
}

func validateSystemMemory(systemMemory string, fldPath *field.Path) *field.Error {
	if _, err := resource.ParseQuantity(systemMemory); err != nil {
		return field.Invalid(fldPath, systemMemory, "is not a valid quantity")
	}
	return nil
}

func validateInjectedFeatures(featureKV []string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// we already ensured in the calling function that featureKV is at least two elements
	switch featureKV[0] {
	case "opt_enable":
		regex := regexp.MustCompile("^&?0x[[:xdigit:]]{8}$")
		if !regex.MatchString(featureKV[1]) {
			allErrs = append(allErrs, field.Invalid(fldPath, featureKV, "opt_enable expects an 8 digit hex-string"))
		}

	case "debug_features_enable":
		regex := regexp.MustCompile("^&?0x[[:xdigit:]]{16}$")
		if !regex.MatchString(featureKV[1]) {
			allErrs = append(allErrs, field.Invalid(fldPath, featureKV, "debug_features_enable expects a 16 digit hex-string"))
		}

	default:
		allErrs = append(allErrs, field.NotSupported(fldPath, featureKV[0], supportedInjectedFeatures))
	}

	return allErrs
}
