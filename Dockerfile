FROM centos:7

WORKDIR /etc/choria-provisioner/

RUN curl -s https://packagecloud.io/install/repositories/choria/release/script.rpm.sh | bash && \
    yum -y update && \
    yum -y install choria-provisioner ruby && \
    yum -y clean all

CMD ["/usr/sbin/choria-provisioner", "--config choria-provisioner.yaml", "--choria-config client.cfg"]
