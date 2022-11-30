+++
title = "Configuration"
toc = true
weight = 20
pre = "<b>2. </b>"
+++

Configuring the Provisioner requires exact knowledge of your deployment needs, we suggest you start with first configuring servers by hand while exploring your needs, once the specific configuration file contents and security needs are defined those can be configured into the Provisioner system.

Deploying a Provisioner should not be the first thing you do.

## General Requirements

### Choria Broker

You must be running the Choria Broker and it will need a few extra configuration items.

### JWT files or custom binaries

You can enable Provisioning mode in the standard Open Source Choria Server by placing a `provisining.jwt` in the right place.  Those doing custom builds can also apply build-time defaults that has no external dependencies.

### Helper Script

Provisioner calls into a user-supplier helper, written in a language like Ruby but any language can be used, to generate the per-node configuration properties. 

### Certificate Authorities

If you have a x509 based Choria deployment (the default) you will need to be able to generate or sign certificates for your nodes. To do this your CA needs to have a API that you can call to issue certificates.

There are other options requiring JWT files to be signed also if that is not possible.

### Authorization Policies

In this mode typically nodes do not have any Policies controlling who can invoke Agents and Actions.  You will probably want to write a [Open Policy Agent](https://www.openpolicyagent.org/) policy that the Provisioner will deploy

### Monitoring

We expose metrics to Prometheus, for in-depth monitoring you will need Prometheus or a compatible system.
