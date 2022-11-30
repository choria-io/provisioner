![Choria Provisioner](https://choria-io.github.io/provisioner/logo.png)

## Overview

This is a Provisioning system capable of Provisioning new Choria Servers onto a Choria Network without using Puppet.

Features:

 * Integrates into any CA with an API to enroll fleet nodes
 * Custom per-node configuration
 * Custom Action policies, Open Policy Agent policies and other components
 * Run-time upgrading of a Choria Server (requires future Choria release)
 * Enrollment into a Choria Organization Issuer based network (requires future Choria release)
 * Enroll 1000+ servers per minute in a highly available cluster of servers

* [Documentation](https://choria-io.github.io/provisioner/)
* [Community](https://github.com/choria-io/general/discussions)

## Status

This project and the Choria authentication landscape in general, is in a period of flux as we move to support a fully Certificate Authority free deployment strategy.

This project can be used today, even by users deploying with Puppet and has proven to be stable and scalable. In a future deployment scenario it will be central to the scalable operation of Choria.
