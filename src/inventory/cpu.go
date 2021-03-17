package inventory

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

func max(f1, f2 float64) float64 {
	if f1 > f2 {
		return f1
	}
	return f2
}

type field struct {
	Field string
	Data  string
}

type lscpu struct {
	Lscpu []*field
}

func GetCPU(dependencies util.IDependencies) *models.CPU {
	ret := &models.CPU{}
	o, e, exitCode := dependencies.Execute("lscpu", "-J")
	if exitCode != 0 {
		logrus.Warnf("Error running lscpu: %s", e)
		return ret
	}
	var l lscpu
	err := json.Unmarshal([]byte(o), &l)
	if err != nil {
		logrus.Warnf("Error unmarshaling lscpu: %s", err.Error())
		return ret
	}
	for _, f := range l.Lscpu {
		switch f.Field[:len(f.Field)-1] {
		case "Architecture":
			ret.Architecture = f.Data
		case "Model name":
			ret.ModelName = f.Data
		case "CPU(s)":
			ret.Count, _ = strconv.ParseInt(f.Data, 10, 64)
		case "CPU MHz", "CPU max MHz":
			f, _ := strconv.ParseFloat(f.Data, 64)
			ret.Frequency = max(ret.Frequency, f)
		case "Flags":
			ret.Flags = strings.Split(f.Data, " ")
		}
	}
	return ret
}
