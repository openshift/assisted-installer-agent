package subsystem

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

const (
	WireMockURL = "http://localhost:8080"
)

var (
	nextHostIndex = 0
	RequestsURL   = fmt.Sprintf("%s/__admin/requests", WireMockURL)
	MappingsURL   = fmt.Sprintf("%s/__admin/mappings", WireMockURL)
)

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
	case http.StatusForbidden:
		errorReply := &models.InfraError{
			Code:   swag.Int32(http.StatusForbidden),
			Message: swag.String(fmt.Sprintf("%d", reply)),
		}
		b, err = json.Marshal(errorReply)
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

func startAgent(args ...string) error {
	return startContainer("agent")
}

func stopAgent() error {
	return stopContainer("agent")
}

func startContainer(args ...string) error {
	args = append([]string{"-f", "docker-compose.yml", "run", "-d"}, args...)

	cmd := exec.Command("docker-compose", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func stopContainer(name string) error {
	cmd := exec.Command("docker-compose", "-f", "docker-compose.yml", "rm", "-s", "-f", name)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func resetAll() {
	Expect(resetRequests()).NotTo(HaveOccurred())
	Expect(deleteAllStubs()).NotTo(HaveOccurred())
	Expect(stopAgent()).NotTo(HaveOccurred())
}

func nextHostID() string {
	hostID := fmt.Sprintf("00000000-0000-0000-0000-0000000000%02x", nextHostIndex)
	nextHostIndex++
	return hostID
}

func waitForWiremock() error {
	_, err := http.Get(RequestsURL)
	return err
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
