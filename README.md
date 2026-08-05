# Hướng dẫn sử dụng GreenNode vMonitor Datasource Plugin cho Grafana

## Mục lục

1. [Tổng quan](#1-tổng-quan)
2. [Yêu cầu](#2-yêu-cầu)
3. [Cài đặt plugin](#3-cài-đặt-plugin)
4. [Cấu hình datasource](#4-cấu-hình-datasource)
5. [Query Metric](#5-query-metric)
6. [Query Logs](#6-query-logs)
7. [Biểu thức toán học (Expression Mode)](#7-biểu-thức-toán-học-expression-mode)
8. [Template Variables](#8-template-variables)
9. [Cảnh báo (Alerting)](#9-cảnh-báo-alerting)
10. [Dashboards tích hợp sẵn](#10-dashboards-tích-hợp-sẵn)
11. [Xử lý sự cố](#11-xử-lý-sự-cố)

---

## 1. Tổng quan

**GreenNode vMonitor Datasource** là plugin Grafana cho phép truy vấn dữ liệu **Metric** và **Log** từ nền tảng GreenNode vMonitor Platform trực tiếp trong Grafana.

### Tính năng chính

| Tính năng | Mô tả |
|-----------|-------|
| **Query Metric** | Truy vấn metric với các statistic (avg, min, max, sum, count, rate), filter theo dimension, group by, và các hàm Rollup / Rate / Timeshift / Rank / Reduce |
| **Expression Mode** | Tính toán biểu thức toán học giữa các query (ví dụ: `A / B * 100`) |
| **Query Logs** | Tìm kiếm log từ log project, hiển thị dạng log lines, bảng, hoặc time series |
| **Alerting** | Cảnh báo trên metric và trên số lượng log khớp điều kiện |
| **Template Variables** | Hỗ trợ biến động cho dashboard |
| **Dashboards tích hợp** | 8 dashboard mẫu cho Host, vServer, vLB, vDB, vStorage, vBackup |

---

## 2. Yêu cầu

- **Grafana** phiên bản >= 12.3.0
- **IAM Access Key** của GreenNode (Client ID + Client Secret) có quyền đọc vMonitor

---

## 3. Cài đặt plugin

### Bước 1: Giải nén file plugin

Giải nén file `greennode-vmonitor-datasource-1.0.0.zip` vào thư mục plugin của Grafana:

```bash
# Linux/macOS
unzip greennode-vmonitor-datasource-1.0.0.zip -d /var/lib/grafana/plugins/

# Hoặc đường dẫn tuỳ theo cấu hình Grafana của bạn
unzip greennode-vmonitor-datasource-1.0.0.zip -d /path/to/grafana/plugins/
```

Sau khi giải nén, cấu trúc thư mục sẽ là:

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

### Bước 2: Cho phép plugin chưa ký chạy

Vì plugin chưa được ký với Grafana, cần thêm cấu hình sau vào file `grafana.ini`:

```ini
[plugins]
allow_loading_unsigned_plugins = greennode-vmonitor-datasource
```

Vị trí file `grafana.ini` tuỳ theo cách cài đặt:
- Linux: `/etc/grafana/grafana.ini`
- macOS (Homebrew): `/opt/homebrew/etc/grafana/grafana.ini`
- Docker: mount file cấu hình hoặc dùng biến môi trường `GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=greennode-vmonitor-datasource`

### Bước 3: Khởi động lại Grafana

```bash
# systemd
sudo systemctl restart grafana-server

# Docker
docker restart <container_name>
```

### Bước 4: Kiểm tra plugin đã được nhận diện

Vào **Grafana → Administration → Plugins**, tìm kiếm "vMonitor" — plugin phải hiển thị trạng thái **Installed**.

---

## 4. Cấu hình datasource

### Thêm datasource mới

1. Vào **Grafana → Connections → Data sources → Add new data source**
2. Tìm kiếm "GreenNode vMonitor" và chọn

### Các trường cấu hình

| Trường | Mô tả | Giá trị mặc định |
|--------|-------|-----------------|
| **API URL** | Địa chỉ API của vMonitor Platform | `https://vmonitorapis.vngcloud.vn/vmonitor-api/api` |
| **Client ID** | IAM Access Key ID | _(bắt buộc)_ |
| **Client Secret** | IAM Access Key Secret (được mã hoá, chỉ backend đọc) | _(bắt buộc)_ |
| **Token URL** | Endpoint lấy token IAM | `https://iamapis.vngcloud.vn/accounts-api/v2/auth/token` |

> **Lưu ý bảo mật:** Client Secret được lưu mã hoá và chỉ được đọc bởi backend Go. Trình duyệt không bao giờ nhận được secret này.

### Lấy IAM Access Key

1. Đăng nhập vào **GreenNode IAM Portal**
2. Vào **Access Keys → Create Access Key**
3. Gán quyền **vMonitor read** cho access key
4. Sao chép **Access Key ID** (Client ID) và **Access Key Secret** (Client Secret)

### Kiểm tra kết nối

Nhấn **Save & Test** — nếu cấu hình đúng sẽ hiển thị thông báo thành công.

---

## 5. Query Metric

### 5.1. Cấu trúc cơ bản

Khi tạo panel mới hoặc chỉnh sửa query, chọn datasource **GreenNode vMonitor** và chế độ **Metric**.

Các trường cơ bản:

| Trường | Mô tả | Ví dụ |
|--------|-------|-------|
| **Metric** | Tên metric cần truy vấn | `net.if.in.bytes_sec` |
| **Statistic** | Hàm tổng hợp | `avg` |
| **Filter by** | Lọc theo dimension (có thể thêm nhiều bộ lọc) | `host = web-server-01` |
| **Group by** | Nhóm kết quả theo dimension | `host` |
| **Alias** | Tên hiển thị của series | `{{host}}` |

### 5.2. Các loại Statistic

| Statistic | Mô tả |
|-----------|-------|
| `avg` | Trung bình cộng |
| `min` | Giá trị nhỏ nhất |
| `max` | Giá trị lớn nhất |
| `sum` | Tổng |
| `count` | Số điểm dữ liệu |
| `rate_1s` | Tốc độ thay đổi mỗi giây |
| `rate_1m` | Tốc độ thay đổi mỗi phút |
| `rate_5m` | Tốc độ thay đổi mỗi 5 phút |

> **Lưu ý:** Các statistic `rate_*` **không** hỗ trợ các hàm Rollup, Rate function và Timeshift.

### 5.3. Bộ lọc Dimension (Filter by)

Cho phép lọc dữ liệu theo các thuộc tính của metric:

- Nhấn **+ Filter** để thêm bộ lọc mới
- Chọn **key** (tên dimension) từ danh sách gợi ý
- Nhập **value** để lọc (hỗ trợ Grafana variables, ví dụ: `$host`)
- Có thể thêm nhiều bộ lọc, tất cả áp dụng theo logic **AND**

**Ví dụ:** Lọc metric của một server cụ thể:
```
host = $selected_host
```

### 5.4. Group by

Nhóm kết quả thành nhiều series theo giá trị của dimension:

- Chọn một hoặc nhiều dimension key từ danh sách gợi ý
- Mỗi tổ hợp giá trị sẽ tạo ra một series riêng biệt
- Kết hợp với **Alias** để đặt tên series tự động: `{{host}} - {{interface}}`

### 5.5. Các hàm bổ sung (Functions)

Nhấn **Add Function** để thêm hàm xử lý dữ liệu:

#### Rollup — Tổng hợp theo khoảng thời gian

Gộp nhiều điểm dữ liệu thành một điểm theo khoảng thời gian:

| Trường | Giá trị |
|--------|---------|
| Loại | avg, min, max, sum, count |
| Giá trị | Số nguyên dương |
| Đơn vị | minutes, hours, days, weeks |

**Ví dụ:** Rollup avg 5 minutes — hiển thị trung bình mỗi 5 phút thay vì từng điểm.

> Không áp dụng cho statistic `rate_*`.

#### Rate — Tốc độ thay đổi

Chuyển đổi metric từ dạng tích luỹ sang tốc độ thay đổi:

| Giá trị | Mô tả |
|---------|-------|
| `per_second` | Tốc độ mỗi giây |
| `per_minute` | Tốc độ mỗi phút |
| `per_hour` | Tốc độ mỗi giờ |

> Không áp dụng cho statistic `rate_*`.

#### Timeshift — Dịch chuyển thời gian

So sánh metric tại thời điểm hiện tại với thời điểm trong quá khứ:

| Trường | Giá trị |
|--------|---------|
| Giá trị | Số nguyên dương |
| Đơn vị | minutes, hours, days, weeks |

**Ví dụ:** Timeshift 1 week — so sánh lưu lượng tuần này với tuần trước.

> Không áp dụng cho statistic `rate_*`.

#### Rank — Lọc Top N / Bottom N

Chỉ giữ lại N series có giá trị cao nhất hoặc thấp nhất tại mỗi điểm thời gian:

| Trường | Giá trị |
|--------|---------|
| Loại | `topk` (top N) hoặc `bottomk` (bottom N) |
| N | 5, 10, 20, 25, 50 hoặc tuỳ chỉnh |

**Ví dụ:** Top 10 hosts có CPU cao nhất.

> Series bị loại sẽ có giá trị trống (gaps). Khi dùng trong alert rule, nên thêm Resample trước Reduce.

#### Reduce — Thu gọn về một giá trị

Rút gọn toàn bộ series thành một giá trị duy nhất (dùng cho panel Stat, Top List):

| Giá trị | Mô tả |
|---------|-------|
| `last` | Giá trị cuối cùng |
| `avg` | Trung bình |
| `min` | Nhỏ nhất |
| `max` | Lớn nhất |
| `sum` | Tổng |

### 5.6. Ví dụ thực tế

**Ví dụ 1: CPU usage của tất cả vServer, top 5**
- Metric: `cpu.usage_user`
- Statistic: `avg`
- Filter: `product = vserver`
- Group by: `resource_id`
- Function: Rank → topk, 5
- Alias: `{{resource_id}}`

**Ví dụ 2: Network throughput so sánh tuần trước**
- Query A: `net.if.in.bytes_sec`, statistic `avg`
- Query B: `net.if.in.bytes_sec`, statistic `avg`, Function: Timeshift 1 week

---

## 6. Query Logs

### 6.1. Chọn Log Project

Sau khi chọn chế độ **Logs** trong query editor:

1. Chọn **Log project** từ danh sách — danh sách lấy từ API vMonitor, chỉ hiển thị project đang hoạt động (billing_status = ACTIVE)
2. Sau khi chọn project, các tuỳ chọn bộ lọc và cột mới xuất hiện

### 6.2. Định dạng hiển thị (Format)

| Format | Mô tả | Dùng khi nào |
|--------|-------|--------------|
| **logs** | Hiển thị dạng log lines, có histogram, có thể mở rộng xem chi tiết JSON | Explore, Logs panel |
| **table** | Hiển thị dạng bảng, mỗi dòng là một log entry | Table panel |
| **timeseries** | Hiển thị số lượng log theo thời gian | Graph panel, Alerting |

> Trong chế độ **Alert**, format tự động chuyển về `timeseries`.

### 6.3. Giới hạn kết quả (Limit)

Chọn số log tối đa trả về: **50, 100, 200, 500**. Mặc định là 100.

> Không áp dụng cho format `timeseries`.

### 6.4. Bộ lọc Log (Log Filters)

Nhấn **+ Filter** để thêm điều kiện lọc. Mỗi bộ lọc gồm:

- **Field**: Tên trường log (có gợi ý tự động)
- **Operator**: Toán tử so sánh
- **Value**: Giá trị lọc

#### Các toán tử hỗ trợ

| Toán tử | Nhập giá trị | Ý nghĩa |
|---------|-------------|---------|
| `is` | 1 giá trị | Bằng đúng |
| `is not` | 1 giá trị | Khác |
| `is one of` | Nhiều giá trị | Bằng một trong các giá trị |
| `is not one of` | Nhiều giá trị | Khác tất cả các giá trị |
| `is between` | 2 giá trị (từ, đến) | Nằm trong khoảng [từ, đến) |
| `is not between` | 2 giá trị (từ, đến) | Nằm ngoài khoảng |
| `exists` | _(không nhập)_ | Trường tồn tại |
| `does not exist` | _(không nhập)_ | Trường không tồn tại |

**Ví dụ lọc log:**
```
level   is          ERROR
host    is one of   web-01, web-02
latency is between  100, 500
```

> Hỗ trợ Grafana variables trong giá trị, ví dụ: `$selected_host`.

### 6.5. Cột hiển thị (Log Columns)

Chọn các trường log muốn hiển thị làm cột (áp dụng cho format `logs` và `table`):

- Nhấn **+ Columns** để thêm nhóm cột mới
- Chọn một hoặc nhiều trường từ danh sách gợi ý
- Dùng nút **mũi tên** để sắp xếp lại thứ tự cột
- Dùng nút **thùng rác** để xoá cột

### 6.6. Nhóm kết quả (Group by) — chỉ dùng với timeseries

Khi format là `timeseries`, có thể nhóm số lượng log theo giá trị của trường:

- Chọn một hoặc nhiều trường **aggregatable** (kiểu `keyword`)
- Mỗi tổ hợp giá trị trở thành một series riêng biệt
- Tối đa 20 giá trị mỗi trường

> **Lưu ý khi dùng với Alert:** Mỗi giá trị group by sẽ tạo ra một alert instance riêng. Các tổ hợp không có log sẽ **không** kích hoạt cảnh báo.

### 6.7. Ví dụ thực tế

**Ví dụ 1: Xem log lỗi của ứng dụng**
- Format: logs
- Limit: 100
- Filter: `level is ERROR`
- Columns: timestamp, message, host, service

**Ví dụ 2: Thống kê số lỗi theo service theo giờ**
- Format: timeseries
- Filter: `level is ERROR`
- Group by: `service`

**Ví dụ 3: Cảnh báo khi có quá nhiều lỗi 5xx**
- Format: timeseries (tự động trong alert mode)
- Filter: `http_status is between 500, 600`

---

## 7. Biểu thức toán học (Expression Mode)

Expression Mode cho phép tính toán giữa các metric query trong cùng một panel.

### Cách sử dụng

1. Tạo ít nhất 2 query metric (ví dụ: Query **A** và Query **B**)
2. Thêm query mới, chọn **Query Type: Expression**
3. Nhập biểu thức sử dụng chữ cái của các query tham chiếu

### Cú pháp biểu thức

| Biểu thức | Ý nghĩa |
|-----------|---------|
| `A + B` | Cộng |
| `A - B` | Trừ |
| `A * B` | Nhân |
| `A / B` | Chia |
| `A / B * 100` | Phần trăm |
| `(A + B) / 2` | Trung bình 2 metric |

### Ví dụ thực tế

**Tính tỷ lệ CPU system so với total:**
- Query A: `cpu.usage_system`, statistic `avg`
- Query B: `cpu.usage_total`, statistic `avg`
- Expression C: `A / B * 100`

> **Giới hạn:** Expression Mode không hỗ trợ các statistic `rate_*`.

---

## 8. Template Variables

Plugin hỗ trợ 5 hàm để tạo biến động trong dashboard.

### Cách tạo Template Variable

1. Vào **Dashboard Settings → Variables → Add variable**
2. Chọn **Type: Query**, **Data source: GreenNode vMonitor**
3. Nhập hàm truy vấn vào trường **Query**

### Các hàm hỗ trợ

#### `metrics()`

Lấy danh sách tất cả metric có sẵn.

```
metrics()
```

**Dùng để tạo biến:** Cho phép người dùng chọn metric từ dropdown.

---

#### `dimensions(metricName)`

Lấy danh sách tất cả dimension key của một metric.

```
dimensions(cpu.usage_user)
```

---

#### `dimensionValues(metricName, dimensionKey)`

Lấy tất cả giá trị của một dimension key cho metric cụ thể.

```
dimensionValues(cpu.usage_user, host)
```

**Ví dụ điển hình:** Tạo biến `$host` để lọc dashboard theo server:
- Variable name: `host`
- Query: `dimensionValues(cpu.usage_user, host)`
- Bật **Multi-value** và **Include All option** nếu muốn chọn nhiều host

---

#### `infrastructure(product[, region])`

Lấy danh sách tài nguyên theo loại sản phẩm, có thể lọc theo region.

**Các product được hỗ trợ:**

| Product | Tài nguyên |
|---------|-----------|
| `host` | Host monitoring agent |
| `vserver` | Virtual Server |
| `vlb` | Load Balancer |
| `vstorage` | vStorage Project |
| `vstorage-bucket` | vStorage Bucket (hỗ trợ region) |
| `vdb` | vDB Relational |
| `vdb-kafka` | vDB Kafka |
| `vdb-cluster` | vDB Cluster |
| `vbackup` | vBackup |

```
infrastructure(vserver)
infrastructure(vstorage-bucket, HCM03)
```

**Ví dụ:** Tạo biến `$server` để chọn vServer:
- Query: `infrastructure(vserver)`

---

#### `regions(product)`

Lấy danh sách region có sẵn của sản phẩm (chỉ áp dụng cho sản phẩm có region như `vstorage-bucket`).

```
regions(vstorage-bucket)
```

---

### Ví dụ: Dashboard động theo vServer

**Tạo các variables:**

| Variable | Query |
|----------|-------|
| `server` | `infrastructure(vserver)` |
| `metric` | `metrics()` |

**Sử dụng trong query:**
- Filter: `resource_id = $server`
- Metric: `$metric`

---

## 9. Cảnh báo (Alerting)

Plugin hỗ trợ tạo alert rule trực tiếp từ query metric hoặc log.

### 9.1. Alert trên Metric

1. Tạo query metric như bình thường (mode: Metric)
2. Chuyển sang tab **Alert** trong panel editor
3. Cấu hình **Reduce** (ví dụ: last, avg) và **Threshold**

**Lưu ý quan trọng khi dùng Rank:**
- Nếu query có hàm Rank, series bị loại sẽ có giá trị trống (gaps)
- Thêm **Resample** trước **Reduce** trong alert rule để xử lý gaps

### 9.2. Alert trên Log Count

Đây là tương đương với **Log Alarm (Frequency)** của vMonitor:

1. Tạo query log với format: `timeseries`
2. Thêm bộ lọc phù hợp (ví dụ: `level is ERROR`)
3. Cấu hình alert rule với Reduce + Threshold

**Cảnh báo khi log ngừng (Flatline):**
- Khoảng thời gian không có log báo cáo là `0` thay vì dữ liệu trống
- Có thể đặt threshold `count = 0` để cảnh báo khi không có log trong khoảng thời gian

### 9.3. Alert với Group by (Log)

Khi log query có **Group by**:
- Mỗi giá trị group tạo ra một **alert instance** riêng biệt
- Grafana hiển thị từng instance riêng với label tương ứng
- Các tổ hợp group không có log sẽ không tạo instance — cần tính đến khi thiết kế rule

---

## 10. Dashboards tích hợp sẵn

Plugin đi kèm 8 dashboard mẫu. Để sử dụng:

**Cách 1: Import từ plugin**
1. Vào **Dashboards → New → Import**
2. Chọn **GreenNode vMonitor** datasource
3. Chọn dashboard mẫu muốn sử dụng

**Cách 2: Tự động (nếu dùng provisioning)**
Dashboard được tự động tải khi plugin được cài đặt.

### Danh sách dashboard

| Dashboard | Mô tả |
|-----------|-------|
| **vMonitor / Host** | Metric từ host agent: CPU, RAM, Disk, Network cho máy chủ bất kỳ |
| **vMonitor / vServer** | Metric platform-level của GreenNode Virtual Server |
| **vMonitor / vLB (Load Balancer)** | Metric lưu lượng và health của Load Balancer |
| **vMonitor / vDB (Relational)** | Metric hiệu suất database quan hệ (MySQL, PostgreSQL...) |
| **vMonitor / vDB (Kafka)** | Metric cluster và topic của Kafka |
| **vMonitor / vStorage** | Metric hiệu suất và dung lượng vStorage project |
| **vMonitor / vStorage (Bucket)** | Metric theo từng bucket |
| **vMonitor / vBackup** | Trạng thái và hiệu suất backup job |

### Cách hoạt động của resource picker trong dashboard

Các dashboard dùng Template Variable `infrastructure(product)` để liệt kê tài nguyên — danh sách này lấy từ API vMonitor nên bao gồm **tất cả** tài nguyên, kể cả những tài nguyên chưa bật detailed monitoring.

- **Host dashboards**: Metric đến từ agent trong VM, không có dimension `product`
- **Các dashboard khác**: Query dùng dimension filter `product:<name>` để lấy metric platform-level

---

## 11. Xử lý sự cố

### Plugin không hiện trong danh sách

- Kiểm tra thư mục plugin đúng cấu trúc: `greennode-vmonitor-datasource/plugin.json` phải tồn tại
- Kiểm tra `grafana.ini` đã có `allow_loading_unsigned_plugins = greennode-vmonitor-datasource`
- Xem log Grafana: `journalctl -u grafana-server` hoặc `/var/log/grafana/grafana.log`

### Save & Test báo lỗi xác thực

| Lỗi | Nguyên nhân | Giải pháp |
|-----|-------------|-----------|
| `401 Unauthorized` | Client ID hoặc Secret sai | Kiểm tra lại IAM Access Key |
| `connection refused` | API URL sai hoặc không thể kết nối | Kiểm tra API URL và network |
| `token exchange failed` | Token URL sai | Kiểm tra Token URL hoặc để trống để dùng mặc định |

### Query trả về không có dữ liệu

- Kiểm tra time range — metric có thể chưa có dữ liệu trong khoảng thời gian chọn
- Kiểm tra bộ lọc dimension — value có thể không khớp
- Thử xoá bớt bộ lọc để kiểm tra metric có dữ liệu không
- Kiểm tra IAM key có quyền đọc metric/log của resource đó

### Log query không hiện project

- Kiểm tra IAM key có quyền đọc log project
- Kiểm tra log project đang ở trạng thái ACTIVE (billing_status = ACTIVE)

### Alert không kích hoạt

- Với metric có Rank: thêm **Resample** trước **Reduce** trong alert rule
- Với log group by: nhớ rằng nhóm không có log sẽ không tạo alert instance
- Kiểm tra Reduce function có phù hợp với loại dữ liệu không
