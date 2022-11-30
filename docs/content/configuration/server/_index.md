+++
title = "Server Setup"
toc = true
weight = 10
pre = "<b>1.1. </b>"
+++

The Choria Server has the required RPC Agent embedded and is ready to be used in a Provisioner managed network but it is disabled by default.  Custom binaries can enable provisioning at compile time for an always-on experience.

Provisioning is enabled in the Open Source server by means of a JWT token that you create and place on the server. The JWT token holds all of the information the server needs to find it's provisioning server and will present that token also to the provisioning server for authentication.

The token is signed using a trusted private key, the provisioner will only provision nodes presenting a trusted key.

## Credentials

If you are using a x509 based setup you really just need any RSA Key pair to sign the JWT with:

```nohighlight
$ openssl genrsa -out provisioning-jwt-signer-key.pem 2048
$ openssl rsa -in provisioning-jwt-signer-key.pem -outform PEM -pubout -out provisioning-jwt-signer.pem
```

The public file will be placed on all your brokers to [enable provisioning](../broker).

For the Organization Issuer based deploys you would sign it using your Issuer.

## Creating the JWT

Choria JWTs are creating using the `choria jwt` command, for provisioning specifically `choria jwt provisioning`.

Creating the JWT:

```nohighlight
$ choria jwt provisioning.jwt provisioning-jwt-signer-key.pem --srv choria.example.net --token toomanysecrets
Saved token to provisioning.jwt, use 'choria jwt view provisioning.jwt' to view it

$ choria jwt provisioning.jwt
Unvalidated Provisioning Token x.jwt

                         Token: *****
                        Secure: true
                    SRV Domain: choria.example.net
       Provisioning by default: false
               Standard Claims: {
                                  "purpose": "choria_provisioning",
                                  "iss": "Choria Tokens Package v0.26.2",
                                  "sub": "choria_provisioning",
                                  "nbf": 1669805752,
                                  "iat": 1669805752,
                                  "jti": "2c99227346f641bbba34faf0a6991d05"
                                }
```


Here we create a provisioning.jwt that will instruct Choria to look for `_choria-provisioner._tcp.choria.example.net` SRV records to find the server to connect to.

Other options can be set for example to hard code provisioning URLs, username and passwords and more.

{{% notice secondary "Version Hint" code-branch %}}
This table is correct for Choria 0.27.0 and newer which is due to ship early 2023. Review `--help` for your version.
{{% /notice %}}

| Option            | Description                                                                                                           | Default      | Required |
|-------------------|-----------------------------------------------------------------------------------------------------------------------|--------------|----------|
| `signing-key`     | The token must be signed using a private key, this can be a file with either RSA or ed25519 private key.              |              | yes      |
| `--[no]-insecure` | During provisioning the protocol security system cannot be active as no private credentials exist, this disables that | since 0.27.0 | no       |
| `--token`         | A basic shared secret that the provisioner must present to perform certain actions                                    |              | no       |
| `--urls`          | A static, comma seperated, list of servers to connect to for provisioning                                             |              | no       |
| `--srv`           | A domain name to query for SRV records                                                                                |              | no       |
| `--default`       | Enables provisioning by default.  Else requires `plugin.choria.server.provision` to be set                            | false        | no       |
| `--registration`  | Path to a file to publish as registration data while in provisioning mode                                             |              | no       |
| `--facts`         | Path to a file to use as fact source data while in provisioning mode                                                  |              | no       |
| `--username`      | A NATS user to use when connecting to the broker                                                                      |              | no       |
| `--password`      | Password to use when connecting to the broker                                                                         |              | no       |
| `--extensions`    | Additional free-form data to embed in the JWT as JSON text                                                            |              | no       |
| `--org`           | The Organization this server belongs to                                                                               | `choria`     | yes      |
| `--vault`         | Use Hashicorp Vault for signing the token, the `signing-key` is then the name of the vault key                        | false        | no       |
| `--protocol-v2`   | Enables the `choria` security system and version 2 protocol. Set for Org Issuer based networks                        | false        | no       |

When this file is placed in `/etc/choria/provisioning.jwt` and Choria starts without a configuration it will provision via these settings.

Choria also support provisioning plugins to resolve this information dynamically but this requires custom binaries and should in general be avoided.

## Confirming

Without the JWT in place Provisioning is not enabled:

```nohighlight
$ choria buildinfo
...
Server Settings:

    Provisioning Target Resolver: Choria JWT Resolver
           Supports Provisioning: false
           Provisioning JWT file: /etc/choria/provisioning.jwt
...
```


We can see it will look for the `/etc/choria/provisioning.jwt` file, lets move our newly created file there and try again:

```nohighlight
$ sudo msg provisioning.jwt /etc/choria/provisioning.jwt
$ choria buildinfo
...
Server Settings:

    Provisioning Target Resolver: Choria JWT Resolver
           Supports Provisioning: true
           Provisioning JWT file: /etc/choria/provisioning.jwt
              Provisioning Token: *****
            Provisioning Default: false
                Provisioning TLS: true
      Default Provisioning Agent: true
         Provisioning SRV Domain: choria.example.net
...
```

Now provisioning is on with the settings we provided in the token.
