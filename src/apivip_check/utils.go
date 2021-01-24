package apivip_check

import (
	"bytes"
	"text/template"
)

const nodeIgnitionFormat = `{
	"ignition": {
		"version": "3.1.0",
		"config": {
			"merge": [{
				"source": "{{.SOURCE}}"
			}]
		}
	}
}`

func FormatNodeIgnitionFile(source string) ([]byte, error) {
	var ignitionParams = map[string]string{
		"SOURCE": source,
	}

	tmpl, err := template.New("nodeIgnition").Parse(nodeIgnitionFormat)
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	if err = tmpl.Execute(buf, ignitionParams); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
