#!/usr/bin/ruby

require "json"
require "yaml"
require "base64"
require "net/http"
require "openssl"

BROKERS = "nats://broker.choria.local:4222"


def parse_input
  input = STDIN.read

  begin
    File.open("/tmp/request.json", "w") {|f| f.write(input)}
  rescue Exception
  end

  request = JSON.parse(input)
  request["inventory"] = JSON.parse(request["inventory"])

  request
end

def validate!(request, reply)
  if request["identity"] && request["identity"].length == 0
    reply["msg"] = "No identity received in request"
    reply["defer"] = true
    return false
  end

  unless request["ed25519_pubkey"]
    reply["msg"] = "No ed15519 public key received"
    reply["defer"] = true
    return false
  end

  unless request["ed25519_pubkey"]
    reply["msg"] = "No ed15519 directory received"
    reply["defer"] = true
    return false
  end

  if request["ed25519_pubkey"]["directory"].length == 0
    reply["msg"] = "No ed15519 directory received"
    reply["defer"] = true
    return false
  end

  true
end

def publish_reply(reply)
  begin
    File.open("/tmp/reply.json", "w") {|f| f.write(reply.to_json)}
  rescue Exception
  end

  puts reply.to_json
end

def publish_reply!(reply)
  publish_reply(reply)
  exit
end

def set_config!(request, reply)
  reply["configuration"].merge!(
    "identity" => request["identity"],
    "loglevel" => "info",
    "plugin.choria.server.provision" => "false",
    "plugin.choria.middleware_hosts" => BROKERS,
    "rpcauthorization" => "0",
    "plugin.choria.status_file_path" => "/tmp/status.json",
    "plugin.choria.submission.spool" => "/tmp/submission",
    "plugin.security.issuer.names" => "choria",
    "plugin.security.issuer.choria.public" => "e72cba5268b34627b75c5ceae9449ad16d62f15f862c30d4e0e7d2588e2e6259",
    "plugin.security.provider" => "choria",
    "plugin.security.choria.token_file" => File.join(request["ed25519_pubkey"]["directory"], "server.jwt"),
    "plugin.security.choria.seed_file" => File.join(request["ed25519_pubkey"]["directory"], "server.seed"),
    "plugin.choria.machine.store" => "/etc/choria/machine",
  )

  reply["server_claims"].merge!(
    "exp" => 5*60*60*24*365,
    "permissions" => {
      "streams" => true,
      "submission" => true
    }
  )
end

reply = {
  "defer" => false,
  "msg" => "",
  "certificate" => "",
  "ca" => "",
  "configuration" => {},
  "server_claims" => {}
}

begin
  request = parse_input

  reply["msg"] = "Validating"
  unless validate!(request, reply)
    publish_reply!(reply)
  end

  set_config!(request, reply)

  reply["msg"] = "Done"
  publish_reply!(reply)
rescue SystemExit
rescue Exception
  reply["msg"] = "Unexpected failure during provisioning: %s: %s" % [$!.class, $!.to_s]
  reply["defer"] = true
  publish_reply!(reply)
end
