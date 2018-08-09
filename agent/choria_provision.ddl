metadata :name => "choria_provision",
         :description => "Choria Provisioner",
         :author => "R.I.Pienaar <rip@devco.net>",
         :license => "Apache-2.0",
         :version => "0.0.1",
         :url => "https://choria.io",
         :timeout => 2

action "gencsr", :description => "Request a CSR from the Choria Server" do
    display :always

    input :token,
          :prompt  => "Token",
          :description => "Authentication token to pass to the server",
          :type => :string,
          :validation => ".",
          :optional => true,
          :default => "",
          :maxlength => 128

    input :cn,
          :prompt => "Common Name",
          :description => "The certificate Common Name to place in the CSR",
          :type => :string,
          :validation => "^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$",
          :optional => true,
          :maxlength => 80

    input :C,
          :prompt => "Country",
          :description => "Country Code",
          :type => :string,
          :validation => "^[A-Z]{2}$",
          :optional => true,
          :maxlength => 2

    input :L,
          :prompt => "Locality",
          :description => "Locality or municipality (such as city or town name)",
          :type => :string,
          :validation => "^[\\w\\s-]+$",
          :optional => true,
          :maxlength => 50

    input :O,
          :prompt => "Organization",
          :description => "Organization",
          :type => :string,
          :validation => "^[\\w\\s-]+$",
          :optional => true,
          :maxlength => 50

    input :OU,
          :prompt => "Organizational Unit",
          :description => "Organizational Unit",
          :type => :string,
          :validation => "^[\\w\\s-]+$",
          :optional => true,
          :maxlength => 50

    input :ST,
          :prompt => "State",
          :description => "State",
          :type => :string,
          :validation => "^[\\w\\s-]+$",
          :optional => true,
          :maxlength => 50

    output :csr,
           :description => "PEM text block for the CSR",
           :display_as => "CSR"

    output :ssldir,
           :description => "SSL directory as determined by the server",
           :display_as => "SSL Dir"
end

action "configure", :description => "Configure the Choria Server" do
    input :token,
          :prompt  => "Token",
          :description => "Authentication token to pass to the server",
          :type => :string,
          :validation => ".",
          :optional => true,
          :default => "",
          :maxlength => 128

    input :config,
          :prompt  => "Configuration",
          :description => "The configuration to apply to this node",
          :type => :string,
          :validation => "^{.+}$",
          :optional => false,
          :maxlength => 2048

    input :certificate,
          :prompt  => "Certificate",
          :description => "PEM text block for the certificate",
          :type => :string,
          :validation => "^-----BEGIN CERTIFICATE-----",
          :optional => true,
          :maxlength => 10240

    input :ca,
          :prompt  => "CA Bundle",
          :description => "PEM text block for the CA",
          :type => :string,
          :validation => "^-----BEGIN CERTIFICATE-----",
          :optional => true,
          :maxlength => 10240

    input :ssldir,
          :prompt  => "SSL Dir",
          :description => "Directory for storing the certificate in",
          :type => :string,
          :validation => ".",
          :optional => true,
          :maxlength => 500

    output :message,
           :description => "Status message from the Provisioner",
           :display_as => "Message"
end

action "restart", :description => "Restart the Choria Server" do
    input :token,
          :prompt  => "Token",
          :description => "Authentication token to pass to the server",
          :type => :string,
          :validation => ".",
          :optional => true,
          :default => "",
          :maxlength => 128

    input :splay,
          :prompt  => "Splay time",
          :description => "The configuration to apply to this node",
          :type => :number,
          :optional => true

    output :message,
           :description => "Status message from the Provisioner",
           :display_as => "Message"
end

action "reprovision", :description => "Reenable provision mode in a running Choria Server" do
    display :always

    input :token,
          :prompt  => "Token",
          :description => "Authentication token to pass to the server",
          :type => :string,
          :validation => ".",
          :optional => true,
          :default => "",
          :maxlength => 128

    output :message,
           :description => "Status message from the Provisioner",
           :display_as => "Message"
end
