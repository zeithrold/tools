FROM ubuntu:24.04

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates dbus gnome-keyring libsecret-tools \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home --uid 10001 --shell /bin/sh zt-test

COPY run-test.sh /usr/local/bin/run-keystore-test
USER 10001:10001
ENV HOME=/home/zt-test
ENV XDG_RUNTIME_DIR=/tmp/zt-runtime
ENTRYPOINT ["sh", "/usr/local/bin/run-keystore-test"]
