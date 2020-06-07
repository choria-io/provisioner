FROM centos:7

ENV PROV_WORKERS=4
ENV PROV_INTERVAL=5m
ENV PROV_LOGLEVEL=info
ENV PROV_INSECURE=false
ENV PROV_SITE=docker
ENV PROV_FEATURES_PKI=false
ENV PROV_FEATURES_JWT=false

WORKDIR /etc/choria-provisioner/

RUN curl -s https://packagecloud.io/install/repositories/choria/release/script.rpm.sh | bash && \
    yum -y update && \
    yum -y install choria-provisioner && \
    rm /etc/choria-provisioner/choria-provisioner.yaml && \
    yum -y clean all

ADD docker/choria-provisioner.yaml.template /etc/choria-provisioner/
ADD docker/client.cfg.template /etc/choria-provisioner/
ADD docker/entrypoint.sh /

CMD "/entrypoint.sh"
