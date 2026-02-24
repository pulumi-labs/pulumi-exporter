import pulumi
import pulumi_kubernetes as kubernetes

config = pulumi.Config()
pulumi_access_token = config.require("pulumi-access-token")
organizations = config.require("organizations")
collect_interval = config.require("collect-interval")
max_concurrency = config.require_int("max-concurrency")
org_list = organizations.split(",")
namespace_resource = kubernetes.core.v1.Namespace("namespace", metadata={
    "name": "pulumi-exporter",
})
secret_resource = kubernetes.core.v1.Secret("secret",
    metadata={
        "name": "pulumi-credentials",
        "namespace": namespace_resource.metadata.name,
    },
    type="Opaque",
    string_data={
        "access-token": pulumi_access_token,
    })
release = kubernetes.helm.v3.Release("release",
    chart="oci://ghcr.io/pulumi-labs/charts/pulumi-exporter",
    version="0.1.1",
    namespace=namespace_resource.metadata.name,
    values={
        "existingSecret": secret_resource.metadata.name,
        "pulumiOrganizations": org_list,
        "collectInterval": collect_interval,
        "maxConcurrency": max_concurrency,
        "otlp": {
            "endpoint": "localhost:4318",
            "protocol": "http/protobuf",
            "insecure": True,
        },
    })
pulumi.export("namespace", namespace_resource.metadata.name)
pulumi.export("releaseName", release.name)
pulumi.export("releaseVersion", release.version)
