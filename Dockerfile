FROM gcr.io/distroless/static-debian11:nonroot

USER 20000:20000
ADD --chmod=555 external-dns-ionos-webhook /opt/external-dns-ionos-webhook/app

ENTRYPOINT ["/opt/external-dns-ionos-webhook/app"]