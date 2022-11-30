+++
title = "Writing a Helper"
toc = true
weight = 30
pre = "<b>1.3. </b>"
+++

The Helper is a script or program written in any language that is in charge of handling the steps that are unique to your environment.

Essentially it receives Server Metadata on its STDIN and writes a response to STDOUT that is used to configure the server.

See the sections below for sample Helper scripts.

## Input

Let's look at what you might receive as input on STDIN. The specifics will vary a bit between scenarios which we will call out in specific sections. The data you will get will all already have been validated. For example the JWT would have been parsed already and known to be valid.

```json
{
  "identity": "24bd22cdb279.choria.local",
  "csr": null,
  "ed25519_pubkey": {
    "public_key": "2ddc906446a935aefde69ceee2beb3b1d85f264153720d9a752aa88771c0594c",
    "directory": "/etc/choria",
    "signature": "34c5c4d0564747b1d0577ddc9c4084de547235d56e2803a754c76ba4eabbf17e31453ebd9ca4b4a7dd5e25ccf57afd4138519de4666c81d3ee3cd5381cc15a0a"
  },
  "inventory": "{\"agents\":[\"choria_provision\",\"choria_util\",\"discovery\",\"rpcutil\"],\"classes\":[],\"collectives\":[\"provisioning\"],\"data_plugins\":[],\"facts\":{},\"machines\":[],\"main_collective\":\"provisioning\",\"version\":\"0.99.0.20221129\",\"upgradable\":true}",
  "jwt": {
    "cht": "s3cret",
    "chs": false,
    "chu": "nats://broker.choria.local:4222",
    "chpd": true,
    "extensions": null,
    "ou": "choria",
    "v2": true,
    "purpose": "choria_provisioning",
    "iss": "Choria Tokens Package v0.26.2",
    "sub": "choria_provisioning",
    "nbf": 1669809657,
    "iat": 1669809657,
    "jti": "3a4f9896e013498daeedbcb9a82fcd3c"
  }
}
```

| Key              | Description                                                                                                                   |
|------------------|-------------------------------------------------------------------------------------------------------------------------------|
| `identity`       | The Choria Server identity being provisioned - typically FQDN                                                                 |
| `csr`            | Would be non nil when the `pki` feature is enabled for obtaining x509 certificates                                            |
| `ed25519_pubkey` | Would be non nil when the `ed25519` feature is enabled                                                                        |
| `inventory`      | Is the JSON result of `choria req rpcutil inventory` this lets you find facts, version information and more about the server. |
| `jwt`            | Is the verified contents of the `provisioning.jwt` on the server when the `jwt` feature is enabled                            |

## Output

The response your helper should write to STDOUT is also in JSON format.

Here is an example that configures Choria Server and uses the `jwt` feature to enroll a node into an Organization Issuer based network.

```json
{
  "defer": false,
  "msg": "Done",
  "certificate": "",
  "ca": "",
  "configuration": {
    "identity": "eb873ce040d7.choria.local",
    "loglevel": "info",
    "plugin.choria.server.provision": "false",
    "plugin.choria.middleware_hosts": "nats://broker.choria.local:4222",
    "rpcauthorization": "0",
    "plugin.choria.status_file_path": "/var/log/choria-status.json",
    "plugin.choria.submission.spool": "/var/lib/choria/submission",
    "plugin.security.issuer.names": "choria",
    "plugin.security.issuer.choria.public": "e72cba5268b34627b75c5ceae9449ad16d62f15f862c30d4e0e7d2588e2e6259",
    "plugin.security.provider": "choria",
    "plugin.security.choria.token_file": "/etc/choria/server.jwt",
    "plugin.security.choria.seed_file": "/etc/choria/server.seed",
    "plugin.choria.machine.store": "/etc/choria/machine"
  },
  "server_claims": {
    "exp": 157680000,
    "permissions": {
      "streams": true,
      "submission": true
    }
  }
}
```

{{% notice style="warning" %}}
The configuration must set `plugin.choria.server.provision` to disable provisioning, else the node will keep being reprovisioned forever.
{{% /notice %}}

Response keys used by scenarios:

| Key               | Description                                                                                                 |
|-------------------|-------------------------------------------------------------------------------------------------------------|
| `defer`           | Defers the provisioning, this is a soft state meaning the server will come back and be retried later        |
| `shutdown`        | Issues a shutdown on the server with exit code 0, systemd will not restart it.                              |
| `msg`             | A message to log on the Server to explain why it is being deferred or shut down                             |
| `configuration`   | A JSON Object of configuration items in key-value pairs, will be written to the server config               |
| `action_policies` | A JSON Object of Action Policy policies in key-value pairs, where the key is an agent name                  |
| `opa_policies`    | A JSON Object of Open Policy Agent policies in key-value pairs, where the key is an agent name or `default` |

When using the `pki` feature used to enroll with a Certificate Authority:

| Key           | Description                                                                                                            |
|---------------|------------------------------------------------------------------------------------------------------------------------|
| `key`         | An optional x509 private key that the server should use, will be encrypted using a unique one-time password in transit |
| `certificate` | The signed certificate in PEM format                                                                                   |
| `ca`          | The Certificate Authority public key in PEM format                                                                     |
| `ssldir`      | What directory to store the key, certificate and ca in on the server                                                   |

When using the `jwt` feature to create server JWT tokens for Organization Issuer based networks:

| Key             | Description                                                          |
|-----------------|----------------------------------------------------------------------|
| `server_claims` | The Choria Server token claims to base the server JWT on             |

When using the `upgrades` feature to in-place upgrade servers:

| Key       | Description                                              |
|-----------|----------------------------------------------------------|
| `upgrade` | The version to upgrade the server to before provisioning |

## Enrolling nodes with a Certificate Authority

Most typically you have a Enterprise Certificate Authority or you made your own using something like [cfssl](https://cfssl.org/).

In this mode Choria Server will generate a private key on its disk, create a CSR and your helper will receive the CSR in PEM format.  In your helper you then simply interact with your CA to sign the CSR and respond with the signed Certificate and CA public key.  This way once provisioned your server will be fully enrolled for mTLS.

To enable the Provisioner to request the CSR the `pki` feature needs to be enabled in the Provisioner Configuration.

The `csr` input field will then be a JSON Object with:

| Key          | Description                                                                                          |
|--------------|------------------------------------------------------------------------------------------------------|
| `csr`        | The PEM encoded CSR the node is sending                                                              |
| `public_key` | The public part of the key the server created                                                        |
| `ssldir`     | The directory the server created the private key in so it can be used in the generated configuration |

Once you have this data you can use your CA API to enroll the node and get a signed certificate back. Simply put the resulting PEM data in the `certificate`and `ca` keys in the reply.  You can set a SSL directory but typically just set `ssldir` to what was received in the input.

{{% notice style="info" %}}
A basic sample helper that enrolls in a `cfssl` based CA can be seen in [cfssl-helper.rb](../../cfssl-helper.rb)
{{% /notice %}}

In general the Private key stays on the node and you do not need it. Some Certificate Authorities require the private key to be accessible when signing a request. Provisioner support that, if you generate a key in the provisioner and add it to the reply in the `key` JSON field a single use Shared Secret negotiated using Diffie-Hellman will be used to encrypt the key in transit. 

I would not suggest ever to use a CA that requires you to transmit the Private Key during enrollment, it's best to assume your CA is unusable at that point and consider a Organization Issuer based deployment.

## Enrolling nodes with an Organization Issuer

In cases where a Certificate Authority is not available or it is operated in a way that makes it unsuitable for mTLS use you might opt to deploy Choria in an Organization Issuer based setup.  The basic setup of that mode is out of scope for this document.

{{% notice secondary "Version Hint" code-branch %}}
This applies only to Choria 0.27.0 and newer which is due to ship early 2023
{{% /notice %}}

To enrol nodes in an Organization Issuer based network you need to enable the `ed25519` feature in the Provisioner Configuration.

Once enabled the `ed25519_pubkey` field will hold a JSON Object with these values:

| Key          | Description                                                                                                                           |
|--------------|---------------------------------------------------------------------------------------------------------------------------------------|
| `public_key` | The `ed25519` public key unique to the server                                                                                         |
| `directory`  | The directory that will hold `server.seed` and where `server.jwt` will be saved to later                                              |
| `signature`  | A signature of the request made using the private key matching the `public_key`, this can be ignored as it would already be validated |

Once you received these you can include a `server_claims` in your reply to give the server access to specific features in its JWT claims. The list here is correct for `0.27.0`. For an up-to-date list see the [Go Documentation for your version of Choria](https://pkg.go.dev/github.com/choria-io/go-choria@v0.26.2/tokens#ServerClaims).

The claims can include `permissions` that have these properties.

| Permission     | Description                                                                                                                         |
|----------------|-------------------------------------------------------------------------------------------------------------------------------------|
| `submission`   | Allows the server to use [Choria Submission](https://choria.io/docs/streams/submission/)                                            |
| `streams`      | Allows the server to access [Choria Streams](https://choria.io/docs/streams/) for example to read KV buckets from autonomous agents |
| `governor`     | Allows the server to access [Choria Governor](https://choria.io/docs/streams/governor/) from autonomous agents                      |
| `service_host` | Allows the server to host Services                                                                                                  |

You can also use them to restrict it to a specific sub collective and more. Most values will default to sane defaults when not given.

## Upgrading Servers

Choria Server can be upgraded in-place to a new version. This is done by overwriting the binary at run-time with one downloaded from a specifically prepared repository.

This can only be done during Provisioning and requires the `upgrades` feature to be enabled.

{{% notice secondary "Version Hint" code-branch %}}
This applies only to Choria 0.27.0 and newer which is due to ship early 2023
{{% /notice %}}

Compatible nodes will have the `upgradable` key in the inventory received and will be set to `true` when upgrades are enabled. If you only want to support the latest nodes you can use this to determine if a upgrade is needed along with the `version` key in the inventory. On nodes that are too old set `shutdown` or `defer`.

You would have to configure the Provisioner with a repository location and set up a repository on a HTTP server as per the guidelines from [go-updater](https://github.com/choria-io/go-updater).

With all of this in place you can add the `upgrade` key to the helper response that should just be a desired version like `0.28.0`. Provisioner will then attempt to upgrade the node.
