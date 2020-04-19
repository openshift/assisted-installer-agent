package subsystem

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/filanov/bm-inventory/models"
	"github.com/go-openapi/strfmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http"
	"os/exec"
	"testing"
	"time"
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
		verifyGetNextRequest(hostID, true)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Step not exists", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		stepID := "wrong-step"
		stepType := "Step-not-exists"
		nextStepsStubID, err := addNextStepStub(hostID, &models.Step{StepType:models.StepType(stepType), StepID:stepID, Args:make([]string, 0)})
		Expect(err).NotTo(HaveOccurred())
		replyStubID, err := addStepReplyStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		expectedReply := &EqualReply{
			Error:    fmt.Sprintf("Unexpected step type: %s", stepType),
			ExitCode: -1,
			Output:   "",
			StepID:   stepID,
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
			StepType:models.StepTypeExecute,
			StepID:stepID,
			Command: "echo",
			Args:[]string {
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
		expectedReply := &EqualReply{
			Error:    "",
			ExitCode: 0,
			Output:   "Hello world\n",
			StepID:   stepID,
		}
		verifyStepReplyRequest(hostID, expectedReply)
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
	URL string  `json:"url"`
	Method string `json:"method"`
	BodyPatterns []interface{} `json:"bodyPatterns"`
}

type ResponseDefinition struct {
	Status int  `json:"status"`
	Body string  `json:"body"`
	Headers map[string]string `json:"headers"`
}

type StubDefinition struct {
	Request *RequestDefinition `json:"request"`
	Response *ResponseDefinition `json:"response"`
}

type ReceivedRequest struct {
	URL string
	Method string
	Body string
}

type ReceivedResponse struct {
	Status int
	Body string
	Headers map[string]string
}


type RequestOccurence struct {
	ID string
	Request *ReceivedRequest
	Response *ReceivedResponse
	WasMatched bool
}

type Mapping struct {
	ID string
}

type Requests struct {
	Requests []*RequestOccurence
}

func verifyRegisterRequest() {
	reqs, err := findAllMatchingRequests(getRegisterURL(), "POST")
	Expect(err).NotTo(HaveOccurred())
	Expect(len(reqs)).Should(BeNumerically(">", 0))
	foundReq := reqs[0]
	m := make(map[string]interface{})
	Expect(json.Unmarshal([]byte(foundReq.Request.Body), &m)).ShouldNot(HaveOccurred())
	v, ok := m["hostId"]
	Expect(ok).Should(BeTrue())
	Expect(v).Should(MatchRegexp("[0-9a-f]{8}-?(?:[0-9a-f]{4}-?){3}[0-9a-f]{12}"))
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

type EqualReply models.StepReply

func (e *EqualReply) verify(actualReply *models.StepReply) bool {
	return *(*models.StepReply)(e) == *actualReply
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
	Expect(true).Should(BeFalse(), "Expected step not found")
}

func getRegisterURL() string {
	return fmt.Sprintf("/api/bm-inventory/v1/clusters/%s/hosts", ClusterID)
}

func getNextStepsURL(hostID string) string {
	return fmt.Sprintf("/api/bm-inventory/v1/clusters/%s/hosts/%s/instructions", ClusterID, hostID)
}

func getStepReplyURL(hostID string) string {
	return fmt.Sprintf("/api/bm-inventory/v1/clusters/%s/hosts/%s/instructions", ClusterID, hostID)
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
	responseBody,err := ioutil.ReadAll(resp.Body)

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
		Base:             models.Base{
			ID:   &hostUUID,
			Kind: &hostKind,
		},
	}
	b, err := json.Marshal(&returnedHost)
	if err != nil {
		return "", err
	}
	stub := StubDefinition{
		Request:  &RequestDefinition{
			URL: getRegisterURL(),
			Method: "POST",
		},
		Response: &ResponseDefinition{
			Status:  201,
			Body:    string(b),
			Headers: map[string] string {
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
		Request:  &RequestDefinition{
			URL: getNextStepsURL(hostID),
			Method: "GET",
		},
		Response: &ResponseDefinition{
			Status:  200,
			Body:    string(b),
			Headers: map[string] string {
				"Content-Type": "application/json",
			},
		},
	}
	return addStub(&stub)
}

func addStepReplyStub(hostID string) (string, error) {
	stub := StubDefinition{
		Request:  &RequestDefinition{
			URL: getStepReplyURL(hostID),
			Method: "POST",
		},
		Response: &ResponseDefinition{
			Status:  204,
			Headers: map[string] string {
				"Content-Type": "application/json",
			},
		},
	}
	return addStub(&stub)

}

func deleteStub(stubID string) error {
	req, err := http.NewRequest("DELETE", "http://127.0.0.1:8080/__admin/mappings/" + stubID, nil)
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

func findRequest(url, method string) (found bool, matched bool, err error) {
	resp,err := http.Get("http://127.0.0.1:8080/__admin/requests")
	if err != nil {
		return false, false, err
	}
	requests := &Requests{}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, false, err
	}
	err = json.Unmarshal(b , &requests)
	if err != nil {
		return false, false, err
	}
	for _, r := range requests.Requests {
		if r.Request.URL == url && r.Request.Method == method {
			return true, r.WasMatched, nil
		}
	}
	return false, false, nil
}

func findAllMatchingRequests(url, method string) ([]*RequestOccurence, error) {
	resp,err := http.Get("http://127.0.0.1:8080/__admin/requests")
	if err != nil {
		return nil, err
	}
	requests := &Requests{}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b , &requests)
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


func resetRequests() error{
	req, err := http.NewRequest("DELETE", "http://127.0.0.1:8080/__admin/requests", nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	_ , err = client.Do(req)
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

func waitForWiremock() error{
	_,err := http.Get("http://127.0.0.1:8080/__admin/requests")
	return err
}

func TestSubsystem(t *testing.T) {
        RegisterFailHandler(Fail)
        RunSpecs(t, "Subsystem Suite")
}


