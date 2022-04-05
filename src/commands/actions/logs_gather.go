package actions

import (
	"bytes"
	"html/template"
	"strconv"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"

	"github.com/go-openapi/swag"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
)

type logsGather struct {
	args         []string
	generatedCmd string
}

func (a *logsGather) Validate() error {
	name := "logs gather"
	params := models.LogsGatherCmdRequest{}
	err := validateCommon(name, 1, a.args, &params)
	if err != nil {
		return err
	}

	a.generatedCmd, err = createUploadLogsCmd(params)
	if err != nil {
		log.WithError(err).Errorf("Failed to generate command for %s with params %s", name, a.args)
		return err
	}

	return nil
}

func (a *logsGather) CreateCmd() (string, []string) {
	return a.Command(), strings.Fields(a.generatedCmd)
}

func createUploadLogsCmd(params models.LogsGatherCmdRequest) (string, error) {

	data := map[string]string{
		"BASE_URL":               config.GlobalAgentConfig.TargetURL,
		"CLUSTER_ID":             params.ClusterID.String(),
		"HOST_ID":                params.HostID.String(),
		"INFRA_ENV_ID":           params.InfraEnvID.String(),
		"AGENT_IMAGE":            config.GlobalAgentConfig.AgentVersion,
		"SKIP_CERT_VERIFICATION": strconv.FormatBool(config.GlobalAgentConfig.InsecureConnection),
		"BOOTSTRAP":              strconv.FormatBool(swag.BoolValue(params.Bootstrap)),
		"INSTALLER_GATHER":       strconv.FormatBool(params.InstallerGather),
		"MASTERS_IPS":            strings.Join(params.MasterIps, ","),
	}

	if config.GlobalAgentConfig.CACertificatePath != "" {
		data["CACERTPATH"] = config.GlobalAgentConfig.CACertificatePath
	}

	cmdArgsTmpl := "1h podman run --rm --privileged --net=host " +
		"-v /run/systemd/journal/socket:/run/systemd/journal/socket -v /var/log:/var/log -v /etc/pki:/etc/pki " +
		"{{if .CACERTPATH}} -v {{.CACERTPATH}}:{{.CACERTPATH}} {{end}}" +
		"{{if eq .BOOTSTRAP `true`}} -v /root/.ssh:/root/.ssh -v /tmp:/tmp {{end}}" +
		"--env PULL_SECRET_TOKEN --name logs-sender --pid=host {{.AGENT_IMAGE}} logs_sender " +
		"-url {{.BASE_URL}} -cluster-id {{.CLUSTER_ID}} -host-id {{.HOST_ID}} -infra-env-id {{.INFRA_ENV_ID}} " +
		"--insecure={{.SKIP_CERT_VERIFICATION}} -bootstrap={{.BOOTSTRAP}} -with-installer-gather-logging={{.INSTALLER_GATHER}}" +
		"{{if .MASTERS_IPS}} -masters-ips={{.MASTERS_IPS}} {{end}}" +
		"{{if .CACERTPATH}} --cacert {{.CACERTPATH}} {{end}}"

	t, err := template.New("cmd").Parse(cmdArgsTmpl)
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	if err := t.Execute(buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (a *logsGather) Run() (stdout, stderr string, exitCode int) {
	command, args := a.CreateCmd()
	return util.ExecutePrivileged(command, args...)
}

func (a *logsGather) Command() string {
	return "timeout"
}

func (a *logsGather) Args() []string {
	return a.args
}