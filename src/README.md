# GreenNode vMonitor

Query **vMonitor Platform Metric** and **vMonitor Platform Log** from
[GreenNode](https://docs.greennode.ai/vmonitor) inside Grafana — build
dashboards on both, and drive Grafana alerting from either.

## Overview

vMonitor Platform is GreenNode's solution for collecting, analysing and alerting
on metric, log and synthetic data from GreenNode, other clouds, or on-premise
environments. It groups into three services: **Metric**, **Log** and
**Synthetic**.

This data source brings the first two into Grafana. Synthetic tests are not
covered — monitor those in the GreenNode portal.

### Metrics

**vMonitor Metric** continuously tracks the performance of every resource in
your system in real time — GreenNode products such as vServer, vLB, vDB,
vStorage and vBackup, plus any server running the vMonitor Metric Agent
(GreenNode, another cloud, or on-premise).

The query editor follows the vMonitor widget builder, so a query you know from
the portal transfers directly:

| vMonitor | In this plugin |
| --- | --- |
| Metric | **Metric** — searchable list of the metrics available to your account |
| Filter | **Dimension filters** — narrow by `host`, `device`, `resource_id`, … |
| Statistics | **Statistic** — `avg`, `count`, `max`, `min`, `sum`, plus per-second/minute/5-minute rates |
| Group by | **Group by** — one series per dimension value |
| Limit to | **Rank** — Top *k* / Bottom *k* series |
| Alias | **Alias** — falls back to `statistic:metric(dimensions)` |

Data granularity is derived from the panel's time range, the same way vMonitor
computes it from the selected range. **Rollup**, **Rate**, **Timeshift** and
**Reduce** functions are available on top of the query, and **Expression** mode
combines other queries in the panel by refId, e.g. `A / B * 100`.

### Logs

**vMonitor Log** collects log data from servers, applications and devices in
real time so you can store, search and analyse it in one place.

Pick a **log project**, then build a **search log** condition with the same
operators as the portal — `is`, `is not`, `is one of`, `is not one of`,
`is between`, `is not between`, `exists`, `does not exist` — with field names
and values suggested from that project's mappings. Results render three ways:

- **Logs** — a log-volume histogram over the log lines, each expandable to its
  full set of fields. Works with Grafana's Explore, including live tailing of
  the time range.
- **Table** — one row per log record, one column per field you pick.
- **Time series** — the **count** of matching log records over time, optionally
  split by a **Group by** field.

Only the `count` statistic is supported today. The other vMonitor log
aggregations — `count unique`, `sum`, `avg`, `min`, `max`, `percentiles` over a
numeric field — are not yet implemented here.

### Alerting

Both metrics and logs feed Grafana's built-in alerting, covering the same ground
as vMonitor's **Metric Alarm** and **Log Alarm**.

For logs, a rule counts the records matching your search over the rule's time
range. Add a **Reduce** expression — use *Sum* to total every record in the
range — and then a **Threshold**:

| vMonitor alarm type | Grafana equivalent |
| --- | --- |
| Frequency — at least X events in Y | Reduce *Sum* → Threshold **is above** X, evaluated over Y |
| Flatline — fewer than X events in Y | Reduce *Sum* → Threshold **is below** X, evaluated over Y |
| Log aggregation | not supported yet |

Flatline works because quiet periods are reported as `0` rather than as missing
data, so a rule can fire on logs *stopping* just as well as on logs spiking.
A **Group by** field raises one alert instance per host, service, or any other
aggregatable field — mirroring vMonitor's single group-by on a log alarm.

One caveat: a group that produces no logs at all never appears in the response,
so it cannot alert. To catch a source going silent entirely, add a separate rule
with no Group by.

### Dashboard variables

`metrics()`, `dimensions(metric)`, `dimensionValues(metric, key)`,
`infrastructure(product[, region])` and `regions(product)`.

## Requirements

- Grafana 12.3.0 or later
- A GreenNode IAM access key (client id and client secret) with permission to
  read vMonitor metrics, and — to query logs — the vMonitor log projects

## Getting started

1. Add the data source and fill in **Client ID** and **Client Secret** from your
   GreenNode IAM access key. **API URL** and **Token URL** already point at the
   public GreenNode endpoints; the log search API is derived from the API URL.
2. Click **Save & test**. The backend exchanges the access key for a Bearer
   token; the secret is stored encrypted and never reaches the browser.
3. Import a bundled dashboard from the plugin's **Dashboards** tab — one exists
   for Host, vServer, vLB, vDB (relational, Kafka and PostgreSQL cluster),
   vStorage (project and bucket) and vBackup.

Each dashboard lists every resource of its product, including resources without
detailed monitoring enabled, and defaults to that product's platform-level
metrics.

## Learn more

- [vMonitor Platform documentation](https://docs.greennode.ai/vmonitor)
- [What is vMonitor Platform Metric?](https://docs.greennode.ai/vmonitor/vmonitor-platform-la-gi/vmonitor-platform-metric-la-gi)
- [What is vMonitor Platform Log?](https://docs.greennode.ai/vmonitor/vmonitor-platform-la-gi/vmonitor-platform-log-la-gi)
