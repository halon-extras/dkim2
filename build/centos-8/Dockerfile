FROM rockylinux:8
LABEL org.opencontainers.image.authors="support@halon.io"

ARG TARGETARCH
ARG TARGETOS

ARG HALON_REPO_USER
ARG HALON_REPO_PASS

RUN echo -n "CENTOS8" > /OSRELEASE.txt

RUN yum update -y && yum install -y epel-release cmake3 make gcc-c++ rpm-build curl git openssl-devel

RUN <<EOF
set -eu
cd /tmp
case "$TARGETARCH" in
  amd64)
    SHA256=031f088e5d955bab8657ede27ad4e3bc5b7c1ba281f05f245bcc304f327c987a
    ;;
  arm64)
    SHA256=a290581cfe4fe28ddd737dde3095f3dbeb7f2e4065cab4eae44dfc53b760c2f7
    ;;
esac
curl -O -L https://go.dev/dl/go1.26.1.${TARGETOS}-${TARGETARCH}.tar.gz
echo "${SHA256} go1.26.1.${TARGETOS}-${TARGETARCH}.tar.gz" | sha256sum -c
tar -C /usr/local -xzf go1.26.1.${TARGETOS}-${TARGETARCH}.tar.gz
rm go1.26.1.${TARGETOS}-${TARGETARCH}.tar.gz
EOF
RUN /usr/local/go/bin/go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.10.0
RUN /usr/local/go/bin/go install github.com/google/go-licenses/v2@v2.0.1
ENV PATH="$PATH:/usr/local/go/bin:/root/go/bin"

RUN echo "[halon]" >> /etc/yum.repos.d/Halon.repo
RUN echo "name=Halon repository" >> /etc/yum.repos.d/Halon.repo
RUN echo "baseurl=https://${HALON_REPO_USER}:${HALON_REPO_PASS}@repo.halon.io/dists/rhel8/${TARGETARCH}/stable" >> /etc/yum.repos.d/Halon.repo
RUN echo "gpgcheck=1" >> /etc/yum.repos.d/Halon.repo
RUN echo "gpgkey=https://raw.githubusercontent.com/halon/changelog/master/pgp-keys/7F0A73B5.asc" >> /etc/yum.repos.d/Halon.repo
RUN echo "enabled=1" >> /etc/yum.repos.d/Halon.repo
RUN yum install -y halon-6.10.0

COPY build.sh /build.sh
CMD ["/build.sh"]
