FROM --platform=amd64 gcr.io/distroless/static-debian11:nonroot
USER 20000:20000
ADD --chmod=555 build/bin/external-dns-hetzner-webhook-amd64 /opt/external-dns-hetzner-webhook/app

ENTRYPOINT ["/opt/external-dns-hetzner-webhook/app"]