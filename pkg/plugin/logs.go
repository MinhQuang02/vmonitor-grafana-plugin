package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/vng/green-node-v-monitor/pkg/vmonitor"
)

const (
	logFieldTimestamp = "@timestamp"
	logDefaultLimit   = 100
	logMaxLimit       = 500
	logVolumeAggName  = "chart"

	logFormatLogs       = "logs"
	logFormatTable      = "table"
	logFormatTimeseries = "timeseries"

	logHistogramBuckets = 120
	logGroupTermsSize   = 20
	logGroupAggPrefix   = "group_"
)

type logFilter struct {
	Field    string   `json:"field"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

func (qm queryModel) isLogs() bool {
	return qm.QueryMode == "logs"
}

func (qm queryModel) logDisplayFormat(fromAlert bool) string {
	if fromAlert {
		return logFormatTimeseries
	}
	if qm.LogFormat == "" {
		return logFormatLogs
	}
	return qm.LogFormat
}

func (qm queryModel) logSize() int {
	if qm.LogLimit <= 0 {
		return logDefaultLimit
	}
	if qm.LogLimit > logMaxLimit {
		return logMaxLimit
	}
	return qm.LogLimit
}

func (d *Datasource) queryLogs(ctx context.Context, query backend.DataQuery, qm queryModel, fromAlert bool) backend.DataResponse {
	var response backend.DataResponse
	if qm.LogProjectID == "" {
		return response
	}

	gte := query.TimeRange.From.UTC().Format(time.RFC3339Nano)
	lte := query.TimeRange.To.UTC().Format(time.RFC3339Nano)
	format := qm.logDisplayFormat(fromAlert)
	step := logStep(query.TimeRange.From, query.TimeRange.To)

	req := vmonitor.LogSearchRequest{
		From:         0,
		Size:         qm.logSize(),
		Query:        buildLogQuery(qm, gte, lte),
		Sorts:        []map[string]any{vmonitor.LogFieldSort(logFieldTimestamp, "desc")},
		Aggregations: []map[string]any{},
	}
	if format == logFormatTimeseries {
		req.Size = 0
		req.Aggregations = []map[string]any{vmonitor.LogCountAggregation(
			logVolumeAggName, logFieldTimestamp, logIntervalString(step), logHistogramBuckets, logGroupBys(qm),
		)}
	}

	result, err := d.client.SearchLogs(ctx, qm.LogProjectID, req)
	if err != nil {
		return errorResponse(err)
	}

	switch format {
	case logFormatTimeseries:
		response.Frames = logCountFrames(qm, result, query.TimeRange, step)
	case logFormatTable:
		response.Frames = append(response.Frames, logTableFrame(qm, result))
	default:
		response.Frames = append(response.Frames, logLinesFrame(qm, result))
	}
	return response
}

func logGroupBys(qm queryModel) []vmonitor.LogGroupBy {
	groups := make([]vmonitor.LogGroupBy, 0, len(qm.LogGroupBy))
	for _, field := range qm.LogGroupBy {
		if field = strings.TrimSpace(field); field == "" {
			continue
		}
		groups = append(groups, vmonitor.LogGroupBy{
			Name:  fmt.Sprintf("%s%d", logGroupAggPrefix, len(groups)),
			Field: field,
			Size:  logGroupTermsSize,
		})
	}
	return groups
}

func buildLogQuery(qm queryModel, gte, lte string) map[string]any {
	var must []map[string]any
	for _, f := range qm.LogFilters {
		if clause := logFilterToQuery(f); clause != nil {
			must = append(must, clause)
		}
	}
	return vmonitor.LogBoolQuery(
		[]map[string]any{vmonitor.LogTimeRange(logFieldTimestamp, gte, lte)},
		must, nil, nil,
	)
}

func logFilterToQuery(f logFilter) map[string]any {
	field := strings.TrimSpace(f.Field)
	if field == "" {
		return nil
	}
	values := make([]string, 0, len(f.Values))
	for _, v := range f.Values {
		if v = strings.TrimSpace(v); v != "" {
			values = append(values, v)
		}
	}

	switch f.Operator {
	case "is":
		if len(values) == 0 {
			return nil
		}
		return vmonitor.LogBoolQuery(nil, []map[string]any{vmonitor.LogMatch(field, values[0])}, nil, nil)
	case "is not":
		if len(values) == 0 {
			return nil
		}
		return vmonitor.LogBoolQuery(nil, nil, []map[string]any{vmonitor.LogMatch(field, values[0])}, nil)
	case "is one of":
		if len(values) == 0 {
			return nil
		}
		should := make([]map[string]any, 0, len(values))
		for _, v := range values {
			should = append(should, vmonitor.LogBoolQuery(nil, []map[string]any{vmonitor.LogMatch(field, v)}, nil, nil))
		}
		return vmonitor.LogBoolQuery(nil, nil, nil, should)
	case "is not one of":
		if len(values) == 0 {
			return nil
		}
		must := make([]map[string]any, 0, len(values))
		for _, v := range values {
			must = append(must, vmonitor.LogBoolQuery(nil, nil, []map[string]any{vmonitor.LogMatch(field, v)}, nil))
		}
		return vmonitor.LogBoolQuery(nil, must, nil, nil)
	case "exists":
		return vmonitor.LogBoolQuery(nil, []map[string]any{vmonitor.LogExists(field)}, nil, nil)
	case "does not exist":
		return vmonitor.LogBoolQuery(nil, nil, []map[string]any{vmonitor.LogExists(field)}, nil)
	case "is between":
		if len(values) < 2 {
			return nil
		}
		return vmonitor.LogBoolQuery(nil, []map[string]any{
			vmonitor.LogRange(field, map[string]any{"gte": values[0]}),
			vmonitor.LogRange(field, map[string]any{"lt": values[1]}),
		}, nil, nil)
	case "is not between":
		if len(values) < 2 {
			return nil
		}
		return vmonitor.LogBoolQuery(nil, nil, nil, []map[string]any{
			vmonitor.LogBoolQuery(nil, []map[string]any{vmonitor.LogRange(field, map[string]any{"lt": values[0]})}, nil, nil),
			vmonitor.LogBoolQuery(nil, []map[string]any{vmonitor.LogRange(field, map[string]any{"gte": values[1]})}, nil, nil),
		})
	}
	return nil
}

func logStep(from, to time.Time) time.Duration {
	span := to.Sub(from)
	if span <= 0 {
		return time.Minute
	}
	step := (span / logHistogramBuckets).Round(time.Second)
	if step < time.Second {
		return time.Second
	}
	return step
}

func logIntervalString(step time.Duration) string {
	seconds := int64(step / time.Second)
	switch {
	case seconds%86400 == 0:
		return fmt.Sprintf("%dd", seconds/86400)
	case seconds%3600 == 0:
		return fmt.Sprintf("%dh", seconds/3600)
	case seconds%60 == 0:
		return fmt.Sprintf("%dm", seconds/60)
	default:
		return fmt.Sprintf("%ds", seconds)
	}
}

func logTimeGrid(timeRange backend.TimeRange, step time.Duration) []time.Time {
	start := floorToStep(timeRange.From.UTC(), step)
	end := timeRange.To.UTC()
	grid := make([]time.Time, 0, logHistogramBuckets+1)
	for t := start; !t.After(end) && len(grid) <= logHistogramBuckets*2; t = t.Add(step) {
		grid = append(grid, t)
	}
	return grid
}

func floorToStep(t time.Time, step time.Duration) time.Time {
	stepMillis := step.Milliseconds()
	if stepMillis <= 0 {
		return t
	}
	millis := t.UnixMilli()
	floored := millis - millis%stepMillis
	if millis < 0 && millis%stepMillis != 0 {
		floored -= stepMillis
	}
	return time.UnixMilli(floored).UTC()
}

func gridSlot(grid []time.Time, step time.Duration, millis int64) (int, bool) {
	if len(grid) == 0 || step <= 0 {
		return 0, false
	}
	offset := millis - grid[0].UnixMilli()
	if offset < 0 {
		return 0, false
	}
	slot := int(offset / step.Milliseconds())
	if slot >= len(grid) {
		return 0, false
	}
	return slot, true
}

type logCountSeries struct {
	labels data.Labels
	counts []float64
}

func logCountFrames(qm queryModel, result *vmonitor.LogSearchResult, timeRange backend.TimeRange, step time.Duration) data.Frames {
	grid := logTimeGrid(timeRange, step)
	groups := logGroupBys(qm)
	series := map[string]*logCountSeries{}
	var order []string

	for _, bucket := range result.Aggregations[logVolumeAggName].Buckets {
		millis, ok := bucket.KeyMillis()
		if !ok {
			continue
		}
		slot, ok := gridSlot(grid, step, millis)
		if !ok {
			continue
		}
		collectLogCounts(bucket, groups, 0, data.Labels{}, slot, len(grid), series, &order)
	}

	if len(groups) == 0 && len(order) == 0 {
		logCountSeriesFor(data.Labels{}, len(grid), series, &order)
	}

	frames := make(data.Frames, 0, len(order))
	for _, key := range order {
		frames = append(frames, logCountFrame(qm, series[key], grid))
	}
	return frames
}

func collectLogCounts(
	bucket vmonitor.LogBucket,
	groups []vmonitor.LogGroupBy,
	depth int,
	labels data.Labels,
	slot, size int,
	series map[string]*logCountSeries,
	order *[]string,
) {
	if depth == len(groups) {
		s := logCountSeriesFor(labels, size, series, order)
		s.counts[slot] = float64(bucket.DocCount)
		return
	}
	group := groups[depth]
	sub, ok := bucket.Aggregations[group.Name]
	if !ok {
		return
	}
	for _, child := range sub.Buckets {
		next := labels.Copy()
		next[group.Field] = child.KeyLabel()
		collectLogCounts(child, groups, depth+1, next, slot, size, series, order)
	}
}

func logCountSeriesFor(labels data.Labels, size int, series map[string]*logCountSeries, order *[]string) *logCountSeries {
	key := labels.String()
	if s, ok := series[key]; ok {
		return s
	}
	s := &logCountSeries{labels: labels, counts: make([]float64, size)}
	series[key] = s
	*order = append(*order, key)
	return s
}

func logCountFrame(qm queryModel, s *logCountSeries, grid []time.Time) *data.Frame {
	times := make([]time.Time, len(grid))
	copy(times, grid)

	name := logCountSeriesName(qm, s.labels)
	countField := data.NewField("count", s.labels, s.counts)
	countField.Config = &data.FieldConfig{DisplayNameFromDS: name}

	frame := data.NewFrame("logs", data.NewField("time", nil, times), countField)
	frame.Meta = &data.FrameMeta{PreferredVisualization: data.VisTypeGraph}
	return frame
}

func logCountSeriesName(qm queryModel, labels data.Labels) string {
	if len(labels) == 0 {
		if qm.Alias != "" {
			return qm.Alias
		}
		return "Log count"
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s(%s)", k, labels[k]))
	}
	grouped := strings.Join(parts, ",")
	if qm.Alias != "" {
		return qm.Alias + " " + grouped
	}
	return grouped
}

func logLinesFrame(qm queryModel, result *vmonitor.LogSearchResult) *data.Frame {
	hits := result.Hits.Hits
	times := make([]time.Time, 0, len(hits))
	bodies := make([]string, 0, len(hits))
	severities := make([]string, 0, len(hits))
	ids := make([]string, 0, len(hits))
	columns := logColumns(qm)
	columnValues := make(map[string][]string, len(columns))

	for _, hit := range hits {
		source := decodeLogSource(hit.Source)
		times = append(times, logHitTime(hit, source))
		bodies = append(bodies, logLineJSON(hit.Source, source))
		severities = append(severities, logSeverity(source))
		ids = append(ids, hit.ID)
		for _, col := range columns {
			value, _ := lookupLogValue(source, col)
			columnValues[col] = append(columnValues[col], value)
		}
	}

	timestampField := data.NewField("timestamp", nil, times)
	timestampField.Config = logLinesFieldConfig()

	bodyField := data.NewField("body", nil, bodies)
	bodyField.Config = logLinesFieldConfig()

	severityField := data.NewField("severity", nil, severities)
	severityField.Config = logLinesFieldConfig()

	idField := data.NewField("id", nil, ids)
	idField.Config = logLinesFieldConfig()

	fields := []*data.Field{
		timestampField,
		bodyField,
		severityField,
		idField,
	}
	reserved := map[string]bool{logFieldTimestamp: true, "timestamp": true, "body": true, "severity": true, "id": true}
	for _, col := range columns {
		if reserved[col] {
			continue
		}
		field := data.NewField(col, nil, columnValues[col])
		field.Config = logLinesFieldConfig()
		fields = append(fields, field)
	}

	frame := data.NewFrame("logs", fields...)
	frame.Meta = &data.FrameMeta{
		Type:                   data.FrameTypeLogLines,
		PreferredVisualization: data.VisTypeLogs,
		Custom: map[string]any{
			"displayedFields": displayedLogFields(columns),
		},
	}
	return frame
}

func displayedLogFields(columns []string) []string {
	displayed := []string{"timestamp", "body", "severity"}
	seen := map[string]bool{"timestamp": true, "body": true, "severity": true}
	reserved := map[string]bool{logFieldTimestamp: true, "timestamp": true, "body": true, "severity": true, "id": true}
	for _, col := range columns {
		col = strings.TrimSpace(col)
		if col == "" || reserved[col] || seen[col] {
			continue
		}
		seen[col] = true
		displayed = append(displayed, col)
	}
	return displayed
}

func logLinesFieldConfig() *data.FieldConfig {
	return &data.FieldConfig{Custom: map[string]any{"inspect": false}}
}

func logLineJSON(raw json.RawMessage, source map[string]any) string {
	if len(raw) > 0 {
		var obj any
		if err := json.Unmarshal(raw, &obj); err == nil {
			if pretty, err := json.MarshalIndent(obj, "", "  "); err == nil {
				return string(pretty)
			}
		}
	}
	pretty, err := json.MarshalIndent(source, "", "  ")
	if err != nil {
		return ""
	}
	return string(pretty)
}

func logTableFrame(qm queryModel, result *vmonitor.LogSearchResult) *data.Frame {
	hits := result.Hits.Hits
	columns := logColumns(qm)
	if len(columns) == 0 {
		columns = []string{logFieldTimestamp}
	}

	times := make([]time.Time, 0, len(hits))
	inspectValues := make([]string, 0, len(hits))
	columnValues := make(map[string][]string, len(columns))
	for _, hit := range hits {
		source := decodeLogSource(hit.Source)
		times = append(times, logHitTime(hit, source))
		inspectValues = append(inspectValues, logLineJSON(hit.Source, source))
		for _, col := range columns {
			value, _ := lookupLogValue(source, col)
			columnValues[col] = append(columnValues[col], value)
		}
	}

	inspectField := data.NewField("inspect", nil, inspectValues)
	inspectField.Config = &data.FieldConfig{
		DisplayNameFromDS: " ",
		Custom: map[string]any{
			"cellOptions": map[string]any{"type": "custom"},
			"hideHeader":  true,
			"inspect":     false,
			"width":       52,
			"minWidth":    52,
		},
	}

	fields := []*data.Field{inspectField, data.NewField("Time", nil, times)}
	for _, col := range columns {
		if col == logFieldTimestamp {
			continue
		}
		fields = append(fields, data.NewField(col, nil, columnValues[col]))
	}

	frame := data.NewFrame("logs", fields...)
	frame.Meta = &data.FrameMeta{PreferredVisualization: data.VisTypeTable}
	return frame
}

func logColumns(qm queryModel) []string {
	columns := make([]string, 0, len(qm.LogColumns))
	for _, col := range qm.LogColumns {
		if col = strings.TrimSpace(col); col != "" {
			columns = append(columns, col)
		}
	}
	return columns
}

func decodeLogSource(raw json.RawMessage) map[string]any {
	var source map[string]any
	if err := json.Unmarshal(raw, &source); err != nil {
		return map[string]any{}
	}
	return source
}

func logHitTime(hit vmonitor.LogHit, source map[string]any) time.Time {
	if len(hit.Sort) > 0 {
		if ms, err := hit.Sort[0].Int64(); err == nil {
			return time.UnixMilli(ms)
		}
	}
	if value, ok := source[logFieldTimestamp]; ok {
		if s, ok := value.(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func logSeverity(source map[string]any) string {
	for _, field := range []string{"severity", "level", "log_level"} {
		if value, ok := lookupLogValue(source, field); ok && value != "" {
			return value
		}
	}
	return ""
}

func lookupLogValue(source map[string]any, path string) (string, bool) {
	if value, ok := lookupLogPath(source, path); ok {
		return value, true
	}
	if base, found := strings.CutSuffix(path, ".keyword"); found {
		return lookupLogPath(source, base)
	}
	return "", false
}

func lookupLogPath(source map[string]any, path string) (string, bool) {
	if value, ok := source[path]; ok {
		return logValueToString(value), true
	}
	var current any = source
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = object[part]
		if !ok {
			return "", false
		}
	}
	return logValueToString(current), true
}

func logValueToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
}
