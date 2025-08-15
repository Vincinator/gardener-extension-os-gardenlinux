// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"fmt"
	"testing"

	memoryonev1alpha1 "github.com/gardener/gardener-extension-os-gardenlinux/pkg/apis/memoryonegardenlinux/v1alpha1"
	"github.com/gardener/gardener-extension-os-gardenlinux/pkg/apis/memoryonegardenlinux/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	DescribeTable("validation of values for memoryTopology", func(value string, expectedErrorCount int) {
		osc.MemoryTopology = &value
		allErrs := validation.ValidateOperatingSystemConfig(osc, fldPath)

		Expect(allErrs).To(HaveLen(expectedErrorCount))

	},
		Entry("should accept 123", "123", 0),
		Entry("should accept 123x", "123x", 0),
		Entry("should deny 123a", "123a", 1),
		Entry("should deny x321", "x321", 1),
		Entry("should deny abc", "abc", 1),
	)

	DescribeTable("validation of values for systemMemory", func(value string, expectedErrorCount int) {
		osc.SystemMemory = &value
		allErrs := validation.ValidateOperatingSystemConfig(osc, fldPath)

		if expectedErrorCount == 0 {
			Expect(allErrs).To(BeEmpty())
		} else {
			Expect(allErrs).To(HaveLen(expectedErrorCount))
		}
	},
		Entry("should accept 123", "123", 0),
		Entry("should deny 123x", "123x", 1),
		Entry("should accept 123m", "123m", 0),
		Entry("should accept 123G", "123G", 0),
		Entry("should deny 123 m", "123 m", 1),
		Entry("should deny 123 G", "123 G", 1),
		Entry("should deny x321", "x321", 1),
		Entry("should deny abc", "abc", 1),
	)

	DescribeTable("validation of values for injected values into memoryTopology", func(value string, expectedErrorCount int) {
		osc.MemoryTopology = ptr.To(fmt.Sprintf("2x;%s", value))
		allErrs := validation.ValidateOperatingSystemConfig(osc, fldPath)

		if expectedErrorCount == 0 {
			Expect(allErrs).To(BeEmpty())
		} else {
			Expect(allErrs).To(HaveLen(expectedErrorCount))
		}
	},
		Entry("should accept debug_features_enable with valid value", "debug_features_enable=&0xffffffffffffffff", 0),
		Entry("should deny debug_features_enable with invalid value", "debug_features_enable=&0xfffffffffffffff", 1),
		Entry("should accept opt_enable with valid value", "opt_enable=&0xffffffff", 0),
		Entry("should deny opt_enable with valid value", "opt_enable=&0xfffffff", 1),
		Entry("should deny an arbitrary value", "abcd=xyz", 1),
		Entry("should deny a key without a value", "opt_enable", 1),
	)
})
