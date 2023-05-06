+++
weight = 5
+++

# Overview

In a typical Choria environment Puppet is used to provision a uniform Choria infrastructure integrated into the Puppet CA. This does not really work in large enterprises or dynamic environments.. 

Choria supports a provisioning mode where the process of enrolling a node can be managed on a per-environment basis, this is where we manage the vast differences in environments, platforms, etc. Once Choria is installed we have a unified overlay interface:

 * Custom endpoints for provisioning where needed. Optionally programmatically determined via plugins that can be compiled into Choria
 * Fully dynamic generation of configuration based on node metadata and fleet environment
 * Self-healing via cycling back into provisioning on critical error
 * On-boarding of machines at a rate of thousands a minute

**Choria Provisioner** owns the early lifecycle of Choria Servers:

 * Discovers unprovisioned Servers
 * Discovers their capabilities
 * Optionally upgrade their versions to the latest version
 * Enroll the Server with the Choria security system with integrations for x509 and ed25519 Choria Organization based networks
 * Create, using user supplied logic, a per-node configuration
 * Configures the Server
 * Deploys Open Policy Agent policies
 * CLI Integrations to re-provision machines on demand.

In essence this can replace the role of traditional Configuration Management with a more dynamic process for the purpose of configuring Choria Server. This is equivalent to an IoT device and it's management.

Choria Provisioner is a very high performance system capable of provisioning 1,000 servers per minute assuming corporate x509 infrastructure is performant enough. It can be deployed in an active-standby cluster mode for high availability.

## Status

This project and the Choria authentication landscape in general, is in a period of flux as we move to support a fully Certificate Authority free deployment strategy.

This project can be used today, even by users deploying with Puppet and has proven to be stable and scalable. In a future deployment scenario it will be central to the scalable operation of Choria.
