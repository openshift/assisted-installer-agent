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
* *--agent-version*: Version (full image reference) of the agent being run, used for diagnostic and upgrade purposes.
* *--interval*: Interval in seconds between consecutive requests that the agent sends to the server. Default is 60.
* *--with-text-logging*: Enable writing the agent logs to ``/var/log/agent.log``. Default is `true`.
* *--with-journal-logging*: Enable writing logs to systemd journal. Default is `true`.
* *--insecure*: Skip certificate validation in case of HTTPS transport. Should be used only for testing. Default is `false`.
* *--cacert*: Path to a custom CA certificate file in PEM format.
* *--help*: Print help message and exit.

### Inventory Flags
> [!WARNING]
> The inventory process is typically executed during the agent provisioning workflow. Therefore, the inventory configuration must be set before reaching the step of calling the `inventory` binary.

* *--gpu-config-file*: Path to a configuration file to filter the GPU discovery process.

The GPU configuration file is in YAML format with 3 elements:
* `classes`. PCI class (base + subclass) allowed.
* `vendors`. Combination of PCI class + PCI vendor-id
* `models`. Combination od PCI class + PCI vendor-id + PCI device-id

The idea for filtering devices is to go from the most generic (PCI Class) to the most specific (Hardware model). If any hardware matches a generic filter it will be added to the list of GPUs. We discard the use of a single element, for example the vendor-id, because it can cause false positives. For example, Intel hardware (`0x8086`) can be a VGA display controller but also a sound controller. For this reason, for a vendor-id we should add a PCI class to be more precise.

Example:
```yaml
---
# Include all Display controllers, VGA compatible (0x0300)
classes:
  - '0300'
# Include Nvidia (0x10de) 3D controllers (0x302)
vendors:
  - '0302 10de'
# Include Habana labs (0x1da3) Gaudi2 (0x1020) AI accelerator (0x1200)
models:
  - '1200 1da3 1020'
```

Example filtering only Nvidia (0x10de) display controllers:
```yaml
---
# Include Nvidia (0x10de) 3D controllers (0x302)
vendors:
  - '0300 10de'
  - '0302 10de'
```

By default, the PCI classes discovered will be:
* VGA compatible display controllers (`0x0300`)
* 3D display controllers (`0x0302`)
* Display controllers (`0x0380`)
* Processing accelerators (`0x1200`)

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

