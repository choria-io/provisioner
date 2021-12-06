FROM centos:7

WORKDIR /etc/choria-provisioner/

RUN curl -s https://packagecloud.io/install/repositories/choria/release/script.rpm.sh | bash && \
    yum -y update && \
    yum -y install choria-provisioner ruby && \
    yum -y clean all

RUN groupadd --gid 2048 choria && \
    useradd -c "Choria Orchestrator - choria.io" -m --uid 2048 --gid 2048 choria && \
    chown -R choria:choria /etc/choria-provisioner

USER choria

ENTRYPOINT ["/usr/sbin/choria-provisioner"]

CMD ["--config choria-provisioner.yaml", "--choria-config client.cfg"]
