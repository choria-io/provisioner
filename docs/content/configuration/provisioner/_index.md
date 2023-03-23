+++
title = "Configuration File"
toc = true
weight = 40
pre = "<b>1.4. </b>"
+++

Provisioner is configured using `/etc/choria-provisioner/choria-provisioner.yaml` typically.  It's a YAML format file with a few required settings and a number of optional ones.  Changes to the file requires the process to be restarted.

## Choria Client Configuration

As the Provisioner connects to the Choria Broker as a client it needs a configuration that allows it access. Create `/etc/choria-provisioner/choria.cfg` with the following based on needs.  These settings will augment those in `/etc/choria/client.cfg`.

### x509 based Networks

```yaml
plugin.security.provider = file
plugin.security.file.certificate = /etc/choria-provisioner/ssl/cert.pem
plugin.security.file.key = /etc/choria-provisioner/ssl/key.pem
plugin.security.file.ca = /etc/choria-provisioner/ssl/ca.pem
```

Obtain these certificates the same way you would obtain any other certificate, perhaps using `choria enroll --certname provisioner.mcollective`.

### Organization Issuer Based Networks

{{% notice secondary "Version Hint" code-branch %}}
This applies only to Choria 0.27.0 and Provisioner 0.15.0 and newer which is due to ship early 2023
{{% /notice %}}

In these setups you need a client JWT with `--server-provisioner` and `--issuer` set while creating the client JWT:

```nohighlight
$ choria jwt keys /etc/choria-provisioner/signer.seed /etc/choria-provisioner/signer.public
$ choria jwt client /etc/choria-provisioner/signer.jwt provisioner_signer issuer \
     --public-key $(cat /etc/choria-provisioner/signer.public) \ 
     --server-provisioner \
     --validity 365d \
     --vault \
     --issuer
```

The `/etc/choria-provisioner/choria.cfg` would then have:

```ini
plugin.security.provider = choria
plugin.security.choria.token_file = /etc/choria-provisioner/credentials/signer.jwt
plugin.security.choria.seed_file = /etc/choria-provisioner/credentials/signer.seed
identity = provisioner_signer
```

## Common Settings

These settings are required for all scenarios, below an example configuration followed by explanation:

```yaml
workers: 4
interval: 1m
logfile: /dev/stdout
loglevel: warn
helper: /etc/choria-provisioner/provisioner/helper.rb
token: s3cret
site: testing
broker_provisioning_password: s3cret
jwt_verify_cert: e72cba5268b34627b75c5ceae9449ad16d62f15f862c30d4e0e7d2588e2e6259
jwt_signing_key: /etc/choria-provisioner/credentials/signer.seed
jwt_signing_token: /etc/choria-provisioner/credentials/signer.jwt

features:
  jwt: true
  ed25519: true
```

| Item                           | Description                                                                                | Default         |
|--------------------------------|--------------------------------------------------------------------------------------------|-----------------|
| `workers`                      | How many concurrent helpers to call while provisioning                                     | number of cores |
| `interval`                     | How often to perform a discovery against the network for new machines                      | `1m`            |
| `logfile`                      | Where to write the log                                                                     |                 |
| `loglevel`                     | The level to log at, `debug`, `info`, `warn` or `error`                                    | `info`          |
| `helper`                       | Path to the helper script                                                                  |                 |
| `token`                        | The value of the token set using `--token` in the `provisioning.jwt`                       |                 |
| `site`                         | A unique name for this installation, surfaced in monitoring data                           |                 |
| `monitor_port`                 | The post to listen on for monitoring requests                                              |                 |
| `broker_provisioning_password` | The password configured in the broker `plugin.choria.network.provisioning.client_password` |                 |
| `features.jwt`                 | Enables fetching and validating `provisioning.jwt`, should almost always be `true`         | `false`         |
| `features.ed25519`             | Enables JWT processing for Organization Issuer based networks                              | `false`         |
| `features.pki`                 | Enables x509 enrollment                                                                    | `false`         |
| `features.upgrades`            | Enables server version upgrades                                                            | `false`         |

## PKI / x509 Enrollment

When enabled the Provisioner will fetch a CSR from the node and ask the node to create a private key that stays on the node. The helper can then get the certificate signed and the signed certificate will be sent to the node.

To enable set `features.pki` to `true`.

Here we reference our x509 key and certificate

| Item                | Description                                                         | Default |
|---------------------|---------------------------------------------------------------------|---------|
| `jwt_verify_cert`   | Full path to the public certificate used to sign `provisioning.jwt` |         |
| `jwt_signing_key`   | Full path to our private key, also used in `choria.conf`            |         |

## Organization Issuer based Enrollment

When enabled the Provisioner will sign and issue server JWTs with custom claims and signatures. No x509 steps will be done.

To enable set `features.ed25519` to `true`.

Here we reference our JWT that gives us the right to provision and issue new JWTs

| Item                | Description                                            | Default |
|---------------------|--------------------------------------------------------|---------|
| `jwt_verify_cert`   | The hex encoded public key of your Organization Issuer |         |
| `jwt_signing_key`   | The private ed25519 key, also used in `choria.conf`    |         |
| `jwt_signing_token` | The JWT token, also used in `choria.conf`              |         |

## Choria Server Upgrades

Choria Provisioner can upgrade Choria Servers using a [go-updated](https://github.com/choria-io/go-updater) repository.

To enable set `features.upgrades` to `true`.

Here we configure the updates repository and say how failures are handled

| Item                  | Description                                   | Default |
|-----------------------|-----------------------------------------------|---------|
| `upgrades_repository` | URL to your updates repository                |         |
| `upgrades_optional`   | Continue provisioning even if upgrading fails | `false` |

## Clustered Deployments

The Provisioner is generally fast enough to not need a cluster of active-active servers, so we support deploying multiple instances of Provisioner and using [Leader Elections](https://choria.io/docs/streams/elections/) to elect one in the cluster as a leader that will be actively provisioning nodes.

To enable this the client needs access to Streams and Leader Elections so need `--stream-user` and `--elections-user` passed when creating its JWT.

Campaigning will be on a backoff schedule up to 20 second between campaigns, this means there can be up to a minute of downtime during a failover scenario, generally that's fine for the Provisioner.

If a Provisioner was on standby and becomes leader it will immediately perform a discovery to pick up any nodes ready for provisioning.

Your broker must therefor have [Choria Streams](https://choria.io/docs/streams/) enabled.

| Item              | Description                     | Default |
|-------------------|---------------------------------|---------|
| `leader_election` | Enables active-standby clusters | `false` |
