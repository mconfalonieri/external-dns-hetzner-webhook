FROM gcr.io/distroless/static-debian11:nonroot
LABEL org.opencontainers.image.description="OCI image for external-dns-hetzner-webhook"
ARG TARGETPLATFORM
USER 20000:20000
ADD --chmod=555 ${TARGETPLATFORM}/external-dns-hetzner-webhook /opt/external-dns-hetzner-webhook/bin/webhook
ENTRYPOINT ["/opt/external-dns-hetzner-webhook/bin/webhook"]
