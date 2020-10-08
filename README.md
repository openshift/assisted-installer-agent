# Assisted Installer Agent

## Contents

This project contains several executables that are intended to work with the
[OpenShift Assisted Installer Service](https://github.com/openshift/assisted-service):

* **agent** - This is the entry point of an assisted installation on the host side. It registers a host with the
              assisted installer service and starts **next_step_runner**.
* **next_step_runner** - Polls the assisted installer service for next steps of an installation, and executes them.
* **inventory** - Provides information about the host's inventory (CPU, memory, disk, network interfaces, etc.).
* **connectivity_check** - Tests connectivity via NICs provided in the argument.
* **free_addresses** - Detects free addresses in a subnet provided in the argument.
* **logs_sender** - Packages system logs and uploads them to the server for troubleshooting.
* **dhcp_lease_allocate** - Allocates IP addresses in DHCP. An assisted cluster needs several reserved IPs such as API VIP and ingress VIP.
* **apivip_check** - Tests connectivity to the API VIP of the assisted cluster.

The project uses the [**ghw**](https://github.com/jaypipes/ghw) library to collect the inventory and generate a unique
identifier (UUID) of the host that the agent is running on.

### Agent Flags

* *--url*: The URL of an assisted installer server, includes a schema and optionally a port.
* *--cluster-id*: ID of the cluster the host will be connected to.
* *--agent-version*: Version (image tag) of the agent being run, collected for diagnostic purposes.
* *--interval*: Interval in seconds between consecutive requests that the agent sends to the server. Default is 60.
* *--with-text-logging*: Enable writing the agent logs to _/var/log/agent.log_. Default is `true`.
* *--with-journal-logging*: Enable writing logs to systemd journal. Default is `true`.
* *--insecure*: Skip certificate validation in case of HTTPS transport. Should be used only for testing. Default is `false`.
* *--cacert*: Path to a custom CA certificate file in PEM format.
* *--help*: Print help message and exit.

### Packaging

By default, the executables are packaged in a container image `quay.io/ocpmetal/assisted-installer-agent:latest`.
The executables inside the image reside under _/usr/bin/_.

### Running

Since the agent is a statically linked go executable, it can be copied and run outside a container. If running outside a container,
the agent must run as root. If running in a Podman container, the `podman run` command should be invoked with `--net=host` and `--privileged`.

The other tools can be invoked using `podman run <flags> quay.io/ocpmetal/assisted-installer-agent:latest <executable>`.

### Dependencies

* [Docker](https://docs.docker.com/) (including [Docker Compose](https://docs.docker.com/compose/)) is used for subsystem testing,
  and is not required in runtime.
* [Skipper](https://github.com/Stratoscale/skipper) is used for building and testing. Can be installed with `pip install strato-skipper`.
* [Podman](https://podman.io/) is the preferred container runtime to run the executables.

### Building

To build the executables run: `skipper make`
To build the container image run: `skipper make build-image`

### Testing

For unit tests, run `skipper make unit-test`.

The subsystem tests use Docker Compose to run the agent and [Wiremock](http://wiremock.org/) stubs that simulate the assisted installer service. To perform the subsystem tests run `skipper make subsystem`.

**WARNING:** The subsystem tests can only run with the default image name and tag. You can build it locally just for this purpose.

To run selected system tests use a [regular expression](https://onsi.github.io/ginkgo/#focused-specs): `skipper make subsystem FOCUS=register`.

### Publishing

To publish the container image run `skipper make push`.
You can override the image name and tag via the `ASSISTED_INSTALLER_AGENT` variable.