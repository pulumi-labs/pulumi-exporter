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
| `pulumi_org_neo_tokens_used_current_month` | Gauge | `org` | Neo tokens consumed by tasks created in the current calendar month (matches the Pulumi Cloud billing-period usage) |
| `pulumi_org_neo_tokens_used_total` | Gauge | `org` | Total Neo tokens consumed across all tasks (lifetime) |
| `pulumi_org_neo_token_budget_consumed` | Gauge | `org` | Neo tokens consumed in the current budget window |
| `pulumi_org_neo_token_budget_allowance` | Gauge | `org` | Effective Neo token allowance for the current window (base plus active bonus) |
| `pulumi_org_neo_token_budget_exhausted` | Gauge | `org` | Whether the Neo token budget for the current window is exhausted (`1`) or not (`0`) |

The `neo_token_budget_*` metrics report the effective budget for the identity behind the access token. When an org-level cap and a per-member cap both apply, Pulumi Cloud returns whichever is closer to exhaustion. Organizations without a Neo allowance return no budget, so these series are absent.

## Compliance Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `pulumi_org_policy_total` | Gauge | `org` | Total number of policies |
| `pulumi_org_policy_with_issues` | Gauge | `org` | Number of policies with issues |
| `pulumi_org_governed_resources_total` | Gauge | `org` | Total governed resources |
| `pulumi_org_governed_resources_with_issues` | Gauge | `org` | Governed resources with issues |

## Custom Agent Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `pulumi_org_agent_count` | Gauge | `org`, `paused` | Number of Pulumi Neo Custom Agents by paused state |
| `pulumi_org_agent_runs_started` | Gauge | `org`, `agent` | Total runs a Custom Agent has started (lifetime, organization-wide) |
| `pulumi_org_agent_runs_running` | Gauge | `org`, `agent` | Runs of a Custom Agent that are currently running |
| `pulumi_org_agent_automation_count` | Gauge | `org`, `kind`, `paused`, `last_run_status` | Number of Custom Agent automations |
| `pulumi_org_agent_automation_next_run_timestamp` | Gauge | `org`, `automation` | Unix timestamp of the next scheduled run of an automation |

Run counts are aggregated across the organization, not scoped to the identity behind the access token. `pulumi_org_agent_runs_started` only grows, so `increase(pulumi_org_agent_runs_started[1h])` gives the runs started per hour. Automations without a scheduled next run, such as triggered ones, have no `next_run_timestamp` series. Organizations without Custom Agents enabled record none of these metrics.

## Label Values

| Label | Values |
|-------|--------|
| `kind` (updates) | `update`, `preview`, `destroy`, `refresh`, `import` |
| `result` | `succeeded`, `failed`, `in-progress` |
| `operation` | `create`, `update`, `delete`, `same`, `replace` |
| `status` (deployments) | `running`, `succeeded`, `failed`, `not-started`, `accepted`, `skipped` |
| `status` (Neo tasks) | `idle`, `running` |
| `level` (violations) | `advisory`, `mandatory`, `disabled` |
| `kind` (violations) | `preventative`, `audit` |
| `paused` (agents, automations) | `true`, `false` |
| `kind` (automations) | `scheduled`, `triggered` |
| `last_run_status` (automations) | the API's last run status (for example `succeeded`, `failed`), or `none` if it has not run yet |
| `agent` | Custom Agent name |
| `automation` | Automation name |

## Histogram Buckets

`pulumi_update_duration_seconds` uses bucket boundaries tuned for IaC operations:

```
5s, 10s, 30s, 1m, 2m, 5m, 10m, 30m
```
