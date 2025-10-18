FROM gcr.io/distroless/static-debian11:nonroot
ARG TARGETPLATFORM
USER 20000:20000
ADD --chmod=555 ${TARGETPLATFORM}/external-dns-hetzner-webhook /opt/external-dns-hetzner-webhook/app

ENTRYPOINT ["/opt/external-dns-hetzner-webhook/app"]
