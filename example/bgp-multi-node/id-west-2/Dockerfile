FROM alpine:3.22 AS tunny

ARG TUNNY_REPO=Aldiwildan77/tunny

RUN apk add --no-cache curl tar

RUN set -eux; \
    URL="$(curl -fsSL \
      "https://api.github.com/repos/${TUNNY_REPO}/releases/latest" \
      | sed -n 's/.*"browser_download_url": "\(.*linux.*amd64.*\.tar\.gz\)".*/\1/p' \
      | head -1)"; \
    echo "Downloading: ${URL}"; \
    test -n "${URL}"; \
    curl -fsSL "${URL}" -o /tmp/tunny.tar.gz; \
    mkdir -p /tmp/tunny; \
    tar -xzf /tmp/tunny.tar.gz -C /tmp/tunny; \
    find /tmp/tunny -type f -name tunny -exec cp {} /tunny \;

FROM frr:10.6.2

COPY --from=tunny /tunny /usr/local/bin/tunny

COPY entrypoint.sh /entrypoint.sh
COPY frr.conf /etc/frr/frr.conf
COPY daemons /etc/frr/daemons
COPY vtysh.conf /etc/frr/vtysh.conf
COPY tunny.yaml /etc/tunny/config.yaml

RUN chmod +x /entrypoint.sh /usr/local/bin/tunny

ENTRYPOINT ["/entrypoint.sh"]