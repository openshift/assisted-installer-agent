package subsystem

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var (
	nextHostIndex                = 0
	WireMockURLFromSubsystemHost = fmt.Sprintf("http://127.0.0.1:%s", os.Getenv("WIREMOCK_PORT"))
	AssistedServiceURLFromAgent  = fmt.Sprintf("http://wiremock:%s", os.Getenv("WIREMOCK_PORT"))
	RequestsURL                  = fmt.Sprintf("%s/__admin/requests", WireMockURLFromSubsystemHost)
	MappingsURL                  = fmt.Sprintf("%s/__admin/mappings", WireMockURLFromSubsystemHost)
	agentImage                   = os.Getenv("ASSISTED_INSTALLER_AGENT")
	defaultContainerTool         = "docker"
)

const (
	maxTimeout       = 300 * time.Second
	agentServiceName = "agent"
	infraEnvID        = "11111111-1111-1111-1111-111111111111" // This is redeclared here (with lowercase) to solve a lint error
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

type RequestOccurrence struct {
	ID         string
	Request    *ReceivedRequest
	Response   *ReceivedResponse
	WasMatched bool
}

type Mapping struct {
	ID string
}

type Requests struct {
	Requests []*RequestOccurrence
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

func verifyNumberOfRegisterRequest(comaparator string, number int) {
	reqs, err := findAllMatchingRequests(getRegisterURL(), "POST")
	Expect(err).NotTo(HaveOccurred())
	Expect(len(reqs)).Should(BeNumerically(comaparator, number))
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
	return fmt.Sprintf("/api/assisted-install/v2/infra-envs/%s/hosts", infraEnvID)
}

func getNextStepsURL(hostID string) string {
	return fmt.Sprintf("/api/assisted-install/v2/infra-envs/%s/hosts/%s/instructions", infraEnvID, hostID)
}

func getStepReplyURL(hostID string) string {
	return fmt.Sprintf("/api/assisted-install/v2/infra-envs/%s/hosts/%s/instructions", infraEnvID, hostID)
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

func addRegisterStubInvalidCommand(hostID string, reply int, infraEnvID string, retryDelay int64) (string, error) {

	hostUUID := strfmt.UUID(hostID)
	hostKind := "host"

	returnedHost := &models.Host{
		ID:   &hostUUID,
		Kind: &hostKind,
	}

	stepRunnerCommand := &models.HostRegistrationResponseAO1NextStepRunnerCommand{
		Command:      "i_do_not_exist",
		RetrySeconds: retryDelay,
	}

	registerResponse := &models.HostRegistrationResponse{
		Host:                  *returnedHost,
		NextStepRunnerCommand: stepRunnerCommand,
	}

	b, err := json.Marshal(&registerResponse)
	if err != nil {
		return "", err
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

func addRegisterStub(hostID string, reply int, infraEnvID string) (string, error) {
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

		stepRunnerCommand := &models.HostRegistrationResponseAO1NextStepRunnerCommand{
			Command: "/usr/bin/next_step_runner",
			Args: []string{
				"--url", AssistedServiceURLFromAgent,
				"--infra-env-id", infraEnvID,
				"--host-id", hostID,
			},
		}

		registerResponse := &models.HostRegistrationResponse{
			Host:                  *returnedHost,
			NextStepRunnerCommand: stepRunnerCommand,
		}

		b, err = json.Marshal(&registerResponse)
		if err != nil {
			return "", err
		}
	case http.StatusForbidden:
		errorReply := &models.InfraError{
			Code:    swag.Int32(http.StatusForbidden),
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

func addNextStepStub(hostID string, nextInstructionSeconds int64, afterStep string, instructions ...*models.Step) (string, error) {
	if instructions == nil {
		instructions = make([]*models.Step, 0)
	}
	steps := models.Steps{
		NextInstructionSeconds: nextInstructionSeconds,
		Instructions:           instructions,
		PostStepAction:         swag.String(afterStep),
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

func findAllMatchingRequests(url, method string) ([]*RequestOccurrence, error) {
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

	ret := make([]*RequestOccurrence, 0)
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
	return startContainer(agentServiceName)
}

func stopAgent() error {
	return stopContainer(agentServiceName)
}

func startContainer(args ...string) error {
	args = append([]string{"-f", "docker-compose.yml", "run", "-d", "--no-deps"}, args...)

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
	Expect(stopAgent()).NotTo(HaveOccurred())
	Expect(resetRequests()).NotTo(HaveOccurred())
	Expect(deleteAllStubs()).NotTo(HaveOccurred())
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

func createCustomStub(stepType models.StepType, command string, args ...string) *models.Step {
	return &models.Step{
		StepType: stepType,
		Command:  command,
		Args:     args,
	}
}

func setReplyStartAgent(hostID string) {
	_, err := addStepReplyStub(hostID)
	Expect(err).NotTo(HaveOccurred())
	Expect(startAgent()).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)
	verifyRegisterRequest()
	verifyGetNextRequest(hostID, true)
}

func setPostReply(hostID string) {
	_, err := addStepReplyStub(hostID)
	Expect(err).NotTo(HaveOccurred())
	verifyGetNextRequest(hostID, true)
}

func generateNsenterStep(stepType models.StepType, args []string) *models.Step {
	commands_args := []string{"-t", "1", "-m", "-i", "--"}
	commands_args = append(commands_args, args...)
	stepID := uuid.New().String()

	return &models.Step{
		StepType: stepType,
		StepID:   stepID,
		Command:  "nsenter",
		Args:     commands_args,
	}
}

func generateContainerStep(stepType models.StepType, containerAdditionalArgs []string, commandArgs []string) *models.Step {
	containerArgs := []string{
		"run", "--privileged", "--rm",
		"-v", "/var/log:/var/log",
		"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
	}
	containerArgs = append(containerArgs, containerAdditionalArgs...)
	stepID := uuid.New().String()

	args := make([]string, 0)
	args = append(args, containerArgs...)
	args = append(args, agentImage)
	args = append(args, commandArgs...)

	return &models.Step{
		StepType: stepType,
		StepID:   stepID,
		Command:  defaultContainerTool,
		Args:     args,
	}
}
