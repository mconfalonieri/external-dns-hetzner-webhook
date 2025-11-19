# Dockerfile used for local test builds.
FROM --platform=${TARGETARCH} gcr.io/distroless/static-debian11:nonroot
ARG TARGETARCH
LABEL org.opencontainers.image.description="TEST Hetzner DNS webhook for external-dns"
USER 20000:20000
ADD --chmod=555 build/bin/external-dns-hetzner-webhook-${TARGETARCH} /opt/external-dns-hetzner-webhook/bin/webhook
ENTRYPOINT ["/opt/external-dns-hetzner-webhook/bin/webhook"]
