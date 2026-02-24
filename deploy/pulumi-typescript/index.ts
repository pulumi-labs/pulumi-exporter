import * as pulumi from "@pulumi/pulumi";
import * as kubernetes from "@pulumi/kubernetes";

const config = new pulumi.Config();
const pulumiAccessToken = config.require("pulumi-access-token");
const organizations = config.require("organizations");
const collectInterval = config.require("collect-interval");
const maxConcurrency = config.requireNumber("max-concurrency");
const orgList = organizations.split(",");
const namespaceResource = new kubernetes.core.v1.Namespace("namespace", {metadata: {
    name: "pulumi-exporter",
}});
const secretResource = new kubernetes.core.v1.Secret("secret", {
    metadata: {
        name: "pulumi-credentials",
        namespace: namespaceResource.metadata.apply(metadata => metadata.name),
    },
    type: "Opaque",
    stringData: {
        "access-token": pulumiAccessToken,
    },
});
const release = new kubernetes.helm.v3.Release("release", {
    chart: "oci://ghcr.io/pulumi-labs/charts/pulumi-exporter",
    version: "0.1.1",
    namespace: namespaceResource.metadata.apply(metadata => metadata.name),
    values: {
        existingSecret: secretResource.metadata.apply(metadata => metadata.name),
        pulumiOrganizations: orgList,
        collectInterval: collectInterval,
        maxConcurrency: maxConcurrency,
        otlp: {
            endpoint: "localhost:4318",
            protocol: "http/protobuf",
            insecure: true,
        },
    },
});
export const namespace = namespaceResource.metadata.apply(metadata => metadata.name);
export const releaseName = release.name;
export const releaseVersion = release.version;
