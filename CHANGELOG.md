# Changelog

## 1.0.0 (Unreleased)

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
