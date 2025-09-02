// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/validation/field"

	memoryonev1alpha1 "github.com/gardener/gardener-extension-os-gardenlinux/pkg/apis/memoryonegardenlinux/v1alpha1"
	"github.com/gardener/gardener-extension-os-gardenlinux/pkg/apis/memoryonegardenlinux/validation"
)

func TestOperatingSystemConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller OperatingSystemConfig Suite")
}

var _ = Describe("Operatingsystemconfiguration", func() {
	var (
		osc *memoryonev1alpha1.OperatingSystemConfiguration

		fldPath = field.NewPath("")
	)

	BeforeEach(func() {
		osc = &memoryonev1alpha1.OperatingSystemConfiguration{}
	})

	It("should reject vSMP configuration keys containing forbidden characters", func() {
		osc.VsmpConfiguration = map[string]string{
			"abc;def": "xyz",
		}

		allErrs := validation.ValidateOperatingSystemConfig(osc, fldPath)
		Expect(allErrs).To(HaveLen(1))
		Expect(allErrs.ToAggregate().Error()).To(ContainSubstring("Invalid value"))
	})

	It("should reject vSMP configuration values containing semicola", func() {
		osc.VsmpConfiguration = map[string]string{
			"abc": "xyz;123",
		}

		allErrs := validation.ValidateOperatingSystemConfig(osc, fldPath)
		Expect(allErrs).To(HaveLen(1))
		Expect(allErrs.ToAggregate().Error()).To(ContainSubstring("values must not contain semicola"))
	})

	It("should accept valid vSMP configuration", func() {
		osc.VsmpConfiguration = map[string]string{
			"abc": "xyz",
			"foo": "bar",
		}

		allErrs := validation.ValidateOperatingSystemConfig(osc, fldPath)
		Expect(allErrs).To(BeEmpty())
	})
})
