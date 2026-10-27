# Changelog

## 1.1.0 (2026-10-27)

- Bundled dashboard for vDB PostgreSQL clusters, "vMonitor / vDB (Relational /
  MemoryStore Cluster)", matching the vMonitor default dashboard.
- The vDB relational dashboard is renamed "vMonitor / vDB (Relational /
  MemoryStore Single-Node)", after the vDB tabs in the vMonitor portal.
- A dimension filter whose value is `*` is dropped, as vMonitor does for a
  variable set to All.

## 1.0.0 (2026-08-05)

Initial release.

- Metric queries against GreenNode vMonitor Platform Metric, with dimension
  filters, group by, and the Rank / Rollup / Rate / Timeshift / Reduce
  functions.
- Expression mode, combining sibling queries by refId (`A / B * 100`).
- Log queries against vMonitor Platform Log, rendered as log lines, a table, or
  a count over time, with group by.
- Grafana alerting on both metrics and logs. Quiet periods report `0` rather
  than missing data, so a rule can fire on logs stopping as well as spiking.
- Template variables: `metrics()`, `dimensions()`, `dimensionValues()`,
  `infrastructure()` and `regions()`.
- Bundled dashboards for Host, vServer, vLB, vDB (relational and Kafka),
  vStorage (project and bucket) and vBackup.
