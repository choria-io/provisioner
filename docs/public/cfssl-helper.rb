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
