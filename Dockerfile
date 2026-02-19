FROM cgr.dev/chainguard/static:latest
COPY pulumi-exporter /usr/bin/pulumi-exporter
ENTRYPOINT ["/usr/bin/pulumi-exporter"]
