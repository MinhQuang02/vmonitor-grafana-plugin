# GreenNode vMonitor data source for Grafana

Query GreenNode vMonitor Platform metrics and logs from Grafana — build dashboards on both, and drive Grafana alerting from either.

- Metric queries that mirror the vMonitor widget builder: metric, statistic, dimension filters, group by, and the Rank / Rollup / Rate / Timeshift / Reduce functions.
- Expression mode for arithmetic across sibling queries, e.g. `A / B * 100`.
- Log queries against a vMonitor log project, rendered as log lines, a table, or a count over time.
- Alerting on metrics and on log counts, covering vMonitor's Metric Alarm and a log alarm's Frequency and Flatline types.
- Template variables: `metrics()`, `dimensions(metric)`, `dimensionValues(metric, key)`, `infrastructure(product[, region])` and `regions(product)`.
- Bundled dashboards for Host, vServer, vLB, vDB (relational, Kafka and PostgreSQL cluster), vStorage (project and bucket) and vBackup.

## Requirements

- Grafana 12.3.0 or later
- `sudo` access on the Grafana host
- A GreenNode IAM service account key pair (client id + client secret)

The release archive ships backend binaries for Linux (amd64, arm64, arm), macOS (amd64, arm64) and Windows (amd64).

## Installation

### 1. Download the release archive

Grab the latest ZIP from the [releases page](https://github.com/GreenNodeHub/vmonitor-grafana-plugin/releases):

```bash
wget https://github.com/GreenNodeHub/vmonitor-grafana-plugin/releases/download/v1.0.0/greennode-vmonitor-datasource-1.0.0.zip
```

### 2. Unpack it into the Grafana plugin directory

```bash
sudo mkdir -p /var/lib/grafana/plugins
sudo unzip greennode-vmonitor-datasource-1.0.0.zip \
  -d /var/lib/grafana/plugins/greennode-vmonitor-datasource
```

`/var/lib/grafana/plugins` is the default for the Debian and RPM packages, but it only exists once Grafana has run at least once — hence the `mkdir`. If your path differs, check the `plugins` key under `[paths]`:

```bash
grep -n '^;*[[:space:]]*plugins[[:space:]]*=' /etc/grafana/grafana.ini
```

> **Upgrading?** Remove the old directory first with `sudo rm -rf /var/lib/grafana/plugins/greennode-vmonitor-datasource`.

### 3. Set ownership

Grafana runs as the `grafana` user and must be able to read the plugin and execute its backend binary. Hand it the whole plugin directory — if `mkdir` just created it as root, Grafana also loses the ability to install any other plugin there:

```bash
sudo chown -R grafana:grafana /var/lib/grafana/plugins
```

### 4. Allow the plugin to load (unsigned builds only)

Grafana only loads the plugin once its id is listed in the `[plugins]` section of `/etc/grafana/grafana.ini`:

```ini
[plugins]
allow_loading_unsigned_plugins = greennode-vmonitor-datasource
```

Edit the file by hand, or set the line in place:

```bash
sudo sed -i 's/^;*[[:space:]]*allow_loading_unsigned_plugins[[:space:]]*=.*/allow_loading_unsigned_plugins = greennode-vmonitor-datasource/' /etc/grafana/grafana.ini
```

The value is the plugin id, and the list is comma-separated if you allow more than one. To read the id back from the plugin itself:

```bash
grep '"id"' /var/lib/grafana/plugins/greennode-vmonitor-datasource/plugin.json
```

### 5. Restart Grafana and verify

```bash
sudo systemctl restart grafana-server
```

A successful load ends with `Plugin registered`:

```
level=warn msg="Plugin is unsigned" id=greennode-vmonitor-datasource
level=warn msg="Permitting unsigned plugin. This is not recommended" pluginId=greennode-vmonitor-datasource
level=info msg="Plugin registered" pluginId=greennode-vmonitor-datasource
```

The two warnings are expected on an unsigned build. The plugin now appears under **Administration → Plugins and data → Plugins**. Continue with [Configuration](#configuration).

## Configuration

### 1. Create a service account key pair

The plugin authenticates as an IAM service account. Follow these steps in the [GreenNode IAM console](https://iam.console.greennode.ai/):

**a. Create a service account**

1. Go to **Service Accounts → Create service account**
2. Enter a name (e.g. `grafana-vmonitor-readonly`) and click **Create**

**b. Attach the vMonitor read policy**

1. Open the service account you just created
2. Go to the **Policies** tab → **Attach policies**
3. Search for and select **`vMonitorMetricReadOnlyAccess`**
4. If you also need log queries, additionally attach **`vMonitorLogReadOnlyAccess`**
5. Click **Attach**

**c. Generate a key pair**

1. Go to the **Keys** tab → **Create key**
2. Copy and save both values immediately — the secret is shown only once:
   - **Access Key ID** → use as **Client ID** in Grafana
   - **Secret Access Key** → use as **Client Secret** in Grafana

### 2. Add the data source

Go to **Administration → Plugins and data → Plugins**, search for **GreenNode vMonitor**, open it, and click **Add new data source**.

Fill in the key pair from step 1:

| Field | Value |
|-------|-------|
| **API URL** | vMonitor API base URL. Leave blank for the default. |
| **Client ID** | The service account key id. |
| **Client Secret** | The service account secret. Stored encrypted, and only ever read by the backend. |
| **Token URL** | IAM token-exchange endpoint. Leave blank for the default. |

Only **Client ID** and **Client Secret** are required — leave both URLs empty and the plugin uses the public GreenNode endpoints.

Click **Save & test**. The backend exchanges the key pair for a Bearer token and caches it until it expires; the secret never reaches the browser.

### 3. Import the bundled dashboards

Go to **Connections → Data sources**, open the data source you just created, switch to its **Dashboards** tab, and click **Import** on each dashboard you want.

They then appear under Grafana's **Dashboards** section, ready to use.

Each dashboard populates its resource picker from the vMonitor infrastructure endpoints rather than from metric dimensions, so it lists every resource of a product — including those without detailed monitoring enabled. Host dashboards are the exception: their metrics come from the in-VM agent.

### 4. Query ad hoc

Use **Explore** to query any metric or log project directly — handy for resources with detailed monitoring enabled, and for building a query before putting it on a dashboard.

## Querying metrics

When creating or editing a panel, select the **GreenNode vMonitor** data source and choose mode **Metric**.

### Basic fields

| Field | Description | Example |
|-------|-------------|---------|
| **Metric** | The metric name to query | `net.if.in.bytes_sec` |
| **Statistic** | Aggregation function | `avg` |
| **Filter by** | Dimension filters — multiple filters are combined with AND | `host = web-server-01` |
| **Group by** | Split results into separate series by dimension values | `host` |
| **Alias** | Display name template for each series | `{{host}}` |

### Statistics

| Statistic | Description |
|-----------|-------------|
| `avg` | Average |
| `min` | Minimum |
| `max` | Maximum |
| `sum` | Sum |
| `count` | Number of data points |
| `rate_1s` | Rate of change per second |
| `rate_1m` | Rate of change per minute |
| `rate_5m` | Rate of change per 5 minutes |

> The `rate_*` statistics do **not** support the Rollup, Rate, or Timeshift functions.

### Functions

Click **Add Function** to apply one or more transformations:

**Rollup** — aggregate over a fixed time window (avg / min / max / sum / count, value + unit). Not available for `rate_*`.

**Rate** — convert a cumulative counter to a per-unit rate (`per_second`, `per_minute`, `per_hour`). Not available for `rate_*`.

**Timeshift** — overlay the same metric from an earlier time period (value + unit). Not available for `rate_*`.

**Rank** — keep only the top N (`topk`) or bottom N (`bottomk`) series at each point. Excluded series have gaps — add **Resample** before **Reduce** in alert rules.

**Reduce** — collapse a series to a single value for Stat or Top List panels (`last`, `avg`, `min`, `max`, `sum`).

### Examples

**Top 5 vServers by CPU usage**
- Metric: `cpu.usage_user`, Statistic: `avg`
- Filter: `product = vserver`
- Group by: `resource_id`, Rank: topk 5
- Alias: `{{resource_id}}`

**Network throughput vs. last week**
- Query A: `net.if.in.bytes_sec`, statistic `avg`
- Query B: same metric with Timeshift 1 week

## Expression mode

Add a query and set **Query Type** to **Expression**, then reference sibling queries by their letter:

| Expression | Meaning |
|------------|---------|
| `A / B * 100` | Percentage |
| `(A + B) / 2` | Average of two metrics |

> Expression mode does not support `rate_*` statistics.

## Querying logs

Select mode **Logs** in the query editor, then choose a **Log project** (only ACTIVE projects are listed).

### Output formats

| Format | Description | Use with |
|--------|-------------|----------|
| `logs` | Log lines with a count histogram; each line is expandable | Explore, Logs panel |
| `table` | One row per log entry, one column per field | Table panel |
| `timeseries` | Log count over time | Graph panel, Alerting |

> In Alert mode the format is automatically set to `timeseries`.

### Filter operators

| Operator | Input | Meaning |
|----------|-------|---------|
| `is` | 1 value | Exact match |
| `is not` | 1 value | Does not match |
| `is one of` | Multiple values | Matches any of the values |
| `is not one of` | Multiple values | Matches none of the values |
| `is between` | from, to | In range [from, to) |
| `is not between` | from, to | Outside the range |
| `exists` | — | Field is present |
| `does not exist` | — | Field is absent |

Grafana variables are supported in filter values (e.g. `$selected_host`).

### Group by (timeseries only)

Split the log count into separate series by selecting one or more aggregatable (`keyword`) fields. Each unique combination of field values becomes a series; maximum 20 values per field.

> In alert rules, groups with no matching logs do not produce an alert instance.

## Template variables

| Query | Returns |
|-------|---------|
| `metrics()` | All available metric names |
| `dimensions(metricName)` | All dimension keys for a metric |
| `dimensionValues(metricName, key)` | All values for a dimension key |
| `infrastructure(product[, region])` | Resources for a product type |
| `regions(product)` | Available regions for a region-aware product |

**Supported infrastructure products:** `host`, `vserver`, `vlb`, `vstorage`, `vstorage-bucket`, `vdb`, `vdb-kafka`, `vdb-cluster`, `vbackup`.

**Example — dynamic vServer dashboard:**
- Variable `server`: `infrastructure(vserver)`
- Filter in query: `resource_id = $server`

A filter whose value is `*` is dropped, as vMonitor does for a variable set to All. Give a dimension variable **Include All option** with custom all value `*` to get the same behaviour.

## Alerting

**On metrics:** Build a metric query, then add **Reduce** + **Threshold** in the alert rule. If the query uses Rank, add **Resample** before Reduce to handle gaps in excluded series.

**On log count (Frequency):** Use a log query with format `timeseries`, add filters, then attach **Reduce** + **Threshold**. Quiet periods are reported as `0` rather than missing data, so a rule can also fire when logs stop (`count = 0` threshold).

**Log group by in alerts:** Each group value creates a separate alert instance. Groups with no logs during an evaluation window do not produce an instance.

## Build from source

**Prerequisites** — Node.js 22 or later (see `.nvmrc`), the Go toolchain named in `go.mod`, and `mage` (`go install github.com/magefile/mage@latest`).

```bash
git clone https://github.com/GreenNodeHub/vmonitor-grafana-plugin.git
cd vmonitor-grafana-plugin

npm ci
npm run build   # frontend, into dist/
mage -v         # backend binaries for every platform, into dist/
```

`mage -l` lists the other targets — build a single platform with, for example, `mage build:linux`.

`dist/` now holds the same layout as the release archive. Package it with:

```bash
cd dist && zip -r ../greennode-vmonitor-datasource-1.0.0.zip . && cd ..
```

## Development

Run a throwaway Grafana with `dist/` already mounted and the data source pre-provisioned — no manual install needed:

```bash
export VM_CLIENT_ID=... VM_CLIENT_SECRET=...
npm run server
```

Grafana comes up on <http://localhost:3000>. Build the frontend and backend at least once first, or `dist/` will be empty.

To install your build into an existing Grafana instead, copy `dist/` into the plugin directory and follow steps 3–5 of [Installation](#installation):

```bash
sudo rm -rf /var/lib/grafana/plugins/greennode-vmonitor-datasource
sudo cp -r dist /var/lib/grafana/plugins/greennode-vmonitor-datasource
```

Checks before opening a merge request:

```bash
npm run typecheck
npm run lint
go vet ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full workflow.

## Troubleshooting

| Symptom | Cause |
|---------|-------|
| Plugin missing from **Plugins** | Unsigned build without the allowance — see step 4, then `journalctl -u grafana-server \| grep -i signature` |
| Logs report the plugin directory was skipped | Ownership — rerun the `chown` from step 3 |
| Plugin unavailable when running a query | The backend binary could not start. Check that `plugin.json` sits directly in the plugin directory, not in a nested folder |
| Grafana reports a version error | The plugin requires Grafana 12.3.0 or later |
| `401 Unauthorized` on Save & test | Wrong Client ID or Secret — re-check your IAM key pair |
| `connection refused` on Save & test | Wrong API URL or network issue — verify the API URL and connectivity |
| Query returns no data | Check the time range, dimension filters, and that the IAM key has read permission for the resource |
| Log project dropdown is empty | IAM key lacks log read permission, or the project is not in ACTIVE billing status |

## GreenNode documentation

- [vMonitor Platform](https://docs.greennode.ai/vmonitor)
- [What is vMonitor Platform Metric?](https://docs.greennode.ai/vmonitor/vmonitor-platform-la-gi/vmonitor-platform-metric-la-gi)
- [What is vMonitor Platform Log?](https://docs.greennode.ai/vmonitor/vmonitor-platform-la-gi/vmonitor-platform-log-la-gi)
- [Metric query](https://docs.greennode.ai/vmonitor/dashboards/dashboard/query/metric-query) and [Log query](https://docs.greennode.ai/vmonitor/dashboards/dashboard/query/log-query)
- [Metric Alarm](https://docs.greennode.ai/vmonitor/dashboards/alarm/metric-alarm) and [Log Alarm](https://docs.greennode.ai/vmonitor/dashboards/alarm/log-alarm)

## License

Apache-2.0. See [LICENSE](LICENSE).
