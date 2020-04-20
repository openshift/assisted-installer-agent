# introspector
## Contents
This projects generates couple of products.  Each one packed in a docker image:
* **agent** - Agent to bm-inventory
* **dmidecode** - A container image that bundles dmidecode
* **hardware_info** - A container image for running hardware_info
* **connectivity_check** - A container image for running connectivity_check
## agent
The agent communicates with bm-inventory.  It uses the **dmidecode** container image to generate unique uuid identifying the host 
that the agent is running on.
Currently, the functionality of **hardware_info** and **connectivity_check** is bundled in the agent itself, besides their dedicated container images.

### Flags

* *--host*:  The inventory host as dns name or ip address. Default is "api.openshift.com".
* *--port*: The inventory port number. Default is 80.
* *--cluster-id*: The cluster id.  Default is "default-cluster".
* *--interval*: Interval in seconds between consecutive times that the agent accesses the inventory.  Default is 60.
* *--dmidecode-image*: The name of the dmidecode image.  Default is "quay.io/oamizur/dmidecode"
* *--text*: Dump the hardware info as json to the standard output and exit.
* *--connectivity*: Perform connectivity check and dump the results as json to the standard output.  
The details of the nodes to check the connectivity with are provided as a parameter having json format.
* *--help*: Provide help message.

### Packaging
The agent's executable is packaged in a container quay.io/oamizur/agent.  The excutable in the container is /usr/bin/agent.

### Running
Since the agent is a statically linked go executable it can be copied and ran outside a container.If running outside a container, 
the agent shoud run as root.  If running in a container, the docker or podman should be invoked with --net=host and --privileged flags.

The agent writes its log messages to /var/log/agent.log.
### Dependecies
* docker
* iputils (ping, arping)
* iproute
* Linux utilities such as lscpu, free (etc.).

### Building
introspector uses skipper for building and testing.

To build executables run: `skipper make`

To build container images run: `skipper make build-image`

### Testing
Only subsystem testing currently exists for agent. The subsystem uses docker-compose to run the agent and wiremock simulating the inventory.

To perform the subsystem test run: `skipper make subsystem`

### Deployment
To deploy all  container images in the project run `skipper make push`

 