package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/client"
	"github.com/openshift/assisted-service/client/installer"
	"github.com/openshift/assisted-service/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func TestValidator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "inventory_test")
}

var _ = Describe("inventory_client_tests", func() {
	var (
		client     *client.AssistedInstall
		server     *ghttp.Server
		params     *installer.V2UpdateHostInstallProgressParams
		infraEnvID = "infra-env-id"
		hostId     = "host-id"
	)

	AfterEach(func() {
		server.Close()
	})

	BeforeEach(func() {
		maxDelay = time.Duration(1) * time.Second
		//testRetryMaxDelay = time.Duration(10) * time.Second
		retries = 4
		var err error
		server = ghttp.NewUnstartedServer()
		server.SetAllowUnhandledRequests(true)
		server.SetUnhandledRequestStatusCode(http.StatusInternalServerError) // 500
		agentConfig := &config.AgentConfig{}
		params = &installer.V2UpdateHostInstallProgressParams{
			InfraEnvID: strfmt.UUID(infraEnvID),
			HostID:     strfmt.UUID(hostId),
			HostProgress: &models.HostProgress{
				CurrentStage: "Installing",
			},
		}
		client, err = createBmInventoryClient(agentConfig, "http://"+server.Addr(), "pullSecret")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(client).ShouldNot(BeNil())
	})

	Context("UpdateHostInstallProgress", func() {
		var (
			expectedJson = make(map[string]string)
		)

		BeforeEach(func() {
			expectedJson["current_stage"] = "Installing"
		})

		It("positive_response", func() {
			server.Start()
			expectServerCall(server, fmt.Sprintf("/api/assisted-install/v2/infra-envs/%s/hosts/%s/progress", infraEnvID, hostId), expectedJson, http.StatusOK)
			_, err := client.Installer.V2UpdateHostInstallProgress(context.Background(), params)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
		})

		It("negative_server_error_response no retry on service error", func() {
			server.Start()
			_, err := client.Installer.V2UpdateHostInstallProgress(context.Background(), params)
			Expect(err).Should(HaveOccurred())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))

		})
		It("server_partially available - retry should reconnect", func() {
			go func() {
				time.Sleep(maxDelay * 2)
				expectServerCall(server, fmt.Sprintf("/api/assisted-install/v2/infra-envs/%s/hosts/%s/progress", infraEnvID, hostId), expectedJson, http.StatusOK)
				server.Start()
			}()

			_, err := client.Installer.V2UpdateHostInstallProgress(context.Background(), params)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
		})
		It("server_down", func() {
			server.Start()
			server.Close()
			_, err := client.Installer.V2UpdateHostInstallProgress(context.Background(), params)
			Expect(err).Should(HaveOccurred())
		})
	})
})

func expectServerCall(server *ghttp.Server, path string, expectedJson interface{}, returnedStatusCode int) {
	var body = []byte("empty")
	data, err := json.Marshal(expectedJson)
	Expect(err).ShouldNot(HaveOccurred())
	content := string(data) + "\n"

	server.AppendHandlers(
		ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path),
			ghttp.VerifyJSON(string(data)),
			ghttp.VerifyHeader(http.Header{"Content-Length": []string{strconv.Itoa(len(content))}}),
			ghttp.RespondWithPtr(&returnedStatusCode, &body),
		),
	)
}
