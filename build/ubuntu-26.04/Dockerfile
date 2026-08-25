FROM ubuntu:26.04
LABEL org.opencontainers.image.authors="support@halon.io"

ARG TARGETARCH
ARG TARGETOS

ARG HALON_REPO_USER
ARG HALON_REPO_PASS

RUN echo -n "UBU2604" > /OSRELEASE.txt

RUN apt-get update && apt-get install -y wget gnupg apt-transport-https build-essential cmake git curl file git libssl-dev libarchive-tools

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
LC_ALL=C.UTF-8 LANG=C.UTF-8 bsdtar -C /usr/local -xzf go1.26.1.${TARGETOS}-${TARGETARCH}.tar.gz
rm go1.26.1.${TARGETOS}-${TARGETARCH}.tar.gz
EOF
RUN /usr/local/go/bin/go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.10.0
RUN /usr/local/go/bin/go install github.com/google/go-licenses/v2@v2.0.1
ENV PATH="$PATH:/usr/local/go/bin:/root/go/bin"

RUN wget -qO- https://raw.githubusercontent.com/halon/pgp-keys/master/pgp-keys/7F0A73B5.gpg > /usr/share/keyrings/halon-archive-keyring.gpg
RUN echo "deb [arch=${TARGETARCH} signed-by=/usr/share/keyrings/halon-archive-keyring.gpg] https://repo.halon.io/ resolute stable" > /etc/apt/sources.list.d/halon.list
RUN echo "machine repo.halon.io login ${HALON_REPO_USER} password ${HALON_REPO_PASS}" >> /etc/apt/auth.conf
RUN apt-get update && apt-get install -y halon-smtpd=6.11.0

COPY build.sh /build.sh
CMD ["/build.sh"]
