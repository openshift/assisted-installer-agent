package subsystem

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/filanov/bm-inventory/models"
	"github.com/go-openapi/strfmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	ClusterID = "11111111-1111-1111-1111-111111111111"
)

var (
	nextHostIndex = 0
)

var _ = Describe("Agent tests", func() {
	BeforeSuite(func() {
		Eventually(waitForWiremock).ShouldNot(HaveOccurred())
	})

	BeforeEach(func() {
		Expect(stopAgent()).NotTo(HaveOccurred())
	})

	JustBeforeEach(func() {
		Expect(resetRequests()).NotTo(HaveOccurred())
		Expect(deleteAllStubs()).NotTo(HaveOccurred())
	})

	It("Happy flow", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Register recovery", func() {
		hostID := nextHostID()
		nextStepsStubID, err := addNextStepStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, false)
		registerStubID, err := addRegisterStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 6)
		verifyRegisterRequest()
		verifyRegistersSameID()
		verifyGetNextRequest(hostID, true)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Step not exists", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		stepID := "wrong-step"
		stepType := models.StepType("Step-not-exists")
		nextStepsStubID, err := addNextStepStub(hostID, &models.Step{StepType: stepType, StepID: stepID, Args: make([]string, 0)})
		Expect(err).NotTo(HaveOccurred())
		replyStubID, err := addStepReplyStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		expectedReply := &EqualReplyVerifier{
			Error:    fmt.Sprintf("Unexpected step type: %s", stepType),
			ExitCode: -1,
			Output:   "",
			StepID:   stepID,
			StepType: stepType,
		}
		verifyStepReplyRequest(hostID, expectedReply)
		err = deleteStub(registerStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(nextStepsStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(replyStubID)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Execute echo", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		stepID := "execute-step"
		nextStepsStubID, err := addNextStepStub(hostID, &models.Step{
			StepType: models.StepTypeExecute,
			StepID:   stepID,
			Command:  "echo",
			Args: []string{
				"Hello",
				"world",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		replyStubID, err := addStepReplyStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		expectedReply := &EqualReplyVerifier{
			Error:    "",
			ExitCode: 0,
			Output:   "Hello world\n",
			StepID:   stepID,
			StepType: models.StepTypeExecute,
		}
		verifyStepReplyRequest(hostID, expectedReply)
		err = deleteStub(registerStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(nextStepsStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(replyStubID)
		Expect(err).NotTo(HaveOccurred())
	})
	It("Hardware info", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		stepID := "hardware-info-step"
		nextStepsStubID, err := addNextStepStub(hostID, &models.Step{
			StepType: models.StepTypeHardwareInfo,
			StepID:   stepID,
			Command:  "docker",
			Args:     strings.Split("run,--rm,--privileged,--net=host,-v,/var/log:/var/log,quay.io/ocpmetal/hardware_info:latest,/usr/bin/hardware_info", ","),
		})
		Expect(err).NotTo(HaveOccurred())
		replyStubID, err := addStepReplyStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		verifyStepReplyRequest(hostID, &HardwareInfoVerifier{})
		err = deleteStub(registerStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(nextStepsStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(replyStubID)
		Expect(err).NotTo(HaveOccurred())
	})
	It("Multiple steps backward compatible", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID,
			&models.Step{
				StepType: models.StepTypeHardwareInfo,
				StepID:   "hardware-info-step",
			},
			&models.Step{
				StepType: models.StepTypeInventory,
				StepID:   "inventory-step",
			},
		)
		Expect(err).NotTo(HaveOccurred())
		replyStubID, err := addStepReplyStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(5 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		verifyStepReplyRequest(hostID, &HardwareInfoVerifier{})
		verifyStepReplyRequest(hostID, &InventoryVerifier{})
		err = deleteStub(registerStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(nextStepsStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(replyStubID)
		Expect(err).NotTo(HaveOccurred())
	})
	It("Multiple steps", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID,
			&models.Step{
				StepType: models.StepTypeExecute,
				StepID:   "echo-step-1",
				Command:  "echo",
				Args: []string{
					"Hello",
					"world",
				},
			},
			&models.Step{
				StepType: models.StepTypeExecute,
				StepID:   "echo-step-2",
				Command:  "echo",
				Args: []string{
					"Bye",
					"bye",
					"world",
				},
			},
			&models.Step{
				StepType: models.StepTypeHardwareInfo,
				StepID:   "hardware-info-step",
				Command:  "docker",
				Args:     strings.Split("run,--rm,--privileged,--net=host,-v,/var/log:/var/log,quay.io/ocpmetal/hardware_info:latest,/usr/bin/hardware_info", ","),
			},
			&models.Step{
				StepType: models.StepTypeInventory,
				StepID:   "inventory-step",
				Command:  "docker",
				Args: []string{
					"run", "--privileged", "--net=host", "--rm",
					"-v", "/var/log:/var/log",
					"-v", "/run/udev:/run/udev",
					"-v", "/dev/disk:/dev/disk",
					"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
					"quay.io/ocpmetal/inventory:latest",
					"inventory",
				},
			},
		)
		Expect(err).NotTo(HaveOccurred())
		replyStubID, err := addStepReplyStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(5 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		verifyStepReplyRequest(hostID, &EqualReplyVerifier{
			Error:    "",
			ExitCode: 0,
			Output:   "Hello world\n",
			StepID:   "echo-step-1",
			StepType: models.StepTypeExecute,
		})
		verifyStepReplyRequest(hostID, &EqualReplyVerifier{
			Error:    "",
			ExitCode: 0,
			Output:   "Bye bye world\n",
			StepID:   "echo-step-2",
			StepType: models.StepTypeExecute,
		})
		verifyStepReplyRequest(hostID, &HardwareInfoVerifier{})
		verifyStepReplyRequest(hostID, &InventoryVerifier{})
		err = deleteStub(registerStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(nextStepsStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(replyStubID)
		Expect(err).NotTo(HaveOccurred())
	})
})

type EqualToJsonDefinition struct {
	EqualToJson string `json:"equalToJson"`
}

type RequestDefinition struct {
	URL          string        `json:"url"`
	Method       string        `json:"method"`
	BodyPatterns []interface{} `json:"bodyPatterns"`
}

type ResponseDefinition struct {
	Status  int               `json:"status"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
}

type StubDefinition struct {
	Request  *RequestDefinition  `json:"request"`
	Response *ResponseDefinition `json:"response"`
}

type ReceivedRequest struct {
	URL    string
	Method string
	Body   string
}

type ReceivedResponse struct {
	Status  int
	Body    string
	Headers map[string]string
}

type RequestOccurence struct {
	ID         string
	Request    *ReceivedRequest
	Response   *ReceivedResponse
	WasMatched bool
}

type Mapping struct {
	ID string
}

type Requests struct {
	Requests []*RequestOccurence
}

func jsonToMap(str string) map[string]interface{} {
	m := make(map[string]interface{})
	Expect(json.Unmarshal([]byte(str), &m)).ShouldNot(HaveOccurred())
	return m
}

func verifyRegisterRequest() {
	reqs, err := findAllMatchingRequests(getRegisterURL(), "POST")
	Expect(err).NotTo(HaveOccurred())
	Expect(len(reqs)).Should(BeNumerically(">", 0))
	m := jsonToMap(reqs[0].Request.Body)
	v, ok := m["host_id"]
	Expect(ok).Should(BeTrue())
	Expect(v).Should(MatchRegexp("[0-9a-f]{8}-?(?:[0-9a-f]{4}-?){3}[0-9a-f]{12}"))
}

func verifyRegistersSameID() {
	reqs, err := findAllMatchingRequests(getRegisterURL(), "POST")
	Expect(err).NotTo(HaveOccurred())
	Expect(len(reqs)).Should(BeNumerically(">", 1))
	m1 := jsonToMap(reqs[0].Request.Body)
	m2 := jsonToMap(reqs[1].Request.Body)
	host1ID, ok1 := m1["host_id"]
	host2ID, ok2 := m2["host_id"]
	Expect(ok1).Should(BeTrue())
	Expect(ok2).Should(BeTrue())
	Expect(host1ID).Should(Equal(host2ID))
}
func verifyGetNextRequest(hostID string, matchExpected bool) {
	reqs, err := findAllMatchingRequests(getNextStepsURL(hostID), "GET")
	Expect(err).NotTo(HaveOccurred())
	if !matchExpected {
		Expect(reqs).To(BeEmpty())
	} else {
		Expect(len(reqs)).Should(BeNumerically(">", 0))
	}
}

type StepVerifier interface {
	verify(actualReply *models.StepReply) bool
}

type EqualReplyVerifier models.StepReply

func (e *EqualReplyVerifier) verify(actualReply *models.StepReply) bool {
	return *(*models.StepReply)(e) == *actualReply
}

type HardwareInfoVerifier struct{}

func (h *HardwareInfoVerifier) verify(actualReply *models.StepReply) bool {
	if actualReply.ExitCode != 0 {
		return false
	}
	var hardwareInfo models.Introspection
	err := json.Unmarshal([]byte(actualReply.Output), &hardwareInfo)
	if err != nil {
		return false
	}
	return len(hardwareInfo.Memory) > 0 && hardwareInfo.CPU != nil && hardwareInfo.CPU.Cpus > 0 && len(hardwareInfo.BlockDevices) > 0 && len(hardwareInfo.Nics) > 0
}

type InventoryVerifier struct{}

func (i *InventoryVerifier) verify(actualReply *models.StepReply) bool {
	if actualReply.ExitCode != 0 {
		return false
	}
	if actualReply.StepType != models.StepTypeInventory {
		return false
	}
	var inventory models.Inventory
	err := json.Unmarshal([]byte(actualReply.Output), &inventory)
	if err != nil {
		return false
	}
	return inventory.Memory != nil && inventory.Memory.UsableBytes > 0 && inventory.Memory.PhysicalBytes > 0 &&
		inventory.CPU != nil && inventory.CPU.Count > 0 &&
		len(inventory.Disks) > 0 &&
		len(inventory.Interfaces) > 0 &&
		inventory.Hostname != ""
}

func verifyStepReplyRequest(hostID string, verifier StepVerifier) {
	reqs, err := findAllMatchingRequests(getStepReplyURL(hostID), "POST")
	Expect(err).NotTo(HaveOccurred())
	for _, r := range reqs {
		var actualReply models.StepReply
		Expect(json.Unmarshal([]byte(r.Request.Body), &actualReply)).NotTo(HaveOccurred())
		if verifier.verify(&actualReply) {
			return
		}
	}
	ExpectWithOffset(1, true).Should(BeFalse(), "Expected step not found")
}

func getRegisterURL() string {
	return fmt.Sprintf("/api/assisted-install/v1/clusters/%s/hosts", ClusterID)
}

func getNextStepsURL(hostID string) string {
	return fmt.Sprintf("/api/assisted-install/v1/clusters/%s/hosts/%s/instructions", ClusterID, hostID)
}

func getStepReplyURL(hostID string) string {
	return fmt.Sprintf("/api/assisted-install/v1/clusters/%s/hosts/%s/instructions", ClusterID, hostID)
}

func addStub(stub *StubDefinition) (string, error) {
	requestBody, err := json.Marshal(stub)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	b.Write(requestBody)
	resp, err := http.Post("http://127.0.0.1:8080/__admin/mappings", "application/json", &b)
	if err != nil {
		return "", err
	}
	responseBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}
	ret := Mapping{}
	err = json.Unmarshal(responseBody, &ret)
	if err != nil {
		return "", err
	}
	return ret.ID, nil
}

func addRegisterStub(hostID string) (string, error) {
	hostUUID := strfmt.UUID(hostID)
	hostKind := "host"
	returnedHost := &models.Host{
		ID:   &hostUUID,
		Kind: &hostKind,
	}
	b, err := json.Marshal(&returnedHost)
	if err != nil {
		return "", err
	}
	stub := StubDefinition{
		Request: &RequestDefinition{
			URL:    getRegisterURL(),
			Method: "POST",
		},
		Response: &ResponseDefinition{
			Status: 201,
			Body:   string(b),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	return addStub(&stub)
}

func addNextStepStub(hostID string, steps ...*models.Step) (string, error) {
	if steps == nil {
		steps = make([]*models.Step, 0)
	}
	b, err := json.Marshal(steps)
	if err != nil {
		return "", err
	}
	stub := StubDefinition{
		Request: &RequestDefinition{
			URL:    getNextStepsURL(hostID),
			Method: "GET",
		},
		Response: &ResponseDefinition{
			Status: 200,
			Body:   string(b),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}
	return addStub(&stub)
}

func addStepReplyStub(hostID string) (string, error) {
	stub := StubDefinition{
		Request: &RequestDefinition{
			URL:    getStepReplyURL(hostID),
			Method: "POST",
		},
		Response: &ResponseDefinition{
			Status: 204,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}
	return addStub(&stub)

}

func deleteStub(stubID string) error {
	req, err := http.NewRequest("DELETE", "http://127.0.0.1:8080/__admin/mappings/"+stubID, nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	_, err = client.Do(req)
	return err
}

func deleteAllStubs() error {
	req, err := http.NewRequest("DELETE", "http://127.0.0.1:8080/__admin/mappings", nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	_, err = client.Do(req)
	return err
}

func findAllMatchingRequests(url, method string) ([]*RequestOccurence, error) {
	resp, err := http.Get("http://127.0.0.1:8080/__admin/requests")
	if err != nil {
		return nil, err
	}
	requests := &Requests{}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &requests)
	if err != nil {
		return nil, err
	}
	ret := make([]*RequestOccurence, 0)
	for _, r := range requests.Requests {
		if r.Request.URL == url && r.Request.Method == method {
			ret = append(ret, r)
		}
	}
	return ret, nil
}

func resetRequests() error {
	req, err := http.NewRequest("DELETE", "http://127.0.0.1:8080/__admin/requests", nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	_, err = client.Do(req)
	return err
}

func startAgent() error {
	cmd := exec.Command("docker", "start", "agent_container")
	return cmd.Run()
}

func stopAgent() error {
	cmd := exec.Command("docker", "stop", "agent_container")
	return cmd.Run()
}

func nextHostID() string {
	hostID := fmt.Sprintf("00000000-0000-0000-0000-0000000000%02x", nextHostIndex)
	nextHostIndex++
	return hostID
}

func waitForWiremock() error {
	_, err := http.Get("http://127.0.0.1:8080/__admin/requests")
	return err
}

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subsystem Suite")
}
