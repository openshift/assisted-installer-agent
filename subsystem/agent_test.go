package subsystem

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

const (
	ClusterID   = "11111111-1111-1111-1111-111111111111"
	WireMockURL = "http://wiremock:8080"
)

var (
	nextHostIndex = 0
	RequestsURL   = fmt.Sprintf("%s/__admin/requests", WireMockURL)
	MappingsURL   = fmt.Sprintf("%s/__admin/mappings", WireMockURL)
)

var _ = Describe("Agent tests", func() {
	defaultnextInstructionSeconds := int64(1)
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
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("register forbidden", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID, http.StatusForbidden)
		Expect(err).NotTo(HaveOccurred())

		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds)
		Expect(err).NotTo(HaveOccurred())

		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(10 * time.Second)

		// validate only register request was called
		resp, err := http.Get(RequestsURL)
		Expect(err).ShouldNot(HaveOccurred())
		requests := &Requests{}
		b, err := ioutil.ReadAll(resp.Body)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(json.Unmarshal(b, &requests)).ShouldNot(HaveOccurred())
		req := make([]*RequestOccurence, 0, len(requests.Requests))
		for _, r := range requests.Requests {
			req = append(req, r)
		}
		Expect(len(req)).Should(Equal(1))
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("register not found - agent should stop trying to register", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID, http.StatusNotFound)
		Expect(err).NotTo(HaveOccurred())

		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds)
		Expect(err).NotTo(HaveOccurred())

		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(10 * time.Second)

		// validate only register request was called
		resp, err := http.Get(RequestsURL)
		Expect(err).ShouldNot(HaveOccurred())
		requests := &Requests{}
		b, err := ioutil.ReadAll(resp.Body)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(json.Unmarshal(b, &requests)).ShouldNot(HaveOccurred())
		req := make([]*RequestOccurence, 0, len(requests.Requests))
		for _, r := range requests.Requests {
			req = append(req, r)
		}
		Expect(len(req)).Should(Equal(1))
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Verify nextInstructionSeconds", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(5 * time.Second)
		verifyRegisterRequest()
		verifyNumberOfGetNextRequest(hostID, ">", 3)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())

		By("verify changing nextInstructionSeconds to large number")
		hostID = nextHostID()
		registerStubID, err = addRegisterStub(hostID, http.StatusCreated)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err = addNextStepStub(hostID, 100)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(30 * time.Second)
		verifyRegisterRequest()
		verifyNumberOfGetNextRequest(hostID, "<", 2)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Cluster not exists", func() {
		hostID := nextHostID()
		registerStubID, err := addNextStepClusterNotExistsStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(10 * time.Second)
		verifyRegisterRequest()
		verifyNumberOfGetNextRequest(hostID, "<", 2)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Register recovery", func() {
		hostID := nextHostID()
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, false)
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated)
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
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated)
		Expect(err).NotTo(HaveOccurred())
		stepID := "wrong-step"
		stepType := models.StepType("Step-not-exists")
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, &models.Step{StepType: stepType, StepID: stepID, Args: make([]string, 0)})
		Expect(err).NotTo(HaveOccurred())
		replyStubID, err := addStepReplyStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		expectedReply := &EqualReplyVerifier{
			Error:    "Missing command",
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
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated)
		Expect(err).NotTo(HaveOccurred())
		stepID := "execute-step"
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, &models.Step{
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
	It("Multiple steps", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds,
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
				StepType: models.StepTypeInventory,
				StepID:   "inventory-step",
				Command:  "docker",
				Args: []string{
					"run", "--privileged", "--net=host", "--rm",
					"-v", "/var/log:/var/log",
					"-v", "/run/udev:/run/udev",
					"-v", "/dev/disk:/dev/disk",
					"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
					"quay.io/ocpmetal/assisted-installer-agent:latest",
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
		verifyStepReplyRequest(hostID, &InventoryVerifier{})
		stepReply := getSpecificStep(hostID, &InventoryVerifier{})
		inventory := getInventoryFromStepReply(stepReply)
		Expect(len(inventory.Interfaces) > 0).To(BeTrue())
		freeAddressesRequest := models.FreeAddressesRequest{}
		for _, intf := range inventory.Interfaces {
			for _, ipAddr := range intf.IPV4Addresses {
				ip, cidr, err := net.ParseCIDR(ipAddr)
				Expect(err).ToNot(HaveOccurred())
				ones, _ := cidr.Mask.Size()
				if ones < 24 {
					_, cidr, err = net.ParseCIDR(ip.To4().String() + "/24")
					Expect(err).ToNot(HaveOccurred())
				}
				freeAddressesRequest = append(freeAddressesRequest, cidr.String())
			}
		}
		Expect(len(freeAddressesRequest)).ToNot(BeZero())
		b, err := json.Marshal(&freeAddressesRequest)
		Expect(err).ToNot(HaveOccurred())
		err = deleteStub(nextStepsStubID)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err = addNextStepStub(hostID, defaultnextInstructionSeconds,
			&models.Step{
				StepType: models.StepTypeFreeNetworkAddresses,
				StepID:   "free-addresses",
				Command:  "docker",
				Args: []string{"run", "--privileged", "--net=host", "--rm",
					"-v", "/var/log:/var/log",
					"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
					"quay.io/ocpmetal/assisted-installer-agent:latest",
					"free_addresses",
					string(b),
				},
			},
		)
		Eventually(func() bool {
			return isReplyFound(hostID, &FreeAddressesVerifier{})
		}, 300*time.Second, 5*time.Second).Should(BeTrue())
		err = deleteStub(registerStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(nextStepsStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(replyStubID)
		Expect(err).NotTo(HaveOccurred())
	})
})

type RequestDefinition struct {
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	BodyPatterns []interface{}     `json:"bodyPatterns"`
	Headers      map[string]string `json:"headers"`
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
	URL     string
	Method  string
	Body    string
	Headers map[string]string
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
	v, ok = reqs[0].Request.Headers["X-Secret-Key"]
	Expect(ok).Should(BeTrue())
	Expect(v).Should(Equal("OpenShiftToken"))
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

func verifyNumberOfGetNextRequest(hostID string, comaparator string, number int) {
	reqs, err := findAllMatchingRequests(getNextStepsURL(hostID), "GET")
	Expect(err).NotTo(HaveOccurred())
	Expect(len(reqs)).Should(BeNumerically(comaparator, number))
}

type StepVerifier interface {
	verify(actualReply *models.StepReply) bool
}

type EqualReplyVerifier models.StepReply

func (e *EqualReplyVerifier) verify(actualReply *models.StepReply) bool {
	return *(*models.StepReply)(e) == *actualReply
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

type FreeAddressesVerifier struct{}

func (f *FreeAddressesVerifier) verify(actualReply *models.StepReply) bool {
	if actualReply.StepType != models.StepTypeFreeNetworkAddresses {
		return false
	}
	Expect(actualReply.ExitCode).To(BeZero())
	var freeAddresses models.FreeNetworksAddresses
	Expect(json.Unmarshal([]byte(actualReply.Output), &freeAddresses)).ToNot(HaveOccurred())
	Expect(len(freeAddresses) > 0).To(BeTrue())
	_, _, err := net.ParseCIDR(freeAddresses[0].Network)
	Expect(err).ToNot(HaveOccurred())
	if len(freeAddresses[0].FreeAddresses) > 0 {
		ip := net.ParseIP(freeAddresses[0].FreeAddresses[0].String())
		Expect(ip).ToNot(BeNil())
		Expect(ip.To4()).ToNot(BeNil())
	}
	return true
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

func isReplyFound(hostID string, verifier StepVerifier) bool {
	reqs, err := findAllMatchingRequests(getStepReplyURL(hostID), "POST")
	Expect(err).NotTo(HaveOccurred())
	for _, r := range reqs {
		var actualReply models.StepReply
		Expect(json.Unmarshal([]byte(r.Request.Body), &actualReply)).NotTo(HaveOccurred())
		if verifier.verify(&actualReply) {
			return true
		}
	}
	return false
}

func getInventoryFromStepReply(actualReply *models.StepReply) *models.Inventory {
	var inventory models.Inventory
	err := json.Unmarshal([]byte(actualReply.Output), &inventory)
	Expect(err).NotTo(HaveOccurred())
	return &inventory
}

func getSpecificStep(hostID string, verifier StepVerifier) *models.StepReply {
	reqs, err := findAllMatchingRequests(getStepReplyURL(hostID), "POST")
	Expect(err).NotTo(HaveOccurred())
	for _, r := range reqs {
		var actualReply models.StepReply
		Expect(json.Unmarshal([]byte(r.Request.Body), &actualReply)).NotTo(HaveOccurred())
		if verifier.verify(&actualReply) {
			return &actualReply
		}
	}
	ExpectWithOffset(1, true).Should(BeFalse(), "Expected step not found")
	return nil
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
	resp, err := http.Post(MappingsURL, "application/json", &b)
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

func addRegisterStub(hostID string, reply int) (string, error) {
	var b []byte
	var err error
	hostUUID := strfmt.UUID(hostID)
	hostKind := "host"

	switch reply {
	case http.StatusCreated:
		returnedHost := &models.Host{
			ID:   &hostUUID,
			Kind: &hostKind,
		}
		b, err = json.Marshal(&returnedHost)
		if err != nil {
			return "", err
		}
	default:
		errorReply := &models.Error{
			Code:   swag.String(fmt.Sprintf("%d", reply)),
			Href:   swag.String(""),
			ID:     swag.Int32(int32(reply)),
			Kind:   swag.String("Error"),
			Reason: swag.String(fmt.Sprintf("%d", reply)),
		}
		b, err = json.Marshal(errorReply)
		if err != nil {
			return "", err
		}
	}
	stub := StubDefinition{
		Request: &RequestDefinition{
			URL:    getRegisterURL(),
			Method: "POST",
		},
		Response: &ResponseDefinition{
			Status: reply,
			Body:   string(b),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	return addStub(&stub)
}

func addNextStepStub(hostID string, nextInstructionSeconds int64, instructions ...*models.Step) (string, error) {
	if instructions == nil {
		instructions = make([]*models.Step, 0)
	}
	steps := models.Steps{NextInstructionSeconds: nextInstructionSeconds, Instructions: instructions}
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

func addNextStepClusterNotExistsStub(hostID string, instructions ...*models.Step) (string, error) {
	if instructions == nil {
		instructions = make([]*models.Step, 0)
	}
	steps := models.Steps{NextInstructionSeconds: 1, Instructions: instructions}
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
			Status: 404,
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
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/%s", MappingsURL, stubID), nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	_, err = client.Do(req)
	return err
}

func deleteAllStubs() error {
	req, err := http.NewRequest("DELETE", MappingsURL, nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	_, err = client.Do(req)
	return err
}

func findAllMatchingRequests(url, method string) ([]*RequestOccurence, error) {
	resp, err := http.Get(RequestsURL)
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
	req, err := http.NewRequest("DELETE", RequestsURL, nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	_, err = client.Do(req)
	return err
}

func startAgent() error {
	cmd := exec.Command("docker-compose", "-f", "docker-compose.yml", "start", "agent")
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func stopAgent() error {
	cmd := exec.Command("docker-compose", "-f", "docker-compose.yml", "stop", "agent")
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func nextHostID() string {
	hostID := fmt.Sprintf("00000000-0000-0000-0000-0000000000%02x", nextHostIndex)
	nextHostIndex++
	return hostID
}

func waitForWiremock() error {
	_, err := http.Get("http://wiremock:8080/__admin/requests")
	return err
}

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subsystem Suite")
}
