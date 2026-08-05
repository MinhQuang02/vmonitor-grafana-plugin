# GreenNode vMonitor Data Source for Grafana

## Table of Contents

1. [Overview](#1-overview)
2. [Requirements](#2-requirements)
3. [Installation](#3-installation)
4. [Configuration](#4-configuration)
5. [Querying Metrics](#5-querying-metrics)
6. [Querying Logs](#6-querying-logs)
7. [Expression Mode](#7-expression-mode)
8. [Template Variables](#8-template-variables)
9. [Alerting](#9-alerting)
10. [Bundled Dashboards](#10-bundled-dashboards)
11. [Troubleshooting](#11-troubleshooting)

---

## 1. Overview

**GreenNode vMonitor Data Source** is a Grafana plugin for querying **Metric** and **Log** data from the GreenNode vMonitor Platform directly inside Grafana.

### Key Features

| Feature | Description |
|---------|-------------|
| **Metric queries** | Query metrics with statistics (avg, min, max, sum, count, rate), dimension filters, group by, and the Rollup / Rate / Timeshift / Rank / Reduce functions |
| **Expression mode** | Evaluate arithmetic across sibling queries (e.g. `A / B * 100`) |
| **Log queries** | Search logs from a log project, rendered as log lines, a table, or a count over time |
| **Alerting** | Alert on metrics and on the count of matching logs |
| **Template variables** | Dynamic dashboard variables backed by vMonitor data |
| **Bundled dashboards** | 8 pre-built dashboards for Host, vServer, vLB, vDB, vStorage and vBackup |

---

## 2. Requirements

- **Grafana** >= 12.3.0
- A **GreenNode IAM Access Key** (Client ID + Client Secret) with vMonitor read permission

---

## 3. Installation

### Step 1: Extract the plugin archive

Extract `greennode-vmonitor-datasource-1.0.0.zip` into the Grafana plugins directory:

```bash
# Linux / macOS
unzip greennode-vmonitor-datasource-1.0.0.zip -d /var/lib/grafana/plugins/

# Or wherever your Grafana plugins directory is configured
unzip greennode-vmonitor-datasource-1.0.0.zip -d /path/to/grafana/plugins/
```

The resulting directory structure should look like:

```
/var/lib/grafana/plugins/
└── greennode-vmonitor-datasource/
    ├── plugin.json
    ├── module.js
    ├── gpx_green_node_v_monitor_linux_amd64
    ├── gpx_green_node_v_monitor_linux_arm64
    ├── ...
    └── dashboards/
```

### Step 2: Allow the unsigned plugin to load

Because this plugin is not signed with Grafana, add the following to `grafana.ini`:

```ini
[plugins]
allow_loading_unsigned_plugins = greennode-vmonitor-datasource
```

Common locations for `grafana.ini`:
- **Linux**: `/etc/grafana/grafana.ini`
- **macOS (Homebrew)**: `/opt/homebrew/etc/grafana/grafana.ini`
- **Docker**: mount a custom config file or set the environment variable `GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=greennode-vmonitor-datasource`

### Step 3: Restart Grafana

```bash
# systemd
sudo systemctl restart grafana-server

# Docker
docker restart <container_name>
```

### Step 4: Verify the plugin is loaded

Go to **Grafana → Administration → Plugins** and search for "vMonitor" — the plugin should show status **Installed**.

---

## 4. Configuration

### Add a new data source

1. Go to **Grafana → Connections → Data sources → Add new data source**
2. Search for "GreenNode vMonitor" and select it

### Configuration fields

| Field | Description | Default |
|-------|-------------|---------|
| **API URL** | vMonitor Platform API base URL | `https://vmonitorapis.vngcloud.vn/vmonitor-api/api` |
| **Client ID** | IAM Access Key ID | _(required)_ |
| **Client Secret** | IAM Access Key Secret — stored encrypted, never sent to the browser | _(required)_ |
| **Token URL** | IAM token exchange endpoint. Leave blank to use the default. | `https://iamapis.vngcloud.vn/accounts-api/v2/auth/token` |

> **Security note:** The Client Secret is stored encrypted and is only ever read by the Go backend. The browser never receives it.

### Obtaining an IAM Access Key

1. Log in to the **GreenNode IAM Portal**
2. Go to **Access Keys → Create Access Key**
3. Assign **vMonitor read** permission to the access key
4. Copy the **Access Key ID** (Client ID) and **Access Key Secret** (Client Secret)

### Test the connection

Click **Save & Test** — a success message confirms the credentials are valid.

---

## 5. Querying Metrics

### 5.1. Basic fields

When creating or editing a panel, select the **GreenNode vMonitor** data source and choose mode **Metric**.

| Field | Description | Example |
|-------|-------------|---------|
| **Metric** | The metric name to query | `net.if.in.bytes_sec` |
| **Statistic** | Aggregation function | `avg` |
| **Filter by** | Dimension filters (multiple allowed, all applied as AND) | `host = web-server-01` |
| **Group by** | Split results into series by dimension values | `host` |
| **Alias** | Display name template for each series | `{{host}}` |

### 5.2. Statistics

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

> **Note:** The `rate_*` statistics do **not** support the Rollup, Rate, or Timeshift functions.

### 5.3. Dimension filters (Filter by)

- Click **+ Filter** to add a new filter row
- Select the **key** (dimension name) from the suggestion list
- Enter the **value** to filter on — Grafana variables are supported (e.g. `$host`)
- Multiple filters are combined with **AND**

**Example:** Filter to a specific server:
```
host = $selected_host
```

### 5.4. Group by

- Select one or more dimension keys from the suggestion list
- Each unique combination of dimension values becomes a separate series
- Use **Alias** with dimension placeholders to name series automatically: `{{host}} - {{interface}}`

### 5.5. Functions

Click **Add Function** to apply one or more data transformation functions:

#### Rollup — aggregate over a time window

Collapses multiple data points into one value per window:

| Field | Values |
|-------|--------|
| Type | avg, min, max, sum, count |
| Value | Positive integer |
| Unit | minutes, hours, days, weeks |

**Example:** Rollup avg 5 minutes — shows the 5-minute average instead of individual points.

> Not available for `rate_*` statistics.

#### Rate — convert to rate of change

Converts a cumulative counter metric into a per-unit rate:

| Value | Description |
|-------|-------------|
| `per_second` | Rate per second |
| `per_minute` | Rate per minute |
| `per_hour` | Rate per hour |

> Not available for `rate_*` statistics.

#### Timeshift — shift the time range

Overlays the same metric from an earlier time period for comparison:

| Field | Values |
|-------|--------|
| Value | Positive integer |
| Unit | minutes, hours, days, weeks |

**Example:** Timeshift 1 week — compare this week's traffic against last week's.

> Not available for `rate_*` statistics.

#### Rank — keep top N or bottom N series

Retains only the N highest or lowest series at each point in time:

| Field | Values |
|-------|--------|
| Type | `topk` (top N) or `bottomk` (bottom N) |
| N | 5, 10, 20, 25, 50, or custom |

**Example:** Top 10 hosts by CPU usage.

> Excluded series have gaps in their data. When used in an alert rule, add a **Resample** step before **Reduce** to handle the gaps.

#### Reduce — collapse to a single value

Reduces an entire series to one value per query (for Stat or Top List panels):

| Value | Description |
|-------|-------------|
| `last` | Last value |
| `avg` | Average |
| `min` | Minimum |
| `max` | Maximum |
| `sum` | Sum |

### 5.6. Examples

**Example 1: Top 5 vServers by CPU usage**
- Metric: `cpu.usage_user`
- Statistic: `avg`
- Filter: `product = vserver`
- Group by: `resource_id`
- Function: Rank → topk, 5
- Alias: `{{resource_id}}`

**Example 2: Network throughput compared to last week**
- Query A: `net.if.in.bytes_sec`, statistic `avg`
- Query B: `net.if.in.bytes_sec`, statistic `avg`, Function: Timeshift 1 week

---

## 6. Querying Logs

### 6.1. Select a log project

After selecting mode **Logs** in the query editor:

1. Choose a **Log project** from the dropdown — only projects with `billing_status = ACTIVE` are listed
2. Once a project is selected, filter and column options become available

### 6.2. Output format

| Format | Description | Use with |
|--------|-------------|----------|
| **logs** | Log lines with a count histogram; each line is expandable to view the full JSON | Explore, Logs panel |
| **table** | One row per log entry, one column per field | Table panel |
| **timeseries** | Log count over time | Graph panel, Alerting |

> In **Alert** mode, the format is automatically set to `timeseries`.

### 6.3. Result limit

Select the maximum number of log entries to return: **50, 100, 200, 500**. Default is 100.

> Does not apply to the `timeseries` format.

### 6.4. Log filters

Click **+ Filter** to add a filter condition. Each filter row has:

- **Field**: The log field name (with auto-suggestions)
- **Operator**: Comparison operator
- **Value**: The value to match

#### Supported operators

| Operator | Input | Meaning |
|----------|-------|---------|
| `is` | 1 value | Exact match |
| `is not` | 1 value | Does not match |
| `is one of` | Multiple values | Matches any of the values |
| `is not one of` | Multiple values | Matches none of the values |
| `is between` | 2 values (from, to) | In range [from, to) |
| `is not between` | 2 values (from, to) | Outside range |
| `exists` | _(none)_ | Field is present |
| `does not exist` | _(none)_ | Field is absent |

**Example filters:**
```
level   is          ERROR
host    is one of   web-01, web-02
latency is between  100, 500
```

> Grafana variables are supported in filter values, e.g. `$selected_host`.

### 6.5. Columns

Choose which log fields to display as columns (applies to `logs` and `table` formats):

- Click **+ Columns** to add a column group
- Select one or more fields from the suggestion list
- Use the **arrow** buttons to reorder columns
- Use the **trash** button to remove a column group

### 6.6. Group by (timeseries only)

When format is `timeseries`, split the log count into separate series by field values:

- Select one or more **aggregatable** fields (type `keyword`)
- Each unique combination of field values becomes a separate series
- Maximum 20 values per field

> **Alert note:** Each group value creates a separate alert instance. Groups with no matching logs during an evaluation period will **not** fire an alert — account for this when designing rules.

### 6.7. Examples

**Example 1: View application error logs**
- Format: logs
- Limit: 100
- Filter: `level is ERROR`
- Columns: timestamp, message, host, service

**Example 2: Error count per service over time**
- Format: timeseries
- Filter: `level is ERROR`
- Group by: `service`

**Example 3: Alert on HTTP 5xx errors**
- Format: timeseries (automatic in alert mode)
- Filter: `http_status is between 500, 600`

---

## 7. Expression Mode

Expression mode evaluates arithmetic across metric queries within the same panel.

### How to use

1. Create at least two metric queries (e.g. Query **A** and Query **B**)
2. Add a new query and set **Query Type** to **Expression**
3. Enter an expression using the letter references of the other queries

### Expression syntax

| Expression | Meaning |
|------------|---------|
| `A + B` | Addition |
| `A - B` | Subtraction |
| `A * B` | Multiplication |
| `A / B` | Division |
| `A / B * 100` | Percentage |
| `(A + B) / 2` | Average of two metrics |

### Example

**Calculate the ratio of system CPU to total CPU:**
- Query A: `cpu.usage_system`, statistic `avg`
- Query B: `cpu.usage_total`, statistic `avg`
- Expression C: `A / B * 100`

> **Limitation:** Expression mode does not support `rate_*` statistics.

---

## 8. Template Variables

The plugin provides five query functions for creating dynamic dashboard variables.

### Creating a template variable

1. Go to **Dashboard Settings → Variables → Add variable**
2. Set **Type** to **Query** and **Data source** to **GreenNode vMonitor**
3. Enter one of the query functions below in the **Query** field

### Supported functions

#### `metrics()`

Returns all available metric names.

```
metrics()
```

Use this to let users pick a metric from a dropdown.

---

#### `dimensions(metricName)`

Returns all dimension keys for a given metric.

```
dimensions(cpu.usage_user)
```

---

#### `dimensionValues(metricName, dimensionKey)`

Returns all values for a dimension key of a specific metric.

```
dimensionValues(cpu.usage_user, host)
```

**Common use case:** Create a `$host` variable to filter a dashboard by server:
- Variable name: `host`
- Query: `dimensionValues(cpu.usage_user, host)`
- Enable **Multi-value** and **Include All option** to allow selecting multiple hosts

---

#### `infrastructure(product[, region])`

Returns a list of resources for the given product type, optionally filtered by region.

**Supported products:**

| Product | Resource type |
|---------|---------------|
| `host` | Host monitoring agent |
| `vserver` | Virtual Server |
| `vlb` | Load Balancer |
| `vstorage` | vStorage Project |
| `vstorage-bucket` | vStorage Bucket (region-aware) |
| `vdb` | vDB Relational |
| `vdb-kafka` | vDB Kafka |
| `vdb-cluster` | vDB Cluster |
| `vbackup` | vBackup |

```
infrastructure(vserver)
infrastructure(vstorage-bucket, HCM03)
```

---

#### `regions(product)`

Returns the available regions for a region-aware product such as `vstorage-bucket`.

```
regions(vstorage-bucket)
```

---

### Example: Dynamic vServer dashboard

**Variables:**

| Variable | Query |
|----------|-------|
| `server` | `infrastructure(vserver)` |
| `metric` | `metrics()` |

**Usage in queries:**
- Filter by: `resource_id = $server`
- Metric: `$metric`

---

## 9. Alerting

The plugin supports Grafana alert rules on both metric and log queries.

### 9.1. Alerting on metrics

1. Build a metric query as usual (mode: Metric)
2. Switch to the **Alert** tab in the panel editor
3. Configure a **Reduce** step (e.g. last, avg) and a **Threshold**

**Important when using Rank:**
- Series excluded by Rank have gaps in their data
- Add a **Resample** step before **Reduce** in the alert rule to fill those gaps

### 9.2. Alerting on log count

This is equivalent to vMonitor's **Log Alarm (Frequency)** type:

1. Create a log query with format: `timeseries`
2. Add the appropriate filters (e.g. `level is ERROR`)
3. Configure a **Reduce + Threshold** in the alert rule

**Alert on logs going quiet (Flatline):**
- Periods with no matching logs are reported as `0` rather than as missing data
- Set a threshold of `count = 0` to fire when no logs are seen within an evaluation window

### 9.3. Alerting with log Group by

When a log query uses **Group by**:
- Each unique group value creates a separate **alert instance**
- Grafana evaluates and labels each instance independently
- Groups with no matching logs during an evaluation period do not produce an instance — factor this into your rule design

---

## 10. Bundled Dashboards

The plugin ships with 8 pre-built dashboards.

**Option 1: Import from the plugin**
1. Go to **Dashboards → New → Import**
2. Select the **GreenNode vMonitor** data source
3. Choose the dashboard you want to use

**Option 2: Automatic provisioning**
Dashboards are loaded automatically when the plugin is installed via provisioning.

### Dashboard list

| Dashboard | Coverage |
|-----------|----------|
| **vMonitor / Host** | Host agent metrics: CPU, RAM, Disk, Network for any monitored server |
| **vMonitor / vServer** | Platform-level metrics for GreenNode Virtual Servers |
| **vMonitor / vLB (Load Balancer)** | Traffic and health metrics for Load Balancers |
| **vMonitor / vDB (Relational)** | Performance metrics for relational databases (MySQL, PostgreSQL, …) |
| **vMonitor / vDB (Kafka)** | Cluster and topic metrics for Kafka |
| **vMonitor / vStorage** | Throughput and capacity metrics for vStorage projects |
| **vMonitor / vStorage (Bucket)** | Per-bucket metrics and analytics |
| **vMonitor / vBackup** | Backup job status and performance |

### How the resource picker works

The bundled dashboards use the `infrastructure(product)` template variable to populate their resource dropdowns. This list comes from the vMonitor infrastructure API and includes **all** resources of a product type — including those without detailed monitoring enabled.

- **Host dashboards:** Metrics come from the in-VM agent and carry no `product` dimension.
- **All other dashboards:** Queries include a `product:<name>` dimension filter, which makes platform-level metrics of any resource reachable regardless of monitoring status.

---

## 11. Troubleshooting

### Plugin does not appear in the plugin list

- Verify the directory structure: `greennode-vmonitor-datasource/plugin.json` must exist inside the plugins directory
- Verify `grafana.ini` contains `allow_loading_unsigned_plugins = greennode-vmonitor-datasource`
- Check the Grafana logs: `journalctl -u grafana-server` or `/var/log/grafana/grafana.log`

### Save & Test returns an authentication error

| Error | Likely cause | Resolution |
|-------|-------------|------------|
| `401 Unauthorized` | Wrong Client ID or Secret | Re-check your IAM Access Key |
| `connection refused` | Wrong API URL or network issue | Verify the API URL and network connectivity |
| `token exchange failed` | Wrong Token URL | Correct the Token URL or leave it blank to use the default |

### Query returns no data

- Check the time range — the metric may not have data in the selected window
- Check dimension filters — the filter value may not match any data
- Remove filters one at a time to isolate the issue
- Verify the IAM key has read permission for the target resource's metrics or logs

### Log project dropdown is empty

- Verify the IAM key has read permission for the log project
- Verify the log project is in ACTIVE billing status

### Alert rule does not fire

- For metric queries using Rank: add a **Resample** step before **Reduce** in the alert rule
- For log queries using Group by: remember that groups with no logs will not generate an alert instance
- Verify the Reduce function matches the shape of the data being evaluated

---

## Development

### Backend

Build the plugin backend for all supported platforms:

```bash
mage -v
```

Run `mage -l` to list all available targets.

### Frontend

```bash
npm install
npm run dev     # watch mode
npm run build   # production build
```

### Run locally

```bash
export VM_CLIENT_ID=... VM_CLIENT_SECRET=...
npm run server
```

This starts Grafana on <http://localhost:3000> with `dist/` mounted as the plugin and the data source provisioned from `provisioning/datasources/datasources.yml`. Build both backend and frontend at least once first.

### Checks

```bash
npm run typecheck
npm run lint
go vet ./...
```

## License

Apache-2.0. See [LICENSE](LICENSE).
