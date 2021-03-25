module github.com/openshift/assisted-installer-agent

go 1.15

require (
	github.com/Microsoft/go-winio v0.4.15-0.20200113171025-3fe6c5262873 // indirect
	github.com/docker/docker v17.12.0-ce-rc1.0.20200505174321-1655290016ac+incompatible // indirect
	github.com/go-openapi/strfmt v0.20.0
	github.com/go-openapi/swag v0.19.14
	github.com/google/uuid v1.2.0
	github.com/hashicorp/go-multierror v1.1.0
	github.com/jaypipes/ghw v0.6.1
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/openshift/assisted-service v1.0.10-0.20210330171458-790c68adcee8
	github.com/openshift/baremetal-runtimecfg v0.0.0-20210210163937-34f98e0f48fd
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/ssgreg/journald v1.0.0
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/thoas/go-funk v0.7.0
	github.com/vishvananda/netlink v1.1.0
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c
	gopkg.in/yaml.v2 v2.4.0
)

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.3.1
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20200715132148-0f91f62a41fe // Use OpenShift fork
	github.com/openshift/hive/pkg/apis => github.com/carbonin/hive/pkg/apis v0.0.0-20210209195732-57e8c3ae12d1
	k8s.io/api => k8s.io/api v0.0.0-20190712022805-31fe033ae6f9
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190711222657-391ed67afa7b
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
)
