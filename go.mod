module github.com/openshift/assisted-installer-agent

go 1.13

require (
	github.com/go-openapi/strfmt v0.19.5
	github.com/go-openapi/swag v0.19.9
	github.com/golang/mock v1.4.4 // indirect
	github.com/google/uuid v1.1.1
	github.com/jaypipes/ghw v0.6.1
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/openshift/assisted-service v1.0.10-0.20201029165706-3f7784f45d1f
	github.com/openshift/baremetal-runtimecfg v0.0.0-20200820213150-b2b74d7c6a5c
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/ssgreg/journald v1.0.0
	github.com/stretchr/testify v1.6.1
	github.com/vishvananda/netlink v1.1.0
	gopkg.in/yaml.v2 v2.3.0
)

replace (
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20200715132148-0f91f62a41fe // Use OpenShift fork
	k8s.io/api => k8s.io/api v0.0.0-20190712022805-31fe033ae6f9
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190711222657-391ed67afa7b
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
)
