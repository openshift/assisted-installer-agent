# Agent Terminal User Interface (agent-tui)

## What is the agent-tui tool?

This tool aims to improve the installation outcomes where:

* the provided manifests do not match the situation found at runtime and
    re-configuration is necessary, or
* an interactive installation is being performed (not yet implemented)

Thus, it is **only aimed at agent-based installation** and not for the managed
service.

In the first case, it will perform some checks to determine if the installation
can proceed. If all the checks come back successful, the tui will quit and
installation will proceed. If some check fails, it will present the user with
information relative to the failed checks and give the user a chance to correct
those.

## How is it built?

It is built using the tivo/tview golang library

## Will this grow to be an entire agent based TUI for interactive installation?

It is not likely

## Where can I get support for it?

The right place is in https://issues.redhat.com. Filing an OpenShift bug
(OCPBUGS project) for the "Installer / Agent based installation" component
