# Auto Provisioning System For Choria Server

Choria Server supports a Provisioning mode that assists bootstrapping the system in [large environments](https://choria.io/docs/concepts/large_scale/) or dynamic cloud based environments that might not be under strict CM control.

When an unconfigured Choria Server is in provisioning mode it will connect to a compiled-in Middleware network and join the `provisioning` sub collective.  It will optionally publish it's metadata and expose it's facts.

You can think of this as a similar setup as the old Provisioning VLANs where new servers would join to PXE boot etc. but here we expose an API that let the provisioning environment control the node.

The idea is that an automated system will discover nodes in the `provisioning` subcollective and guide them through the on-boarding process.  The on-boarding process can be entirely custom, one possible flow might be:

  * Discover all nodes in the `provisioning` sub-collective with the `choria_provision` agent
    * For every discovered nodes
        * Retrieve facts and metadata
        * Based on it's facts programmatically determine which Member Collective in a [Federation](https://choria.io/docs/federation/) this node should belong to.
        * For x509 based nodes
            * Ask the node for a CSR, potentially supplying a custom CN, OU, O, C and L
            * Sign the CSR the node provided against your own CA
        * For ed25519 based nodes
            * Ask the node for a public key and signed nonce
        * Construct a configuration tailored to this node, setting things like SRV domain or hard coded brokers
        * For ed25519 based nodes
            * Generate and sign a server JWT
        * Send the configuration, server JWT, certificate and CA chain to the node where it will configure itself
        * Request the node restarts itself within a provided splay time

After this flow the node will join it's configured Member Collective with it's signed Certificate and CA known it becomes a normal node like any other.

You can invoke the `choria_provision#reprovision` action to ask it to leave its Member Collective and re-enter the provisioning flow.

If you have a token compiled in you can restart a server when not in provisioning mode with the token via the `choria_provision#restart` action.

This project includes a provisioner that you can use, it will call a `helper` that you provide and can write in any language to integrate with your CA and generate configuration.

## Configuring Choria Server

Provisioning is enabled in the Open Source server by means of a JWT token that you create during provisioning. The JWT token holds all of the information the server
needs to find it's provisioning server and will present that token also to the provisioning server for authentication.

The token is signed using a trusted private key, the provisioner will only provision nodes presenting a trusted key.

```nohighlight
$ choria tool jwt provisioning.jwt key.pem --srv choria.example.net --token toomanysecrets
```

Here we create a `provisioning.jwt` that will instruct Choria to look for `_choria-provisioner._tcp.choria.example.net` SRV
records to find the server to connect to.

Other options can be set for example to hard code provisioning URLs, username and passwords and more.

When this file is placed in `/etc/choria/provisioning.jwt` and Choria starts without a configuration it will provision
via these settings.

Choria also support provisioning plugins to resolve this information dynamically but this requires custom binaries and should
in general be avoided.

## Preparing a Broker Environment

The broker used for provisioning is the same as for our fleet, in a special mode the broker will accept unverified TLS connections on the same port as verified mTLS ones. The unverified connections may only be used for servers in provisioning mode with a `provisioning.jwt` token and very strict permissions are applied.  These unverified TLS connections may not communicate with any other node.

A further mitigation is in place by using the Choria Broker multi tenancy features these provisioning node servers are completely isolated from any provisioned machine.

The Provisioner continues to connect over verified mTLS and presents a Username and Password to communicate with these fenced off servers.

```ini
plugin.choria.network.provisioning.signer_cert = /etc/choria-provisioner/signer-public.pem
plugin.choria.network.provisioning.client_password = provS3cret
```

This is the relevant snippet in the `broker.conf`, here the `/etc/choria-provisioner/signer-public.pem` is the public certificate used to sign the `provisioning.jwt`.

When this broker starts it will log the following warning:

```
WARN[0001] Allowing non TLS connections for provisioning purposes  component=network
```

## High Availability

The Choria Provisioner can be run in a HA cluster of any size, they will campaign for leadership using Choria Streams and whichever instance is leader will provision nodes.

**NOTE**: This requires Choria Broker 0.25.0 or newer.

Campaigning will be on a backoff schedule up to 20 second between campaigns, this means there can be up to a minute of downtime during a failover scenario, generally that's fine for the Provisioner.

If a Provisioner was on standby and becomes leader it will immediately perform a discovery to pick up any nodes ready for provisioning.

To enable the Choria Broker must be of the kind described above in `Preparing a Broker Environment` and [Choria Streams](https://choria.io/docs/streams) must be enabled.

Setting `leader_election: true` in the Provisioner configuration will enable campaigns, when this is set the Provisioners will start in the Paused mode.

#### Writing the helper

Your helper can be written in any language, it will receive JSON on its STDIN and should return JSON on its STDOUT. It should complete within 10 seconds and could be called concurrently.

The input is in the format:

```json
{
	"identity": "dev1.devco.net",
	"csr": {
		"csr": "-----BEGIN CERTIFICATE REQUEST-----....-----END CERTIFICATE REQUEST-----",
        "public_key": "-----BEGIN PUBLIC KEY-----....-----END PUBLIC KEY-----",
		"ssldir": "/path/to/ssldir"
	},
	"inventory": "{\"agents\":[\"choria_provision\",\"choria_util\",\"discovery\",\"rpcutil\"],\"facts\":{},\"classes\":[],\"version\":\"0.0.0\",\"data_plugins\":[],\"main_collective\":\"provisioning\",\"collectives\":[\"provisioning\"]}"
}
```

The CSR structure will be empty when the PKI feature is not enabled, the `inventory` is the output from `rpcutil#inventory`, you'll be mainly interested in the `facts` hash I suspect. The data is JSON encoded. The `public_key` entry is available since Choria 0.23.0.

The output from your script should be like this:

```json
{
  "defer": false,
  "msg": "Optional message indicating success or why things are failing",
  "certificate": "-----BEGIN CERTIFICATE-----......-----END CERTIFICATE-----",
  "ca": "-----BEGIN CERTIFICATE-----......-----END CERTIFICATE-----",
  "opa_policies": {
    "default.rego": "....."
  },
  "action_policies": {
    "rpcutil.policy": "....."
  },
  "configuration": {
    "plugin.choria.server.provision": "false",
    "identity": "node1.example.net"
  }
}
```

If you set the `ProvisionModeDefault` compile time flag to `"true"` then you must set `plugin.choria.server.provision` to `"false"` else provisioning will fail to avoid a endless loop.

If you want to defer the provisioning - like perhaps you are still waiting for facts to be generated - set `defer` to true and supply a reason in `msg` which will be logged. The node will be tried again on the following cycle.

If you do not care for PKI then do not set `certificate` and `ca`.

The `configuration` contains the config in key value pairs where everything should be strings, this gets written directly into the Choria Server configuration.

#### Sample CFSSL Helper

Here's a sample helper that support enrolling nodes into a CFSSL CA, the CA is assumed to be running and listening on `localhost:8888`.  We use this helper in production and can provision 1000 nodes in under a minute using it - including enrolling in the CA.

For this to work place the CA bundle in `/etc/choria-provisioner/ca.pem`.

```ruby
#!/opt/puppetlabs/puppet/bin/ruby

require "json"
require "open3"

input = STDIN.read

request = JSON.parse(input)
request["inventory"] = JSON.parse(request["inventory"])

reply = {
  "defer" => false,
  "msg" => "",
  "certificate" => "",
  "ca" => "",
  "configuration" => {}
}

identity = request["identity"]
brokers = "broker.example.net:4222"
registerinterval = "300"
registration_data = "/etc/node/metadata.json"

# PKI is optional, if you do enable it in the provisioner this code will kick in
if request["csr"] && request["csr"]["csr"]
  begin
    out, err, status = Open3.capture3("/path/to/cfssl sign -remote http://localhost:8888 -", :stdin_data => request["csr"]["csr"])
    if status.exitstatus > 0 || err != ""
      raise("Could not sign certificate: %s" % err)
    end

    signed = JSON.parse(out)

    if signed["cert"]
      reply["ca"] = File.read("/etc/choria-provisioner/ca.pem")
      reply["certificate"] = signed["cert"]
    else
      raise("Did not received a signed certificate from cfssl")
    end

    ssldir = request["csr"]["ssldir"]

    reply["configuration"].merge!(
      "plugin.security.provider" => "file",
      "plugin.security.file.certificate" => File.join(ssldir, "certificate.pem"),
      "plugin.security.file.key" => File.join(ssldir, "private.pem"),
      "plugin.security.file.ca" => File.join(ssldir, "ca.pem"),
      "plugin.security.file.cache" => File.join(ssldir, "cache")
    )
  rescue
    reply["defer"] = true
    reply["msg"] = "cfssl integration failed: %s: %s" % [$!.class, $!.to_s]
  end
end

reply["configuration"].merge!(
  "identity" => identity,
  "registerinterval" => registerinterval,
  "plugin.choria.middleware_hosts" => brokers,
  "plugin.choria.registration.file_content.data" => registration_data,
  # include any other settings you wish to set
)

puts reply.to_json
```

#### Configuring the provisioner

The provisioner takes a YAML or JSON configuration file, something like:

```yaml
---
# how many concurrent provisions can be run
workers: 4

# how frequently to start the cycle in go duration format
interval: 5m

# where to log
logfile: "/var/log/provisioner.log"

# loglevel - debug, info, warn, error
loglevel: info

# path to your helper script
helper: /usr/local/bin/provision

# the token you compiled into choria or stored into the jwt
token: toomanysecrets

# a site name exposed stats for differentiating different clusters
site: testing

# sets a custom lifecycle component to listen on for events that trigger provisioning
# not compatible with leader election based HA
lifecycle_component: acme_provisioning

# Certificate patterns that should never be signed from CSRs, these are ones choria
# set aside as client only certificates and someone might configure a node to obtain
# a signed cert otherwise.  When not set below is the default value
cert_deny_list:
  - "\.choria$"
  - "\.mcollective$"
  - "\.privileged.choria$"
  - "\.privileged.mcollective$"

# if not 0 then /metrics will be prometheus metrics
monitor_port: 9999

# provisioning server will connect with this password to connect
# the same one configured on the broker with plugin.choria.network.provisioning.client_password
broker_provisioning_password: provS3cret

# a public cert that will be used to verify the JWT on the node is one we know and signed by us
# the same one configured on the broker with plugin.choria.network.provisioning.signer_cert
jwt_verify_cert: /etc/choria_provisioner/jwt-signer.pem

features:
  # enables fetching of the CSR
  pki: true
  # fetches the provisioning jwt and verify it against jwt_verify_cert
  jwt: false
```

A choria client configuration should be made in `/etc/choria-provisioner/choria.cfg`, it looks like a normal choria client config and would support SRV and all the usual settings.

#### Statistics

The daemon keeps a number of Prometheus format stats and will expose it in `/metrics` if the `monitor_port` settings is over 0.

|Statistic|Descriptions|
|---------|------------|
|choria_provisioner_rpc_time|How long each RPC request takes|
|choria_provisioner_helper_time|How long the helper takes to run|
|choria_provisioner_discovered|How many nodes are discovered using the broadcast discovery|
|choria_provisioner_event_discovered|How many nodes were discovered due to events being fired about them|
|choria_provisioner_discover_cycles|How many discovery cycles were ran|
|choria_provisioner_rpc_errors|How many times a RPC request failed|
|choria_provisioner_helper_errors|How many times the helper failed to run|
|choria_provisioner_discovery_errors|How many times the discovery failed to run|
|choria_provisioner_provision_errors|How many times provisioning failed|
|choria_provisioner_paused|1 when the backplane paused operations, 0 otherwise|
|choria_provisioner_busy_workers|How many workers are busy processing servers|
|choria_provisioner_provisioned|Host many nodes were successfully provisioned|

A Grafana dashboard is included in `dashboard.json` that produce a set of graphs like here:

![](provisioner-dashboard.png)

#### Packages

RPMs are hosted in the Choria yum repository for el7 and 8 64bit systems, packages are called `choria-provisioner`:

```ini
[choria_release]
name=Choria Orchestrator Releases
mirrorlist=http://mirrorlists.choria.io/yum/release/el/$releasever/$basearch.txt
enabled=True
gpgcheck=True
repo_gpgcheck=True
gpgkey=https://choria.io/RELEASE-GPG-KEY
metadata_expire=300
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
sslverify=True
```

We also publish nightly builds in the following repository:

```ini
[choria_nightly]
name=Choria Orchestrator Nightly
mirrorlist=http://mirrorlists.choria.io//yum/nightly/el/$releasever/$basearch.txt
enabled=True
gpgcheck=True
repo_gpgcheck=True
gpgkey=https://choria.io/NIGHTLY-GPG-KEY
metadata_expire=300
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
sslverify=True
```
