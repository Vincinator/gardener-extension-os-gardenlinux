// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig_test

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"strings"

	"github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/test"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	runtimeutils "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	memoryonev1alpha1 "github.com/gardener/gardener-extension-os-gardenlinux/pkg/apis/memoryonegardenlinux/v1alpha1"
	. "github.com/gardener/gardener-extension-os-gardenlinux/pkg/controller/operatingsystemconfig"
	"github.com/gardener/gardener-extension-os-gardenlinux/pkg/memoryone"
	"github.com/gardener/gardener-extension-os-gardenlinux/pkg/gardenlinux"
)

var codec runtime.Codec

func init() {
	scheme := runtime.NewScheme()
	runtimeutils.Must(memoryonev1alpha1.AddToScheme(scheme))
	codec = serializer.NewCodecFactory(scheme, serializer.EnableStrict).LegacyCodec(memoryonev1alpha1.SchemeGroupVersion)
}

var _ = Describe("Actuator", func() {
	var (
		ctx        = context.TODO()
		log        = logr.Discard()
		fakeClient client.Client
		mgr        manager.Manager

		osc      *extensionsv1alpha1.OperatingSystemConfig
		actuator operatingsystemconfig.Actuator
	)

	BeforeEach(func() {
		fakeClient = fakeclient.NewClientBuilder().Build()
		mgr = test.FakeManager{Client: fakeClient}
		actuator = NewActuator(mgr)

		osc = &extensionsv1alpha1.OperatingSystemConfig{
			Spec: extensionsv1alpha1.OperatingSystemConfigSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type: gardenlinux.OSTypeGardenLinux,
				},
				Purpose: extensionsv1alpha1.OperatingSystemConfigPurposeProvision,
				Units:   []extensionsv1alpha1.Unit{{Name: "some-unit", Content: ptr.To("foo")}},
				Files:   []extensionsv1alpha1.File{{Path: "/some/file", Content: extensionsv1alpha1.FileContent{Inline: &extensionsv1alpha1.FileContentInline{Data: "bar"}}}},
			},
		}
	})

	When("purpose is 'provision'", func() {

		When("OS type is 'suse-chost'", func() {
			Describe("#Reconcile", func() {
				It("should not return an error", func() {
					_ , extensionUnits, extensionFiles, inplaceUpdateStatus, err := actuator.Reconcile(ctx, log, osc)
					Expect(err).NotTo(HaveOccurred())

					Expect(extensionUnits).To(BeEmpty())
					Expect(extensionFiles).To(BeEmpty())
					Expect(inplaceUpdateStatus).To(BeNil())
				})
			})
		})

		When("OS type is 'memoryone-chost'", func() {
			var (
				memoryOneConfiguration memoryonev1alpha1.OperatingSystemConfiguration
			)

			BeforeEach(func() {
				memoryOneConfiguration = memoryonev1alpha1.OperatingSystemConfiguration{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "memoryone-gartdenlinux.os.extensions.gardener.cloud/v1alpha1",
						Kind:       "OperatingSystemConfiguration",
					},
				}

				osc.Spec.Type = memoryone.OSTypeMemoryOneGardenLinux
			})

			When("Legacy fields are used", func() {
				It("should use default values for the system_memory and mem_topology", func() {
					osc.Spec.ProviderConfig = nil

					userData, extensionUnits, extensionFiles, inplaceUpdateStatus, err := actuator.Reconcile(ctx, log, osc)
					Expect(err).NotTo(HaveOccurred())

					vSmpConfig, _:= decodeVsmpUserData(string(userData))

					Expect(vSmpConfig).To(BeEquivalentTo(map[string]string{
						"mem_topology":  "2",
						"system_memory": "6x",
					}))
					Expect(extensionUnits).To(BeEmpty())
					Expect(extensionFiles).To(BeEmpty())
					Expect(inplaceUpdateStatus).To(BeNil())
				})

				It("should use custom values for system_memory and mem_topology", func() {
					memoryOneConfiguration.MemoryTopology = ptr.To("4")
					memoryOneConfiguration.SystemMemory = ptr.To("8x")
					Expect(encodeMemoryOneConfigurationIntoOsc(codec, osc, &memoryOneConfiguration)).To(Succeed())

					userData, extensionUnits, extensionFiles, inplaceUpdateStatus, err := actuator.Reconcile(ctx, log, osc)
					Expect(err).NotTo(HaveOccurred())

					vSmpConfig, _:= decodeVsmpUserData(string(userData))

					Expect(vSmpConfig).To(BeEquivalentTo(map[string]string{
						"mem_topology":  "4",
						"system_memory": "8x",
					}))

					Expect(extensionUnits).To(BeEmpty())
					Expect(extensionFiles).To(BeEmpty())
					Expect(inplaceUpdateStatus).To(BeNil())
				})

				It("should allow injecting additional key-value pairs by semicola", func() {
					memoryOneConfiguration.MemoryTopology = ptr.To("4; foo=bar")
					memoryOneConfiguration.SystemMemory = ptr.To("8x")
					Expect(encodeMemoryOneConfigurationIntoOsc(codec, osc, &memoryOneConfiguration)).To(Succeed())

					userData, extensionUnits, extensionFiles, inplaceUpdateStatus, err := actuator.Reconcile(ctx, log, osc)
					Expect(err).NotTo(HaveOccurred())

					vSmpConfig,  _:= decodeVsmpUserData(string(userData))

					Expect(vSmpConfig).To(BeEquivalentTo(map[string]string{
						"mem_topology":  "4; foo=bar",
						"system_memory": "8x",
					}))

					Expect(extensionUnits).To(BeEmpty())
					Expect(extensionFiles).To(BeEmpty())
					Expect(inplaceUpdateStatus).To(BeNil())
				})
			})

			When("MemoryOne configuration map is used", func() {
				It("Should include arbitrary configuration values in vSMP config", func() {
					memoryOneConfiguration.VsmpConfiguration = map[string]string{
						"foo": "bar",
						"abc": "xyz",
					}

					Expect(encodeMemoryOneConfigurationIntoOsc(codec, osc, &memoryOneConfiguration)).To(Succeed())

					userData, extensionUnits, extensionFiles, inplaceUpdateStatus, err := actuator.Reconcile(ctx, log, osc)
					Expect(err).NotTo(HaveOccurred())

					vSmpConfig, _:= decodeVsmpUserData(string(userData))

					Expect(vSmpConfig).To(BeEquivalentTo(map[string]string{
						"mem_topology":  "2",
						"system_memory": "6x",
						"foo":           "bar",
						"abc":           "xyz",
					}))

					Expect(extensionUnits).To(BeEmpty())
					Expect(extensionFiles).To(BeEmpty())
					Expect(inplaceUpdateStatus).To(BeNil())
				})

				It("Should not allow injecting additional key-value pairs by semicola", func() {
					memoryOneConfiguration.VsmpConfiguration = map[string]string{
						"foo": "bar; foobar",
						"abc": "xyz",
					}

					Expect(encodeMemoryOneConfigurationIntoOsc(codec, osc, &memoryOneConfiguration)).To(Succeed())

					userData, extensionUnits, extensionFiles, inplaceUpdateStatus, err := actuator.Reconcile(ctx, log, osc)
					Expect(err).NotTo(HaveOccurred())

					vSmpConfig,  _ := decodeVsmpUserData(string(userData))

					Expect(vSmpConfig).To(BeEquivalentTo(map[string]string{
						"mem_topology":  "2",
						"system_memory": "6x",
						"foo":           "bar",
						"abc":           "xyz",
					}))

					Expect(extensionUnits).To(BeEmpty())
					Expect(extensionFiles).To(BeEmpty())
					Expect(inplaceUpdateStatus).To(BeNil())
				})

				It("Should give priority to legacy values", func() {
					memoryOneConfiguration.MemoryTopology = ptr.To("3")
					memoryOneConfiguration.SystemMemory = ptr.To("7x")
					memoryOneConfiguration.VsmpConfiguration = map[string]string{
						"mem_topology":  "5",
						"system_memory": "13x",
					}

					Expect(encodeMemoryOneConfigurationIntoOsc(codec, osc, &memoryOneConfiguration)).To(Succeed())

					userData, extensionUnits, extensionFiles, inplaceUpdateStatus, err := actuator.Reconcile(ctx, log, osc)
					Expect(err).NotTo(HaveOccurred())

					vSmpConfig,  _ := decodeVsmpUserData(string(userData))

					Expect(vSmpConfig).To(BeEquivalentTo(map[string]string{
						"mem_topology":  "3",
						"system_memory": "7x",
					}))

					Expect(extensionUnits).To(BeEmpty())
					Expect(extensionFiles).To(BeEmpty())
					Expect(inplaceUpdateStatus).To(BeNil())
				})
			})
		})
	})

	When("purpose is 'reconcile'", func() {
		BeforeEach(func() {
			osc.Spec.Purpose = extensionsv1alpha1.OperatingSystemConfigPurposeReconcile
		})

		Describe("#Reconcile", func() {
			It("should not return an error", func() {
				userData, extensionUnits, _, _, err := actuator.Reconcile(ctx, log, osc)
				Expect(err).NotTo(HaveOccurred())

				Expect(userData).To(BeEmpty())
				Expect(extensionUnits).To(BeEmpty())
			})

			It("should deploy a sysctl file to configure IPv6 router advertisements", func() {
				_, _, extensionFiles, _, err := actuator.Reconcile(ctx, log, osc)
				Expect(err).NotTo(HaveOccurred())

				sysctl_content := `# enables IPv6 router advertisements on all interfaces even when ip forwarding for IPv6 is enabled
net.ipv6.conf.all.accept_ra = 2

# specifically enable IPv6 router advertisements on the first ethernet interface (eth0 for net.ifnames=0)
net.ipv6.conf.eth0.accept_ra = 2
`

				Expect(extensionFiles).To(HaveLen(1))
				Expect(extensionFiles[0].Path).To(Equal("/etc/sysctl.d/98-enable-ipv6-ra.conf"))
				Expect(extensionFiles[0].Permissions).To(Equal(ptr.To(uint32(0644))))
				Expect(extensionFiles[0].Content.Inline.Data).To(Equal(sysctl_content))
			})
		})
	})
})

type multiPart struct {
	contentType string
	params      map[string]string
	content     string
}

func readMimeMultiParts(s string) []multiPart {
	GinkgoHelper()
	const (
		contentTypeIdentifier = "Content-Type"
		boundary              = "==BOUNDARY=="
	)

	var parts []multiPart

	mr := multipart.NewReader(strings.NewReader(s), boundary)
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		Expect(err).ShouldNot(HaveOccurred())

		c, err := io.ReadAll(p)
		Expect(err).ShouldNot(HaveOccurred())

		mediaType, params, err := mime.ParseMediaType(p.Header.Get(contentTypeIdentifier))
		Expect(err).ShouldNot(HaveOccurred())

		parts = append(parts, multiPart{
			contentType: mediaType,
			params:      params,
			content:     string(c),
		})
	}
	return parts
}

func extractVsmpConfiguration(p multiPart) map[string]string {
	GinkgoHelper()
	Expect(p.contentType).To(Equal("text/x-vsmp"))
	Expect(p.params).To(HaveLen(1))
	Expect(p.params).To(HaveKeyWithValue("section", "vsmp"))

	lines := strings.Split(p.content, "\n")

	var config = make(map[string]string, len(lines))
	for _, v := range lines {
		key, value, found := strings.Cut(v, "=")
		Expect(found).To(BeTrue())
		config[key] = value
	}

	return config
}

func extractUserdata(p multiPart) string {
	GinkgoHelper()
	Expect(p.contentType).To(Equal("text/x-shellscript"))
	Expect(p.params).To(BeEmpty())
	return p.content
}

func decodeVsmpUserData(s string) (map[string]string, string) {
	GinkgoHelper()
	prefix := `Content-Type: multipart/mixed; boundary="==BOUNDARY=="
MIME-Version: 1.0`

	Expect(strings.HasPrefix(s, prefix)).To(BeTrue())
	parts := readMimeMultiParts(s)
	Expect(parts).To(HaveLen(2))
	vsmpConfig := extractVsmpConfiguration(parts[0])
	userData := extractUserdata(parts[1])
	return vsmpConfig, userData
}

func encodeMemoryOneConfigurationIntoOsc(codec runtime.Codec, osc *extensionsv1alpha1.OperatingSystemConfig, moc *memoryonev1alpha1.OperatingSystemConfiguration) error {
	encoded, err := runtime.Encode(codec, moc)
	if err != nil {
		return err
	}
	osc.Spec.ProviderConfig = &runtime.RawExtension{
		Raw: encoded,
	}
	return nil
}
