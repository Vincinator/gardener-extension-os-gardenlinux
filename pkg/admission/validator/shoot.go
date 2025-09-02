// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/util"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	gardencorehelper "github.com/gardener/gardener/pkg/apis/core/helper"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	memoryonev1alpha1 "github.com/gardener/gardener-extension-os-gardenlinux/pkg/apis/memoryonegardenlinux/v1alpha1"
	memoryonegardenlinuxValidation "github.com/gardener/gardener-extension-os-gardenlinux/pkg/apis/memoryonegardenlinux/validation"
	"github.com/gardener/gardener-extension-os-gardenlinux/pkg/memoryone"
)

// NewShootValidator returns a new instance of a shoot validator.
func NewShootValidator(mgr manager.Manager) extensionswebhook.Validator {
	return &shoot{
		client:         mgr.GetClient(),
		decoder:        serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
		lenientDecoder: serializer.NewCodecFactory(mgr.GetScheme()).UniversalDecoder(),
	}
}

type shoot struct {
	client         client.Client
	decoder        runtime.Decoder
	lenientDecoder runtime.Decoder
}

// Validate validates the given shoot object.
func (s *shoot) Validate(ctx context.Context, newObj, _ client.Object) error {
	shoot, ok := newObj.(*core.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", newObj)
	}

	// Skip if it's a workerless Shoot
	if gardencorehelper.IsWorkerless(shoot) {
		return nil
	}

	shootV1Beta1 := &gardencorev1beta1.Shoot{}
	err := gardencorev1beta1.Convert_core_Shoot_To_v1beta1_Shoot(shoot, shootV1Beta1, nil)
	if err != nil {
		return err
	}

	return s.validateShoot(ctx, shoot)
}

func (s *shoot) validateShoot(_ context.Context, shoot *core.Shoot) error {
	allErrs := field.ErrorList{}
	fldPath := field.NewPath("spec", "provider", "workers")

	for i, worker := range shoot.Spec.Provider.Workers {
		machineImage := worker.Machine.Image

		if machineImage == nil || !isSupportedMachineImage(machineImage.Name) {
			continue
		}

		if machineImage.ProviderConfig == nil {
			continue
		}

		var operatingSystemConfig *memoryonev1alpha1.OperatingSystemConfiguration

		fldPath = fldPath.Index(i).Child("providerConfig")
		if err := util.Decode(s.decoder, machineImage.ProviderConfig.Raw, operatingSystemConfig); err != nil {
			return field.Invalid(fldPath, string(machineImage.ProviderConfig.Raw), "is not a valid OperatingSystemConfiguration")
		}

		if errList := memoryonegardenlinuxValidation.ValidateOperatingSystemConfig(operatingSystemConfig, fldPath); len(errList) != 0 {
			allErrs = append(allErrs, errList...)
		}
	}

	if len(allErrs) > 1 {
		return allErrs.ToAggregate()
	}

	return nil
}

func isSupportedMachineImage(machineImageName string) bool {
	return machineImageName == memoryone.OSTypememoryonegardenlinux
}
