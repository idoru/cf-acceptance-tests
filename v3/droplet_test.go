package v3

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"
)

var _ = Describe("droplet features", func() {

	var (
		appGuid     string
		appName     string
		packageGuid string
		spaceGuid   string
		token       string
	)

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")
		spaceGuid = GetSpaceGuidFromName(context.RegularUserContext().Space)
		appGuid = CreateApp(appName, spaceGuid, "{}")
		packageGuid = CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl := fmt.Sprintf("%s%s/v3/packages/%s/upload", config.Protocol(), config.ApiEndpoint, packageGuid)
		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)
	})

	Context("copying a droplet", func() {
		var (
			destinationAppGuid string
			destinationAppName string
			sourceDropletGuid  string
		)

		BeforeEach(func() {
			sourceDropletGuid = StageBuildpackPackage(packageGuid, "ruby_buildpack")
			WaitForDropletToStage(sourceDropletGuid)

			destinationAppName = generator.PrefixedRandomName("CATS-APP-")
			destinationAppGuid = CreateApp(destinationAppName, spaceGuid, "{}")
		})

		It("can copy a droplet", func() {
			copyRequestBody := fmt.Sprintf("{\"relationships\":{\"app\":{\"guid\":\"%s\"}}}", destinationAppGuid)
			copyUrl := fmt.Sprintf("/v3/droplets/%s/copy", sourceDropletGuid)
			session := cf.Cf("curl", copyUrl, "-X", "POST", "-d", copyRequestBody)

			bytes := session.Wait(DEFAULT_TIMEOUT).Out.Contents()
			var droplet struct {
				Guid string `json:"guid"`
			}
			json.Unmarshal(bytes, &droplet)
			copiedDropletGuid := droplet.Guid

			WaitForDropletToStage(copiedDropletGuid)

			DeleteApp(appGuid) // to prove that the new app does not depend on the old app

			AssignDropletToApp(destinationAppGuid, copiedDropletGuid)

			processes := GetProcesses(destinationAppGuid, destinationAppName)
			webProcess := GetProcessByType(processes, "web")
			workerProcess := GetProcessByType(processes, "worker")

			Expect(webProcess.Guid).ToNot(BeEmpty())
			Expect(workerProcess.Guid).ToNot(BeEmpty())

			CreateAndMapRoute(destinationAppGuid, context.RegularUserContext().Space, helpers.LoadConfig().AppsDomain, webProcess.Name)
			StartApp(destinationAppGuid)
			Eventually(func() string {
				return helpers.CurlAppRoot(webProcess.Name)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
		})

	})

})
