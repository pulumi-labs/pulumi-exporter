# Metrics Reference

## Stack Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `pulumi_stack_resource_count` | Gauge | `org`, `project`, `stack` | Number of resources in a stack |
| `pulumi_update_duration_seconds` | Histogram | `org`, `project`, `stack`, `kind`, `result` | Duration of stack updates (seconds) |
| `pulumi_update_total` | Counter | `org`, `project`, `stack`, `kind`, `result` | Total number of stack updates |
| `pulumi_update_resource_changes` | Counter | `org`, `project`, `stack`, `kind`, `operation` | Resource changes per update |
| `pulumi_deployment_status` | Gauge | `org`, `status` | Deployments by status |
| `pulumi_stack_last_update_timestamp` | Gauge | `org`, `project`, `stack` | Unix timestamp of last update |

## Organization Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `pulumi_org_member_count` | Gauge | `org` | Number of organization members |
| `pulumi_org_team_count` | Gauge | `org` | Number of teams |
| `pulumi_org_environment_count` | Gauge | `org` | Number of ESC environments |
| `pulumi_org_policy_group_count` | Gauge | `org` | Number of policy groups |
| `pulumi_org_policy_pack_count` | Gauge | `org` | Number of policy packs |
| `pulumi_org_policy_violations` | Gauge | `org`, `level`, `kind` | Policy violations by severity and type |
| `pulumi_org_neo_task_count` | Gauge | `org`, `status` | Pulumi Neo AI tasks by status |

## Compliance Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `pulumi_org_policy_total` | Gauge | `org` | Total number of policies |
| `pulumi_org_policy_with_issues` | Gauge | `org` | Number of policies with issues |
| `pulumi_org_governed_resources_total` | Gauge | `org` | Total governed resources |
| `pulumi_org_governed_resources_with_issues` | Gauge | `org` | Governed resources with issues |

## Label Values

| Label | Values |
|-------|--------|
| `kind` (updates) | `update`, `preview`, `destroy`, `refresh`, `import` |
| `result` | `succeeded`, `failed`, `in-progress` |
| `operation` | `create`, `update`, `delete`, `same`, `replace` |
| `status` (deployments) | `running`, `succeeded`, `failed`, `not-started`, `accepted` |
| `status` (Neo tasks) | `idle`, `running` |
| `level` (violations) | `advisory`, `mandatory`, `disabled` |
| `kind` (violations) | `preventative`, `audit` |

## Histogram Buckets

`pulumi_update_duration_seconds` uses bucket boundaries tuned for IaC operations:

```
5s, 10s, 30s, 1m, 2m, 5m, 10m, 30m
```
