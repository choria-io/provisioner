FROM almalinux:8

ARG REPO="https://yum.eu.choria.io/release/el/release.repo"

WORKDIR /etc/choria-provisioner/

RUN curl -s "${REPO}" > /etc/yum.repos.d/choria.repo && \
    yum -y install choria-provisioner ruby nc procps-ng openssl && \
    yum -y update && \
    yum -y clean all

RUN groupadd --gid 2048 choria && \
    useradd -c "Choria Orchestrator - choria.io" -m --uid 2048 --gid 2048 choria && \
    chown -R choria:choria /etc/choria-provisioner

USER choria
ENV USER=choria

ENTRYPOINT ["/usr/sbin/choria-provisioner"]

CMD ["run", "--config=/etc/choria-provisioner/choria-provisioner.yaml", "--choria-config=/etc/choria-provisioner/client.cfg"]
