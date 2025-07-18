package builder

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	gomega_format "github.com/onsi/gomega/format"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/assisted-service/internal/common"
	"github.com/openshift/assisted-service/internal/installcfg"
	"github.com/openshift/assisted-service/internal/network"
	"github.com/openshift/assisted-service/internal/provider/registry"
	"github.com/openshift/assisted-service/models"
	"github.com/openshift/assisted-service/pkg/mirrorregistries"
)

var (
	mockMirrorRegistriesConfigBuilder *mirrorregistries.MockServiceMirrorRegistriesConfigBuilder
	providerRegistry                  registry.ProviderRegistry
	ctrl                              *gomock.Controller
)

func createInstallConfigBuilder() *installConfigBuilder {
	ctrl = gomock.NewController(GinkgoT())

	mockMirrorRegistriesConfigBuilder = mirrorregistries.NewMockServiceMirrorRegistriesConfigBuilder(ctrl)
	providerRegistry = registry.InitProviderRegistry(common.GetTestLog())
	return &installConfigBuilder{
		log:                     common.GetTestLog(),
		mirrorRegistriesBuilder: mockMirrorRegistriesConfigBuilder,
		providerRegistry:        providerRegistry}
}

var _ = Describe("installcfg", func() {

	gomega_format.CharactersAroundMismatchToInclude = 80

	const testBundle1 string = `-----BEGIN CERTIFICATE-----
MIIDozCCAougAwIBAgIULCOqWTF
aEA8gNEmV+rb7h1v0r3EwDQYJKoZIhvcNAQELBQAwYTELMAkGA1UEBhMCaXMxCzAJBgNVBAgMAmRk
2lyDI6UR3Fbz4pVVAxGXnVhBExjBE=
-----END CERTIFICATE-----`

	const testBundle2 string = `-----BEGIN CERTIFICATE-----
MIIDozCCAougAwIBAgIULCOqWTT
aEA8gNEmV+rb7h1v0r3EwDQYJKoZIhvcNAQELBQAwYTELMAkGA1UEBhMCaXMxCzAJBgNVBAgMAmRk
2lyDI6UR3Fbz4pVVAxGXnVhBExjBE=
-----END CERTIFICATE-----`

	const testBundle3 string = `-----BEGIN CERTIFICATE-----
MIIDozCCAougAwIBAgIULCOqWTG
aEA8gNEmV+rb7h1v0r3EwDQYJKoZIhvcNAQELBQAwYTELMAkGA1UEBhMCaXMxCzAJBgNVBAgMAmRk
2lyDI6UR3Fbz4pVVAxGXnVhBExjBE=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDozCCAougAwIBAgIULCOqWTJ
aEA8gNEmV+rb7h1v0r3EwDQYJKoZIhvcNAQELBQAwYTELMAkGA1UEBhMCaXMxCzAJBgNVBAgMAmRk
2lyDI6UR3Fbz4pVVAxGXnVhBExjBE=
-----END CERTIFICATE-----`

	const testBundle4 string = `-----BEGIN CERTIFICATE-----
MIIDozCCAougAwIBAgIULCOqWTQ
aEA8gNEmV+rb7h1v0r3EwDQYJKoZIhvcNAQELBQAwYTELMAkGA1UEBhMCaXMxCzAJBgNVBAgMAmRk
2lyDI6UR3Fbz4pVVAxGXnVhBExjBE=
-----END CERTIFICATE-----`

	var (
		host1            models.Host
		host2            models.Host
		host3            models.Host
		cluster          common.Cluster
		clusterInfraenvs []*common.InfraEnv
		installConfig    *installConfigBuilder
	)
	BeforeEach(func() {
		clusterId := strfmt.UUID(uuid.New().String())
		cluster = common.Cluster{Cluster: models.Cluster{
			ID:                     &clusterId,
			OpenshiftVersion:       common.TestDefaultConfig.OpenShiftVersion,
			Name:                   "test-cluster",
			BaseDNSDomain:          "redhat.com",
			ClusterNetworks:        []*models.ClusterNetwork{{Cidr: "1.1.1.0/24"}},
			ServiceNetworks:        []*models.ServiceNetwork{{Cidr: "2.2.2.0/24"}},
			MachineNetworks:        []*models.MachineNetwork{{Cidr: "1.2.3.0/24"}},
			APIVips:                []*models.APIVip{{IP: "1.2.3.11", ClusterID: clusterId}},
			IngressVips:            []*models.IngressVip{{IP: "1.2.3.12", ClusterID: clusterId}},
			InstallConfigOverrides: `{"fips":true}`,
			ImageInfo:              &models.ImageInfo{},
			Platform:               &models.Platform{Type: common.PlatformTypePtr(models.PlatformTypeBaremetal)},
			NetworkType:            swag.String("OpenShiftSDN"),
		}}
		clusterInfraenvs = []*common.InfraEnv{}
		id := strfmt.UUID(uuid.New().String())
		// By default all the hosts in the cluster are dual-stack
		host1 = models.Host{
			ID:        &id,
			ClusterID: &clusterId,
			Status:    swag.String(models.HostStatusKnown),
			Role:      "master",
			Inventory: getInventoryStr("hostname0", "bootMode", true, true),
			Bootstrap: true,
		}
		id = strfmt.UUID(uuid.New().String())
		host2 = models.Host{
			ID:        &id,
			ClusterID: &clusterId,
			Status:    swag.String(models.HostStatusKnown),
			Role:      "worker",
			Inventory: getInventoryStr("hostname1", "bootMode", true, true),
		}

		host3 = models.Host{
			ID:        &id,
			ClusterID: &clusterId,
			Status:    swag.String(models.HostStatusKnown),
			Role:      "worker",
			Inventory: getInventoryStr("hostname2", "bootMode", true, true),
		}

		cluster.Hosts = []*models.Host{&host1, &host2, &host3}
		installConfig = createInstallConfigBuilder()
	})

	It("create_configuration_with_all_hosts", func() {
		var result installcfg.InstallerConfigBaremetal
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
		Expect(result.Arbiter).Should(BeNil())
		Expect(result.FeatureSet).To(Equal(configv1.Default))
		Expect(result.FeatureGates).To(HaveLen(0))
	})

	It("create_configuration_with_all_hosts - TNA cluster TechPreview", func() {
		id := strfmt.UUID(uuid.New().String())
		host4 := models.Host{
			ID:        &id,
			ClusterID: cluster.ID,
			Status:    swag.String(models.HostStatusKnown),
			Role:      "arbiter",
			Inventory: getInventoryStr("hostname4", "bootMode", true, true),
		}
		cluster.Hosts = []*models.Host{&host1, &host2, &host4}
		cluster.OpenshiftVersion = common.MinimumVersionForArbiterClusters

		var result installcfg.InstallerConfigBaremetal
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
		Expect(result.Arbiter).Should(Not(BeNil()))
		Expect(result.Arbiter.Replicas).To(Equal(1))
		Expect(result.FeatureSet).To(Equal(configv1.TechPreviewNoUpgrade))
		Expect(result.FeatureGates).To(HaveLen(0))
	})

	It("create_configuration_with_all_hosts - TNA cluster", func() {
		id := strfmt.UUID(uuid.New().String())
		host4 := models.Host{
			ID:        &id,
			ClusterID: cluster.ID,
			Status:    swag.String(models.HostStatusKnown),
			Role:      "arbiter",
			Inventory: getInventoryStr("hostname4", "bootMode", true, true),
		}
		cluster.Hosts = []*models.Host{&host1, &host2, &host4}
		cluster.OpenshiftVersion = "4.20"

		var result installcfg.InstallerConfigBaremetal
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
		Expect(result.Arbiter).Should(Not(BeNil()))
		Expect(result.Arbiter.Replicas).To(Equal(1))
	})

	It("create_configuration_with_hostnames", func() {
		var result installcfg.InstallerConfigBaremetal
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	It("create_configuration_with_mirror_registries", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.OpenshiftVersion = "4.14.0-0.0"
		regData := []mirrorregistries.RegistriesConf{{Location: "location1", Mirror: []string{"mirror1"}}, {Location: "location2", Mirror: []string{"mirror2"}}}
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(true).Times(2)
		mockMirrorRegistriesConfigBuilder.EXPECT().ExtractLocationMirrorDataFromRegistries().Return(regData, nil).Times(1)
		mockMirrorRegistriesConfigBuilder.EXPECT().GetMirrorCA().Return([]byte("some sa data"), nil).Times(1)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
		Expect(result.ImageDigestSources).To(Equal([]installcfg.ImageDigestSource{
			installcfg.ImageDigestSource{Source: "location1",
				Mirrors: []string{"mirror1"},
			},
			installcfg.ImageDigestSource{Source: "location2",
				Mirrors: []string{"mirror2"},
			},
		}))
		Expect(result.DeprecatedImageContentSources).To(BeEmpty())
	})

	It("create_configuration_with_mirror_registries_idms_not_supported", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.OpenshiftVersion = "4.12.0-0.0"
		regData := []mirrorregistries.RegistriesConf{{Location: "location1", Mirror: []string{"mirror1"}}, {Location: "location2", Mirror: []string{"mirror2"}}}
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(true).Times(2)
		mockMirrorRegistriesConfigBuilder.EXPECT().ExtractLocationMirrorDataFromRegistries().Return(regData, nil).Times(1)
		mockMirrorRegistriesConfigBuilder.EXPECT().GetMirrorCA().Return([]byte("some sa data"), nil).Times(1)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.ImageDigestSources).To(BeEmpty())
		Expect(result.DeprecatedImageContentSources).To(Equal([]installcfg.ImageContentSource{
			installcfg.ImageContentSource{Source: "location1",
				Mirrors: []string{"mirror1"},
			},
			installcfg.ImageContentSource{Source: "location2",
				Mirrors: []string{"mirror2"},
			},
		}))
	})

	It("create_configuration_with_proxy", func() {
		var result installcfg.InstallerConfigBaremetal
		proxyURL := "http://proxyserver:3218"
		cluster.HTTPProxy = proxyURL
		cluster.HTTPSProxy = proxyURL
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Proxy.HTTPProxy).Should(Equal(proxyURL))
		Expect(result.Proxy.HTTPSProxy).Should(Equal(proxyURL))
		splitNoProxy := strings.Split(result.Proxy.NoProxy, ",")
		Expect(splitNoProxy).To(HaveLen(4))
		Expect(splitNoProxy).To(ContainElement(network.GetMachineCidrById(&cluster, 0)))
		Expect(splitNoProxy).To(ContainElement(string(cluster.ServiceNetworks[0].Cidr)))
		Expect(splitNoProxy).To(ContainElement(string(cluster.ClusterNetworks[0].Cidr)))
		domainName := "." + cluster.Name + "." + cluster.BaseDNSDomain
		Expect(splitNoProxy).To(ContainElement(domainName))
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	It("create_configuration_with_proxy_with_no_proxy", func() {
		var result installcfg.InstallerConfigBaremetal
		proxyURL := "http://proxyserver:3218"
		cluster.HTTPProxy = proxyURL
		cluster.HTTPSProxy = proxyURL
		cluster.NoProxy = "no-proxy.com"
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Proxy.HTTPProxy).Should(Equal(proxyURL))
		Expect(result.Proxy.HTTPSProxy).Should(Equal(proxyURL))
		splitNoProxy := strings.Split(result.Proxy.NoProxy, ",")
		Expect(splitNoProxy).To(HaveLen(5))
		Expect(splitNoProxy).To(ContainElement("no-proxy.com"))
		Expect(splitNoProxy).To(ContainElement(network.GetMachineCidrById(&cluster, 0)))
		Expect(splitNoProxy).To(ContainElement(string(cluster.ServiceNetworks[0].Cidr)))
		Expect(splitNoProxy).To(ContainElement(string(cluster.ClusterNetworks[0].Cidr)))
		domainName := "." + cluster.Name + "." + cluster.BaseDNSDomain
		Expect(splitNoProxy).To(ContainElement(domainName))
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	vsphereInstallConfigOverrides := `{"platform":{"vsphere":{"vcenters":[{"server":"vcenter.openshift.com","user":"testUser","password":"testPassword","datacenters":["testDatacenter"]}],"failureDomains":[{"name":"testfailureDomain","region":"testRegion","zone":"testZone","server":"vcenter.openshift.com","topology":{"datacenter":"testDatacenter","computeCluster":"/testDatacenter/host/testComputecluster","networks":["testNetwork"],"datastore":"/testDatacenter/datastore/testDatastore","resourcePool":"/testDatacenter/host/testComputecluster//Resources","folder":"/testDatacenter/vm/testFolder"}}]}}}`

	It("vSphere credentials in overrides is decoded correctly - installConfig.applyConfigOverrides", func() {
		var result installcfg.InstallerConfigBaremetal
		overrides := vsphereInstallConfigOverrides
		err := installConfig.applyConfigOverrides(overrides, &result)
		Expect(err).ShouldNot(HaveOccurred())
		assertVSphereCredentials(result)
	})

	It("vSphere credentials are unmarshalled correctly - installConfig.GetInstallConfig", func() {
		var result installcfg.InstallerConfigBaremetal
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		cluster.InstallConfigOverrides = vsphereInstallConfigOverrides
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		assertVSphereCredentials(result)
	})

	nutanixInstallConfigOverrides := `{"platform":{"nutanix":{"prismCentral":{"endpoint":{"address":"prismcentral.openshift.com","port":9440},"username":"testUser","password":"testPassword"},"prismElements":[{"uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx","endpoint":{"address":"prism.openshift.com","port":9440}}],"subnetUUIDs":["yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"],"apiVIPs":["10.0.0.1"],"ingressVIPs":["10.0.0.2"]}}}`

	It("Nutanix platform in overrides is decoded correctly - installConfig.applyConfigOverrides", func() {
		var result installcfg.InstallerConfigBaremetal
		overrides := nutanixInstallConfigOverrides
		err := installConfig.applyConfigOverrides(overrides, &result)
		Expect(err).ShouldNot(HaveOccurred())
		assertNutanixPlatform(result)
	})

	It("Nutanix platform are unmarshalled correctly - installConfig.GetInstallConfig", func() {
		var result installcfg.InstallerConfigBaremetal
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		cluster.InstallConfigOverrides = nutanixInstallConfigOverrides
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		assertNutanixPlatform(result)
	})

	It("Nutanix credentials are unmarshalled correctly - installConfig.GetInstallConfig", func() {
		var result installcfg.InstallerConfigBaremetal
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		cluster.InstallConfigOverrides = nutanixInstallConfigOverrides
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		assertNutanixCredentials(result)
	})

	It("correctly applies cluster overrides", func() {
		var result installcfg.InstallerConfigBaremetal
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		// test that overrides worked
		Expect(result.FIPS).Should(Equal(true))
		// test that existing values are kept
		Expect(result.APIVersion).Should(Equal("v1"))
		Expect(result.BaseDomain).Should(Equal("redhat.com"))
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	It("doesn't fail with empty overrides", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	It("doesn't fail with empty overrides, IPv6 machine CIDR", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		cluster.MachineNetworks = []*models.MachineNetwork{{Cidr: "1001:db8::/120"}}
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("doesn't fail with empty overrides, IPv6 cluster CIDR", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		cluster.ClusterNetworks = []*models.ClusterNetwork{{Cidr: "1001:db8::/120"}}
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("doesn't fail with empty overrides, IPv6 service CIDR", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		cluster.ServiceNetworks = []*models.ServiceNetwork{{Cidr: "1001:db8::/120"}}
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("Just redhat root CA", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		redhatRootCA := testBundle1
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, redhatRootCA)
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.AdditionalTrustBundle).Should(Equal(testBundle1))
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	It("Both mirror CA and redhat root CA", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""

		mirrorCA := testBundle2
		gomock.InOrder(mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(1),
			mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(true))
		mockMirrorRegistriesConfigBuilder.EXPECT().GetMirrorCA().Return([]byte(mirrorCA), nil).Times(1)

		rhRootCA := testBundle1
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, rhRootCA)
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.AdditionalTrustBundle).Should(Equal(fmt.Sprintf("%s\n%s", rhRootCA, mirrorCA)))
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	It("One infraenv", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		clusterInfraenvs = append(clusterInfraenvs, &common.InfraEnv{
			InfraEnv: models.InfraEnv{
				AdditionalTrustBundle: testBundle3,
			},
		})
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.AdditionalTrustBundle).Should(Equal(testBundle3))
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	It("Two infraenvs", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		clusterInfraenvs = append(clusterInfraenvs, &common.InfraEnv{
			InfraEnv: models.InfraEnv{
				AdditionalTrustBundle: testBundle3,
			},
		})
		clusterInfraenvs = append(clusterInfraenvs, &common.InfraEnv{
			InfraEnv: models.InfraEnv{
				AdditionalTrustBundle: testBundle4,
			},
		})
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.AdditionalTrustBundle).Should(Equal(fmt.Sprintf("%s\n%s", testBundle3, testBundle4)))
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	It("Two infraenvs and mirror CA", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""

		mirrorCA := testBundle2
		gomock.InOrder(mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(1),
			mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(true))
		mockMirrorRegistriesConfigBuilder.EXPECT().GetMirrorCA().Return([]byte(mirrorCA), nil).Times(1)

		clusterInfraenvs = append(clusterInfraenvs, &common.InfraEnv{
			InfraEnv: models.InfraEnv{
				AdditionalTrustBundle: testBundle3,
			},
		})
		clusterInfraenvs = append(clusterInfraenvs, &common.InfraEnv{
			InfraEnv: models.InfraEnv{
				AdditionalTrustBundle: testBundle4,
			},
		})
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.AdditionalTrustBundle).Should(Equal(fmt.Sprintf("%s\n%s\n%s", testBundle2, testBundle3, testBundle4)))
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	It("No certificates at all", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.AdditionalTrustBundle).Should(Equal(""))
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	It("CA AdditionalTrustBundle is set to mirror CA, don't overwrite InstallConfigOverrides certs", func() {
		var result installcfg.InstallerConfigBaremetal

		// Arbitrary install config override that happens to have additionalTrustBundle
		cluster.InstallConfigOverrides = fmt.Sprintf(
			`{"additionalTrustBundle":"%s","imageDigestSources":[{"mirrors":["f04-h09-000-r640.rdu2.scalelab.redhat.com:5000/localimages/local-release-image"],"source":"quay.io/openshift-release-dev/ocp-release"},{"mirrors":["f04-h09-000-r640.rdu2.scalelab.redhat.com:5000/localimages/local-release-image"],"source":"quay.io/openshift-release-dev/ocp-v4.0-art-dev"}]}`,
			strings.ReplaceAll(testBundle4, "\n", "\\n"))

		ca := testBundle1
		mirrorCA := testBundle2

		gomock.InOrder(mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(1),
			mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(true))
		mockMirrorRegistriesConfigBuilder.EXPECT().GetMirrorCA().Return([]byte(mirrorCA), nil).Times(1)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, ca)
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.AdditionalTrustBundle).Should(Equal(fmt.Sprintf("%s\n%s\n%s", testBundle4, testBundle1, testBundle2)))
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	It("UserManagedNetworking None Platform", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		cluster.UserManagedNetworking = swag.Bool(true)
		cluster.Platform = &models.Platform{Type: common.PlatformTypePtr(models.PlatformTypeNone)}
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Platform.Baremetal).Should(BeNil())
		var none = installcfg.PlatformNone{}
		Expect(*result.Platform.None).Should(Equal(none))
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	It("UserManagedNetworking None Platform Machine network", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		cluster.UserManagedNetworking = swag.Bool(true)
		cluster.Platform = &models.Platform{Type: common.PlatformTypePtr(models.PlatformTypeNone)}
		cluster.MachineNetworks = []*models.MachineNetwork{}
		host1.Bootstrap = true
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Platform.Baremetal).Should(BeNil())
		var none = installcfg.PlatformNone{}
		Expect(*result.Platform.None).Should(Equal(none))
		Expect(result.Networking.MachineNetwork[0].Cidr).Should(Equal("1.2.3.0/24"))
	})

	It("UserManagedNetworking None Platform Machine network IPV6 only", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		cluster.UserManagedNetworking = swag.Bool(true)
		cluster.Platform = &models.Platform{Type: common.PlatformTypePtr(models.PlatformTypeNone)}
		cluster.MachineNetworks = []*models.MachineNetwork{}
		host1.Bootstrap = true
		host1.Inventory = getInventoryStr("hostname0", "bootMode", false, true)
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Platform.Baremetal).Should(BeNil())
		var none = installcfg.PlatformNone{}
		Expect(*result.Platform.None).Should(Equal(none))
		Expect(len(result.Networking.MachineNetwork)).Should(Equal(1))
		Expect(result.Networking.MachineNetwork[0].Cidr).Should(Equal("1001:db8::/120"))
	})

	It("UserManagedNetworking None Platform Machine network dual-stack", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		cluster.UserManagedNetworking = swag.Bool(true)
		cluster.Platform = &models.Platform{Type: common.PlatformTypePtr(models.PlatformTypeNone)}
		cluster.MachineNetworks = []*models.MachineNetwork{}
		host1.Bootstrap = true
		host1.Inventory = getInventoryStr("hostname0", "bootMode", true, true)
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Platform.Baremetal).Should(BeNil())
		var none = installcfg.PlatformNone{}
		Expect(*result.Platform.None).Should(Equal(none))
		Expect(len(result.Networking.MachineNetwork)).Should(Equal(2))
		Expect(result.Networking.MachineNetwork[0].Cidr).Should(Equal("1.2.3.0/24"))
		Expect(result.Networking.MachineNetwork[1].Cidr).Should(Equal("1001:db8::/120"))
	})

	It("UserManagedNetworking BareMetal", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		cluster.UserManagedNetworking = swag.Bool(false)
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Platform.None).Should(BeNil())
	})

	It("Single node", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		cluster.UserManagedNetworking = swag.Bool(true)
		cluster.Platform = &models.Platform{Type: common.PlatformTypePtr(models.PlatformTypeNone)}
		cluster.ControlPlaneCount = 1
		cluster.Hosts[0].Bootstrap = true
		cluster.Hosts[0].InstallationDiskPath = "/dev/test"
		cluster.MachineNetworks = []*models.MachineNetwork{{Cidr: "1.2.3.0/24"}}
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)

		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Platform.Baremetal).Should(BeNil())
		Expect(*result.Platform.None).Should(Equal(installcfg.PlatformNone{}))
		Expect(result.BootstrapInPlace.InstallationDisk).Should(Equal("/dev/test"))
		Expect(len(result.Networking.MachineNetwork)).Should(Equal(1))
		Expect(result.Networking.MachineNetwork[0].Cidr).Should(Equal(network.GetMachineCidrById(&cluster, 0)))
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOpenShiftSDN))
	})

	It("Single node with default network type", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		cluster.UserManagedNetworking = swag.Bool(true)
		cluster.Platform = &models.Platform{Type: common.PlatformTypePtr(models.PlatformTypeNone)}
		cluster.ControlPlaneCount = 1
		cluster.NetworkType = nil
		cluster.Hosts[0].Bootstrap = true
		cluster.Hosts[0].InstallationDiskPath = "/dev/test"
		cluster.MachineNetworks = []*models.MachineNetwork{{Cidr: "1.2.3.0/24"}}
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)

		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Platform.Baremetal).Should(BeNil())
		Expect(*result.Platform.None).Should(Equal(installcfg.PlatformNone{}))
		Expect(result.BootstrapInPlace.InstallationDisk).Should(Equal("/dev/test"))
		Expect(len(result.Networking.MachineNetwork)).Should(Equal(1))
		Expect(result.Networking.MachineNetwork[0].Cidr).Should(Equal(network.GetMachineCidrById(&cluster, 0)))
		Expect(result.Networking.NetworkType).To(Equal(models.ClusterNetworkTypeOVNKubernetes))
	})

	It("Single node IPV6 only", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		cluster.UserManagedNetworking = swag.Bool(true)
		cluster.Platform = &models.Platform{Type: common.PlatformTypePtr(models.PlatformTypeNone)}
		cluster.MachineNetworks = []*models.MachineNetwork{{Cidr: "fe80::/64"}}
		host1.Bootstrap = true
		host1.Inventory = getInventoryStr("hostname0", "bootMode", false, true)
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Platform.Baremetal).Should(BeNil())
		var none = installcfg.PlatformNone{}
		Expect(*result.Platform.None).Should(Equal(none))
		Expect(len(result.Networking.MachineNetwork)).Should(Equal(1))
		Expect(result.Networking.MachineNetwork[0].Cidr).Should(Equal(network.GetMachineCidrById(&cluster, 0)))
	})

	It("Single node dual-stack", func() {
		var result installcfg.InstallerConfigBaremetal
		cluster.InstallConfigOverrides = ""
		cluster.UserManagedNetworking = swag.Bool(true)
		cluster.Platform = &models.Platform{Type: common.PlatformTypePtr(models.PlatformTypeNone)}
		cluster.MachineNetworks = []*models.MachineNetwork{{Cidr: "1.2.3.0/24"}, {Cidr: "1001:db8::/120"}}
		host1.Bootstrap = true
		host1.Inventory = getInventoryStr("hostname0", "bootMode", true, true)
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Platform.Baremetal).Should(BeNil())
		var none = installcfg.PlatformNone{}
		Expect(*result.Platform.None).Should(Equal(none))
		Expect(len(result.Networking.MachineNetwork)).Should(Equal(2))
		Expect(result.Networking.MachineNetwork[0].Cidr).Should(Equal(network.GetMachineCidrById(&cluster, 0)))
		Expect(result.Networking.MachineNetwork[1].Cidr).Should(Equal(network.GetMachineCidrById(&cluster, 1)))
	})

	DescribeTable("Hyperthreading config", func(clusterHyperthreading, controlPlaneHyperthreading, arbiterHyperthreading, computeHyperthreading string) {
		cluster.Hyperthreading = clusterHyperthreading
		if arbiterHyperthreading != "" {
			cluster.Hosts[2].Role = models.HostRoleArbiter
		}
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(1)
		data, err := installConfig.getBasicInstallConfig(&cluster)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(data.ControlPlane.Hyperthreading).Should(Equal(controlPlaneHyperthreading))
		if arbiterHyperthreading != "" {
			Expect(data.Arbiter.Hyperthreading).Should(Equal(arbiterHyperthreading))
		} else {
			Expect(data.Arbiter).To(BeNil())
		}
		Expect(data.Compute[0].Hyperthreading).Should(Equal(computeHyperthreading))
	},
		Entry("all", models.ClusterHyperthreadingAll, "Enabled", "", "Enabled"),
		Entry("all - TNA cluster", models.ClusterHyperthreadingAll, "Enabled", "Enabled", "Enabled"),
		Entry("none", models.ClusterHyperthreadingNone, "Disabled", "", "Disabled"),
		Entry("none - TNA cluster", models.ClusterHyperthreadingNone, "Disabled", "Disabled", "Disabled"),
		Entry("masters", models.ClusterHyperthreadingMasters, "Enabled", "", "Disabled"),
		Entry("masters - TNA cluster", models.ClusterHyperthreadingMasters, "Enabled", "Disabled", "Disabled"),
		Entry("arbiters", models.ClusterHyperthreadingArbiters, "Disabled", "", "Disabled"),
		Entry("arbiters - TNA cluster", models.ClusterHyperthreadingArbiters, "Disabled", "Enabled", "Disabled"),
		Entry("workers", models.ClusterHyperthreadingWorkers, "Disabled", "", "Enabled"),
		Entry("workers - TNA cluster", models.ClusterHyperthreadingWorkers, "Disabled", "Disabled", "Enabled"),
		Entry("masters,arbiters", models.ClusterHyperthreadingMastersArbiters, "Enabled", "", "Disabled"),
		Entry("masters,arbiters - TNA cluster", models.ClusterHyperthreadingMastersArbiters, "Enabled", "Enabled", "Disabled"),
		Entry("masters,workers", models.ClusterHyperthreadingMastersWorkers, "Enabled", "", "Enabled"),
		Entry("masters,workers - TNA cluster", models.ClusterHyperthreadingMastersWorkers, "Enabled", "Disabled", "Enabled"),
		Entry("arbiters,workers", models.ClusterHyperthreadingArbitersWorkers, "Disabled", "", "Enabled"),
		Entry("arbiters,workers - TNA cluster", models.ClusterHyperthreadingArbitersWorkers, "Disabled", "Enabled", "Enabled"),
		Entry("masters,arbiters,workers", models.ClusterHyperthreadingMastersArbitersWorkers, "Enabled", "", "Enabled"),
		Entry("masters,arbiters,workers - TNA cluster", models.ClusterHyperthreadingMastersArbitersWorkers, "Enabled", "Enabled", "Enabled"),
	)

	It("CPUPartitioningMode config overrides", func() {
		var result installcfg.InstallerConfigBaremetal
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		cluster.InstallConfigOverrides = `{"cpuPartitioningMode":"AllNodes"}`
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		// test that overrides worked
		Expect(string(result.CPUPartitioningMode)).Should(Equal("AllNodes"))
	})

	It("AdditionalTrustBundlePolicy config overrides", func() {
		var result installcfg.InstallerConfigBaremetal
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		cluster.InstallConfigOverrides = `{"additionalTrustBundlePolicy":"Always"}`
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		// test that overrides worked
		Expect(string(result.AdditionalTrustBundlePolicy)).Should(Equal("Always"))
	})

	It("Baremetal host BMC configuration overrides", func() {
		var result installcfg.InstallerConfigBaremetal
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		cluster.InstallConfigOverrides = `{"platform":{"baremetal":{"hosts":[{"name":"master-0","bmc":{"username":"admin","password":"pwd","address":"http://10.10.10.1:8000/v1/Systems","disableCertificateVerification":false},"role":"","bootMACAddress":"00:65:0f:82:fd:3b","hardwareProfile":""},{"name":"master-1","bmc":{"username":"admin2","password":"pwd2","address":"http://10.10.10.2:8000/v1/Systems","disableCertificateVerification":false},"role":"","bootMACAddress":"00:65:0f:82:fd:3f","hardwareProfile":""},{"name":"master-2","bmc":{"username":"admin3","password":"pwd3","address":"http://10.10.10.3:8000/v1/Systems","disableCertificateVerification":false},"role":"","bootMACAddress":"00:65:0f:82:fd:43","hardwareProfile":""}],"clusterProvisioningIP":"172.22.0.3","provisioningNetwork":"Managed","provisioningNetworkInterface":"enp1s0","provisioningNetworkCIDR":"172.22.0.0/24","provisioningDHCPRange":"172.22.0.10,172.22.0.254"}}}`
		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())
		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())
		assertBaremetalHostBMCConfig(result)
	})

	It("cluster-managed load balancer should not move bootstrap host machine network to the first element", func() {
		var result installcfg.InstallerConfigBaremetal
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)

		cluster.MachineNetworks = []*models.MachineNetwork{
			{Cidr: "1.2.4.0/24"},
			{Cidr: "1.2.3.0/24"},
		}
		cluster.LoadBalancer = &models.LoadBalancer{Type: models.LoadBalancerTypeClusterManaged}

		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())

		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(result.Networking.MachineNetwork).To(HaveLen(2))
		Expect(result.Networking.MachineNetwork[0].Cidr).To(Equal("1.2.4.0/24"))
	})

	It("user-managed load balancer should move bootstrap host machine network to the first element", func() {
		var result installcfg.InstallerConfigBaremetal
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)

		cluster.MachineNetworks = []*models.MachineNetwork{
			{Cidr: "1.2.4.0/24"},
			{Cidr: "1.2.3.0/24"},
		}
		cluster.LoadBalancer = &models.LoadBalancer{Type: models.LoadBalancerTypeUserManaged}

		data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
		Expect(err).ShouldNot(HaveOccurred())

		err = json.Unmarshal(data, &result)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(result.Networking.MachineNetwork).To(HaveLen(2))
		Expect(result.Networking.MachineNetwork[0].Cidr).To(Equal("1.2.3.0/24"))
	})

	Context("networking", func() {
		It("Single network fields", func() {
			var result installcfg.InstallerConfigBaremetal
			mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
			data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(json.Unmarshal(data, &result)).ShouldNot(HaveOccurred())
			Expect(result.Networking.ClusterNetwork).To(HaveLen(1))
			Expect(result.Networking.MachineNetwork).To(HaveLen(1))
			Expect(result.Networking.ServiceNetwork).To(HaveLen(1))
		})

		It("Multiple network fields", func() {
			cluster.ClusterNetworks = []*models.ClusterNetwork{
				{
					Cidr:       "1.3.0.0/16",
					HostPrefix: 24,
				},
				{
					Cidr:       "1.3.0.0/16",
					HostPrefix: 24,
				},
			}
			cluster.ServiceNetworks = []*models.ServiceNetwork{
				{
					Cidr: "1.2.5.0/24",
				},
				{
					Cidr: "1.4.0.0/16",
				},
			}
			cluster.MachineNetworks = []*models.MachineNetwork{
				{
					Cidr: "1.2.3.0/24",
				},
				{
					Cidr: "1.2.3.0/24",
				},
			}

			var result installcfg.InstallerConfigBaremetal
			mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
			data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(json.Unmarshal(data, &result)).ShouldNot(HaveOccurred())
			Expect(result.Networking.ClusterNetwork).To(HaveLen(2))
			Expect(result.Networking.MachineNetwork).To(HaveLen(2))
			Expect(result.Networking.ServiceNetwork).To(HaveLen(2))
		})
	})

	Context("Mirror Registry", func() {

		const (
			mirrorRegistryCertificate = "-----BEGIN CERTIFICATE-----\n    certificate contents\n-----END CERTIFICATE------"
			sourceRegistry            = "quay.io"
			mirrorRegistry            = "example-user-registry.com"
		)

		getSecureRegistryToml := func() string {
			return fmt.Sprintf(`
[[registry]]
location = "%s"

[[registry.mirror]]
location = "%s"
`,
				sourceRegistry,
				mirrorRegistry,
			)
		}

		getMirrorRegistryConfigurations := func(registriesToml, certificate string) (*common.MirrorRegistryConfiguration, []configv1.ImageDigestMirrors) {
			imageDigestMirrors, imageTagMirrors, insecure, err := mirrorregistries.GetImageRegistries(registriesToml)
			Expect(err).To(Not(HaveOccurred()))

			mirrors := &common.MirrorRegistryConfiguration{
				ImageDigestMirrors: imageDigestMirrors,
				ImageTagMirrors:    imageTagMirrors,
				Insecure:           insecure,
				CaBundleCrt:        certificate,
				RegistriesConf:     registriesToml,
			}

			return mirrors, imageDigestMirrors
		}

		It("service MirrorRegistriesConfig and cluster MirrorRegistryConfiguration are both set - only MirrorRegistryConfiguration applied", func() {
			var result installcfg.InstallerConfigBaremetal
			cluster.OpenshiftVersion = "4.16.0-0.0"

			mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(true).Times(1)
			mockMirrorRegistriesConfigBuilder.EXPECT().GetMirrorCA().Return([]byte("service-ca-data"), nil).Times(1)

			// Set cluster-level MirrorRegistryConfiguration
			mirrors, imageDigestMirrors := getMirrorRegistryConfigurations(getSecureRegistryToml(), mirrorRegistryCertificate)
			cluster.SetMirrorRegistryConfiguration(mirrors)

			// Generate install config
			data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
			Expect(err).ShouldNot(HaveOccurred())

			err = json.Unmarshal(data, &result)
			Expect(err).ShouldNot(HaveOccurred())

			// Ensure only one mirror registry source is applied
			Expect(len(result.ImageDigestSources)).To(Equal(len(imageDigestMirrors)))
			Expect(len(result.ImageDigestSources)).To(Equal(1))
			Expect(result.ImageDigestSources[0].Source).To(Equal(imageDigestMirrors[0].Source))
			Expect(len(result.ImageDigestSources[0].Mirrors)).To(Equal(len(imageDigestMirrors[0].Mirrors)))
			Expect(result.ImageDigestSources[0].Mirrors[0]).To(Equal(string(imageDigestMirrors[0].Mirrors[0])))

			// Validate trust bundle concatenation
			Expect(result.AdditionalTrustBundle).To(Equal(fmt.Sprintf("%s\n%s", "service-ca-data", mirrorRegistryCertificate)))
		})

		It("success - cluster holds mirror registry configurations", func() {
			var result installcfg.InstallerConfigBaremetal
			cluster.OpenshiftVersion = "4.16.0-0.0"
			mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(1)

			mirrors, imageDigestMirrors := getMirrorRegistryConfigurations(getSecureRegistryToml(), mirrorRegistryCertificate)
			cluster.SetMirrorRegistryConfiguration(mirrors)
			data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
			Expect(err).ShouldNot(HaveOccurred())

			err = json.Unmarshal(data, &result)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(len(result.ImageDigestSources)).To(Equal(len(imageDigestMirrors)))
			Expect(len(result.ImageDigestSources)).To(Equal(1))
			Expect(result.ImageDigestSources[0].Source).To(Equal(imageDigestMirrors[0].Source))
			Expect(len(result.ImageDigestSources[0].Mirrors)).To(Equal(len(imageDigestMirrors[0].Mirrors)))
			Expect(result.ImageDigestSources[0].Mirrors[0]).To(Equal(string(imageDigestMirrors[0].Mirrors[0])))

			Expect(result.DeprecatedImageContentSources).To(BeEmpty())
		})

		It("success - cluster and infraenvs holds mirror registry configurations", func() {
			var result installcfg.InstallerConfigBaremetal
			cluster.OpenshiftVersion = "4.16.0-0.0"
			mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(1)

			mirrors, _ := getMirrorRegistryConfigurations(getSecureRegistryToml(), mirrorRegistryCertificate)
			cluster.SetMirrorRegistryConfiguration(mirrors)
			testBundles := []string{testBundle1, testBundle2, testBundle3}
			for i := 0; i < 3; i++ {
				infraenv := &common.InfraEnv{InfraEnv: models.InfraEnv{}}
				mirrors.CaBundleCrt = testBundles[i]
				infraenv.SetMirrorRegistryConfiguration(mirrors)
				clusterInfraenvs = append(clusterInfraenvs, infraenv)
			}
			data, err := installConfig.GetInstallConfig(&cluster, clusterInfraenvs, "")
			Expect(err).ShouldNot(HaveOccurred())

			err = json.Unmarshal(data, &result)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result.AdditionalTrustBundle).Should(Equal(fmt.Sprintf("%s\n%s\n%s\n%s", mirrorRegistryCertificate, testBundle1, testBundle2, testBundle3)))
		})
	})

	AfterEach(func() {
		// cleanup
		ctrl.Finish()
	})
})

var _ = Describe("ValidateInstallConfigPatch", func() {
	var (
		cluster          *common.Cluster
		installConfig    *installConfigBuilder
		clusterInfraenvs []*common.InfraEnv
	)
	BeforeEach(func() {
		id := strfmt.UUID(uuid.New().String())
		cluster = &common.Cluster{Cluster: models.Cluster{
			ID:               &id,
			OpenshiftVersion: "4.6",
			BaseDNSDomain:    "example.com",
			APIVips:          []*models.APIVip{{IP: "102.345.34.34", ClusterID: id}},
			IngressVips:      []*models.IngressVip{{IP: "376.5.56.6", ClusterID: id}},
			ImageInfo:        &models.ImageInfo{},
			Platform:         &models.Platform{Type: common.PlatformTypePtr(models.PlatformTypeBaremetal)},
		}}
		clusterInfraenvs = []*common.InfraEnv{}
		installConfig = createInstallConfigBuilder()
	})

	It("Succeeds when provided valid json", func() {
		s := `{"apiVersion": "v3", "baseDomain": "example.com", "metadata": {"name": "things"}}`
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		err := installConfig.ValidateInstallConfigPatch(cluster, clusterInfraenvs, s)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("Fails when provided invalid json", func() {
		s := `{"apiVersion": 3, "baseDomain": "example.com", "metadata": {"name": "things"}}`
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		err := installConfig.ValidateInstallConfigPatch(cluster, clusterInfraenvs, s)
		Expect(err).Should(HaveOccurred())
	})

	It("Fails when provided invalid json fields", func() {
		s := `{"apiVersion": "v3", "foo": "example.com", "metadata": {"name": "things"}}`
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		err := installConfig.ValidateInstallConfigPatch(cluster, clusterInfraenvs, s)
		Expect(err).Should(HaveOccurred())
	})

	It("Fails with an invalid cert", func() {
		s := `{"additionalTrustBundle":  "-----BEGIN CERTIFICATE-----\nMIIFozCCA4ugAwIBAgIUVlT4eKQQ43HN31jQzsez+iEmpw8wDQYJKoZIhvcNAQEL\nBQAwYTELMAkGA1UEBhMCQUExFTATBgNVBAcMDERlZmF1bHQgQ2l0eTEcMBoGA1UE\nCgwTRGVmYXVsdCBDb21wYW55IEx0ZDEdMBsGA1UEAwwUcmVnaXN0cnkuZXhhbXBs\nZS5jb20wHhcNMjAxMDI3MTI0OTEwWhcNMjExMDI3MTI0OTEwWjBhMQswCQYDVQQG\nEwJBQTEVMBMGA1UEBwwMRGVmYXVsdCBDaXR5MRwwGgYDVQQKDBNEZWZhdWx0IENv\nbXBhbnkgTHRkMR0wGwYDVQQDDBRyZWdpc3RyeS5leGFtcGxlLmNvbTCCAiIwDQYJ\nKoZIhvcNAQEBBQADggIPADCCAgoCggIBAKm/wEl5B6lDOwYtkOxoLHQySA5RySEU\nkEMoGxBtGewLjLRMS9zp5pgYNcRenOTfUeyx6n4vE+lLn6p4laSig6QGDK0mmPl/\nt8OVZGBNE/dOZEoGe3I+gQux0oErhzjNxrf1EGfeBRVVuSqmgQnFaeLq2mGsbb5+\nyz114seD7u0Vb6OIX5sA+ytvr+jV3HK0jf5H9AHvSnNzF0UE+S7CHTJSDqQNUPxp\n8rAtfOvWyndDJBBmA0fdnDRYNtUqKcj/YBSntuZAmSJ0Woq9NrE+H3e61kvF0AP8\nHz21FSD/GqCn97Q8Mh8uTKx8jas2XBLyWdi0OCIV+a4jTadez1zPCWT+zgD5rHAk\np5RyXgkRU3guJydNMlpRPsGur3pUM4Q3zQfArZ+OxTkU/SLZbBmAVMPDI2pwL6qE\n2F8So4JdysH1MiwtYDYVIxKChrpBtTVunIe+Jyl/w8a3xR77r++3MFauobGLpeCL\nptbSz0aFZIIIwoLw2JVaWe7BWryjk8fDYrlPkLWqgQ956lcZppqiUzvEVv3p7wC2\nmfWkXJBGZZ0CZcYUoEE7zQ5T0RHLXqf0lSMf8I1SPzBF+Wl6G2gUOaZtYT5s0LA5\nid+gSDtKqyDH1HwPGO0eQB1LGeXOCLBA3cgmxYXtIMLfds0LgcJF+vRV3868abpD\n+yVMxGQRzRZFAgMBAAGjUzBRMB0GA1UdDgQWBBTUHUuivG1L6rTHS9v8KHTtOVpL\ncjAfBgNVHSMEGDAWgBTUHUuivG1L6rTHS9v8KHTtOVpLcjAPBgNVHRMBAf8EBTAD\nAQH/MA0GCSqGSIb3DQEBCwUAA4ICAQAFTmSriXnTJ/9cbO2lJmH7OFcrgKWdsycU\ngc9aeLSpnlYPuMjRHgXpq0X5iZzJaOXu8WKmbxTItxfd7MD/9rsaDMo7uDs6cZhC\nsdpWDVzZlP1PRcy1uT3+g12QmMmt89WBtauKEMukI3mOlx6y1VzPj9Vw5gfBKYjS\nh2NJPSVzgkLlLTOsY6bHesXVWrHVtCS5fUiE2xNkE6hXS0hZWYZlzLwn55wIrchx\nB3G++mPnNL3SbH62lXyWcrc1M/+gNl3F3jSd5WfxZQVllZ9vK1DnBKDisTUax5fR\nqK/D7vgkvHJa0USzGhcYV3DEdbgP/COgWrpbA0TTFcasWWYQdBk+2EUPcWKAh0JB\nVgql3o0pmyzfqQtuRRMC4D6Ip6y6IE2opK2c7ipXT4iEyPqr4uk4IeVFXghCYW92\nkCI+FyRJgbSu9ZuIug8AUlea7UOLTC4mxAayXvTwA6bNlGoSLmojgQHG7GlGj+E8\n57AHM2sD9Qi1VYyLuMVhJB3DzlQKtEFuvZsvi/rSIGqT8UfNbxk7OCtxceyzECqW\n2ptIv7tDhQeAGqkGqhTj1WdH+16+QZpsfmkwt5+hAaOeZfQ/nOCP7CGwbl4nYc3X\narDiqhVUXlv84/7XrOyoDJo3AVGidq902h6MYenX9T//XYbWkUK7nkvYMVoxu/Ek\nx/aT+8yOHQ==\n", "imageDigestSources": [{"mirrors": ["registry.example.com:5000/ocp4"], "source": "quay.io/openshift-release-dev/ocp-release"}, {"mirrors": ["registry.example.com:5000/ocp4"], "source": "quay.io/openshift-release-dev/ocp-release-nightly"}, {"mirrors": ["registry.example.com:5000/ocp4"], "source": "quay.io/openshift-release-dev/ocp-v4.0-art-dev"}]}`
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		//providerRegistry.EXPECT().AddPlatformToInstallConfig(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		err := installConfig.ValidateInstallConfigPatch(cluster, clusterInfraenvs, s)
		Expect(err).Should(HaveOccurred())
	})

	It("Single node - valid json without bootstrap node", func() {
		s := `{"apiVersion": "v3", "baseDomain": "example.com", "metadata": {"name": "things"}}`
		cluster.UserManagedNetworking = swag.Bool(true)
		cluster.Platform = &models.Platform{Type: common.PlatformTypePtr(models.PlatformTypeNone)}
		cluster.ControlPlaneCount = 1
		mockMirrorRegistriesConfigBuilder.EXPECT().IsMirrorRegistriesConfigured().Return(false).Times(2)
		err := installConfig.ValidateInstallConfigPatch(cluster, clusterInfraenvs, s)
		Expect(err).ShouldNot(HaveOccurred())
	})
})

// asserts credential values against vsphereInstallConfigOverrides
func assertVSphereCredentials(result installcfg.InstallerConfigBaremetal) {
	Expect(result.Platform.Vsphere.VCenters[0].Server).Should(Equal("vcenter.openshift.com"))
	Expect(result.Platform.Vsphere.VCenters[0].Username).Should(Equal("testUser"))
	Expect(result.Platform.Vsphere.VCenters[0].Password.String()).Should(Equal("testPassword"))
	Expect(len(result.Platform.Vsphere.VCenters[0].Datacenters)).Should(Equal(1))
	Expect(result.Platform.Vsphere.VCenters[0].Datacenters[0]).Should(Equal("testDatacenter"))
	Expect(result.Platform.Vsphere.FailureDomains[0].Name).Should(Equal("testfailureDomain"))
	Expect(result.Platform.Vsphere.FailureDomains[0].Region).Should(Equal("testRegion"))
	Expect(result.Platform.Vsphere.FailureDomains[0].Server).Should(Equal("vcenter.openshift.com"))
	Expect(result.Platform.Vsphere.FailureDomains[0].Topology.Datacenter).Should(Equal("testDatacenter"))
	Expect(result.Platform.Vsphere.FailureDomains[0].Topology.ComputeCluster).Should(Equal("/testDatacenter/host/testComputecluster"))
	Expect(len(result.Platform.Vsphere.FailureDomains[0].Topology.Networks)).Should(Equal(1))
	Expect(result.Platform.Vsphere.FailureDomains[0].Topology.Networks[0]).Should(Equal("testNetwork"))
	Expect(result.Platform.Vsphere.FailureDomains[0].Topology.Datastore).Should(Equal("/testDatacenter/datastore/testDatastore"))
	Expect(result.Platform.Vsphere.FailureDomains[0].Topology.ResourcePool).Should(Equal("/testDatacenter/host/testComputecluster//Resources"))
	Expect(result.Platform.Vsphere.FailureDomains[0].Topology.Folder).Should(Equal("/testDatacenter/vm/testFolder"))
}

// asserts platform values against nutanixInstallConfigOverrides
func assertNutanixPlatform(result installcfg.InstallerConfigBaremetal) {
	Expect(len(result.Platform.Nutanix.APIVIPs)).Should(Equal(1))
	Expect(len(result.Platform.Nutanix.IngressVIPs)).Should(Equal(1))
	Expect(result.Platform.Nutanix.PrismCentral.Endpoint.Address).Should(Equal("prismcentral.openshift.com"))
	Expect(result.Platform.Nutanix.PrismCentral.Endpoint.Port).Should(Equal(int32(9440)))
	Expect(string(result.Platform.Nutanix.PrismElements[0].UUID)).Should(Equal("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"))
	Expect(result.Platform.Nutanix.PrismElements[0].Endpoint.Address).Should(Equal("prism.openshift.com"))
	Expect(result.Platform.Nutanix.PrismElements[0].Endpoint.Port).Should(Equal(int32(9440)))
	Expect(len(result.Platform.Nutanix.SubnetUUIDs)).Should(Equal(1))
}

// asserts platform values against nutanixInstallConfigOverrides
func assertNutanixCredentials(result installcfg.InstallerConfigBaremetal) {
	Expect(result.Platform.Nutanix.PrismCentral.Username).Should(Equal("testUser"))
	Expect(string(result.Platform.Nutanix.PrismCentral.Password)).Should(Equal("testPassword"))
}

// asserts Baremetal Host BMC Configuration set by InstallConfigOverrides
func assertBaremetalHostBMCConfig(result installcfg.InstallerConfigBaremetal) {
	Expect(result.Platform.Baremetal.Hosts[0].Name).Should(Equal("master-0"))
	Expect(result.Platform.Baremetal.Hosts[0].BMC.Username).Should(Equal("admin"))
	Expect(result.Platform.Baremetal.Hosts[0].BMC.Password).Should(Equal("pwd"))
	Expect(result.Platform.Baremetal.Hosts[0].BMC.Address).Should(Equal("http://10.10.10.1:8000/v1/Systems"))
	Expect(result.Platform.Baremetal.Hosts[0].BMC.DisableCertificateVerification).Should(Equal(false))
	Expect(result.Platform.Baremetal.Hosts[1].Name).Should(Equal("master-1"))
	Expect(result.Platform.Baremetal.Hosts[1].BMC.Username).Should(Equal("admin2"))
	Expect(result.Platform.Baremetal.Hosts[1].BMC.Password).Should(Equal("pwd2"))
	Expect(result.Platform.Baremetal.Hosts[1].BMC.Address).Should(Equal("http://10.10.10.2:8000/v1/Systems"))
	Expect(result.Platform.Baremetal.Hosts[1].BMC.DisableCertificateVerification).Should(Equal(false))
	Expect(result.Platform.Baremetal.Hosts[2].Name).Should(Equal("master-2"))
	Expect(result.Platform.Baremetal.Hosts[2].BMC.Username).Should(Equal("admin3"))
	Expect(result.Platform.Baremetal.Hosts[2].BMC.Password).Should(Equal("pwd3"))
	Expect(result.Platform.Baremetal.Hosts[2].BMC.Address).Should(Equal("http://10.10.10.3:8000/v1/Systems"))
	Expect(result.Platform.Baremetal.Hosts[2].BMC.DisableCertificateVerification).Should(Equal(false))
	Expect(result.Platform.Baremetal.ClusterProvisioningIP).Should(Equal("172.22.0.3"))
	Expect(result.Platform.Baremetal.ProvisioningNetwork).Should(Equal("Managed"))
	Expect(result.Platform.Baremetal.ProvisioningNetworkInterface).Should(Equal("enp1s0"))
	Expect(*result.Platform.Baremetal.ProvisioningNetworkCIDR).Should(Equal("172.22.0.0/24"))
	Expect(result.Platform.Baremetal.ProvisioningDHCPRange).Should(Equal("172.22.0.10,172.22.0.254"))
}

func getInventoryStr(hostname, bootMode string, ipv4 bool, ipv6 bool) string {
	inventory := models.Inventory{
		Hostname: hostname,
		Boot:     &models.Boot{CurrentBootMode: bootMode},
		Interfaces: []*models.Interface{
			{
				IPV4Addresses: []string{},
				IPV6Addresses: []string{},
				MacAddress:    "some MAC address",
			},
		},
	}
	if ipv4 {
		inventory.Interfaces[0].IPV4Addresses = []string{"1.2.3.1/24"}

	}
	if ipv6 {
		inventory.Interfaces[0].IPV6Addresses = []string{"1001:db8::1/120"}

	}
	ret, _ := json.Marshal(&inventory)
	return string(ret)
}

var _ = Describe("Generate NoProxy", func() {
	var (
		cluster       *common.Cluster
		installConfig *installConfigBuilder
	)
	BeforeEach(func() {
		cluster = &common.Cluster{Cluster: models.Cluster{
			MachineNetworks: []*models.MachineNetwork{{Cidr: "1.2.3.0/24"}},
			Name:            "proxycluster",
			BaseDNSDomain:   "myproxy.com",
			ClusterNetworks: []*models.ClusterNetwork{{Cidr: "192.168.1.0/24"}},
			ServiceNetworks: []*models.ServiceNetwork{{Cidr: "fe80::1/64"}},
			Platform:        &models.Platform{Type: common.PlatformTypePtr(models.PlatformTypeBaremetal)},
		}}
		installConfig = createInstallConfigBuilder()
	})
	It("Default NoProxy", func() {
		noProxy := installConfig.generateNoProxy(cluster)
		Expect(noProxy).Should(Equal(fmt.Sprintf(".proxycluster.myproxy.com,%s,%s,%s",
			cluster.ClusterNetworks[0].Cidr, cluster.ServiceNetworks[0].Cidr, network.GetMachineCidrById(cluster, 0))))
	})
	It("Update NoProxy", func() {
		cluster.NoProxy = "domain.org,127.0.0.2"
		noProxy := installConfig.generateNoProxy(cluster)
		Expect(noProxy).Should(Equal(fmt.Sprintf("domain.org,127.0.0.2,.proxycluster.myproxy.com,%s,%s,%s",
			cluster.ClusterNetworks[0].Cidr, cluster.ServiceNetworks[0].Cidr, network.GetMachineCidrById(cluster, 0))))
	})
	It("All-excluded NoProxy", func() {
		cluster.NoProxy = "*"
		noProxy := installConfig.generateNoProxy(cluster)
		Expect(noProxy).Should(Equal("*"))
	})
	It("All-excluded NoProxy with spaces", func() {
		cluster.NoProxy = " * "
		noProxy := installConfig.generateNoProxy(cluster)
		Expect(noProxy).Should(Equal("*"))
	})
})

func TestBuilder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "installcfg tests")
}
