package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/vng/green-node-v-monitor/pkg/models"
	"github.com/vng/green-node-v-monitor/pkg/vmonitor"
)

var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ backend.CallResourceHandler   = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

func NewDatasource(_ context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	config, err := models.LoadPluginSettings(settings)
	if err != nil {
		return nil, err
	}
	tokens := vmonitor.NewTokenSource(config.TokenURL, config.ClientID, config.Secrets.ClientSecret)
	return &Datasource{
		settings: config,
		client:   vmonitor.NewClient(config.URL, config.LogURL, tokens),
	}, nil
}

type Datasource struct {
	settings *models.PluginSettings
	client   *vmonitor.Client
}

func (d *Datasource) Dispose() {}

func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()
	parsed := make(map[string]queryModel, len(req.Queries))
	graphs := make(map[string]vmonitor.Graph)
	rejected := make(map[string]bool)
	for _, q := range req.Queries {
		var qm queryModel
		if err := json.Unmarshal(q.JSON, &qm); err != nil {
			response.Responses[q.RefID] = backend.ErrDataResponse(
				backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
			rejected[q.RefID] = true
			continue
		}
		if err := validateFilters(qm.Filters); err != nil {
			response.Responses[q.RefID] = backend.ErrDataResponse(backend.StatusBadRequest, err.Error())
			rejected[q.RefID] = true
			continue
		}
		parsed[q.RefID] = qm
		if !qm.isExpression() && qm.MetricName != "" {
			graphs[q.RefID] = buildGraph(qm)
		}
	}

	fromAlert := isFromAlert(req.Headers)
	for _, q := range req.Queries {
		qm, ok := parsed[q.RefID]
		if !ok {
			continue
		}
		switch {
		case qm.isLogs():
			response.Responses[q.RefID] = d.queryLogs(ctx, q, qm, fromAlert)
		case qm.isExpression():
			response.Responses[q.RefID] = d.queryExpression(ctx, q, qm, graphs, rejected)
		default:
			response.Responses[q.RefID] = d.queryMetric(ctx, q, qm)
		}
	}
	return response, nil
}

func isFromAlert(headers map[string]string) bool {
	for key, value := range headers {
		if strings.EqualFold(key, backend.FromAlertHeaderName) {
			return strings.EqualFold(value, "true")
		}
	}
	return false
}

type dimensionFilter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type rankSpec struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type rollupSpec struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type timeShiftSpec struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type queryModel struct {
	QueryMode  string            `json:"queryMode"`
	Expression string            `json:"expression"`
	MetricName string            `json:"metricName"`
	Statistic  string            `json:"statistic"`
	Filters    []dimensionFilter `json:"filters"`
	GroupBy    []string          `json:"groupBy"`
	Rank       *rankSpec         `json:"rank"`
	Rollup     *rollupSpec       `json:"rollup"`
	Rate       string            `json:"rate"`
	TimeShift  *timeShiftSpec    `json:"timeShift"`
	Reduce     string            `json:"reduce"`
	Alias      string            `json:"alias"`
	Hide       bool              `json:"hide"`

	LogProjectID string      `json:"logProjectId"`
	LogFilters   []logFilter `json:"logFilters"`
	LogColumns   []string    `json:"logColumns"`
	LogGroupBy   []string    `json:"logGroupBy"`
	LogFormat    string      `json:"logFormat"`
	LogLimit     int         `json:"logLimit"`
}

func (qm queryModel) isExpression() bool {
	return qm.QueryMode == "expression"
}

func (qm queryModel) offset() int {
	if qm.TimeShift == nil {
		return 0
	}
	return unitSeconds(qm.TimeShift.Value, qm.TimeShift.Unit)
}

func (qm queryModel) limitSpec() string {
	if qm.Rank != nil && qm.Rank.Type != "" && qm.Rank.Value != "" {
		return qm.Rank.Type + ":" + qm.Rank.Value
	}
	return ""
}

func isRateStatistic(statistic string) bool {
	switch statistic {
	case "rate_1s", "rate_1m", "rate_5m":
		return true
	}
	return false
}

func (qm queryModel) rateStatisticConflicts() []string {
	var conflicts []string
	if qm.Rollup != nil && qm.Rollup.Type != "" && unitSeconds(qm.Rollup.Value, qm.Rollup.Unit) > 0 {
		conflicts = append(conflicts, "Rollup")
	}
	if rateSeconds(qm.Rate) > 0 {
		conflicts = append(conflicts, "Rate")
	}
	if qm.offset() != 0 {
		conflicts = append(conflicts, "Timeshift")
	}
	return conflicts
}

func validateFilters(filters []dimensionFilter) error {
	for _, f := range filters {
		key, value := strings.TrimSpace(f.Key), strings.TrimSpace(f.Value)
		if key == "" && value == "" {
			continue
		}
		if key == "" {
			return fmt.Errorf("a dimension filter has the value %q but no key", value)
		}
		if value == "" {
			return fmt.Errorf("dimension filter %q has no value. Pick a value, or remove the filter — an empty value would silently widen the query to every resource", key)
		}
	}
	return nil
}

func reducePoints(reducer string, pts []vmonitor.DataPoint) (float64, error) {
	switch reducer {
	case "last":
		return pts[len(pts)-1].Value, nil
	case "min":
		v := pts[0].Value
		for _, p := range pts[1:] {
			if p.Value < v {
				v = p.Value
			}
		}
		return v, nil
	case "max":
		v := pts[0].Value
		for _, p := range pts[1:] {
			if p.Value > v {
				v = p.Value
			}
		}
		return v, nil
	case "sum":
		var v float64
		for _, p := range pts {
			v += p.Value
		}
		return v, nil
	case "avg":
		var v float64
		for _, p := range pts {
			v += p.Value
		}
		return v / float64(len(pts)), nil
	}
	return 0, fmt.Errorf("unknown reduce function %q", reducer)
}

func applyReduce(reducer string, series []vmonitor.Series) ([]vmonitor.Series, error) {
	if reducer == "" {
		return series, nil
	}
	out := make([]vmonitor.Series, 0, len(series))
	for _, s := range series {
		if len(s.Points) == 0 {
			out = append(out, s)
			continue
		}
		value, err := reducePoints(reducer, s.Points)
		if err != nil {
			return nil, err
		}
		s.Points = []vmonitor.DataPoint{{Time: s.Points[len(s.Points)-1].Time, Value: value}}
		out = append(out, s)
	}
	return out, nil
}

func (d *Datasource) queryMetric(ctx context.Context, query backend.DataQuery, qm queryModel) backend.DataResponse {
	var response backend.DataResponse
	if qm.MetricName == "" {
		return response // nothing to query yet
	}

	var series []vmonitor.Series
	var err error

	if isRateStatistic(qm.Statistic) {
		if conflicts := qm.rateStatisticConflicts(); len(conflicts) > 0 {
			return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf(
				"statistic %q is served by vMonitor's basic statistics endpoint, which does not support %s. Remove %s, or choose avg, min, max, sum or count.",
				qm.Statistic, strings.Join(conflicts, " and "), strings.Join(conflicts, " and ")))
		}
		series, err = d.client.GetStatistics(ctx, vmonitor.StatisticsParams{
			Name:        qm.MetricName,
			Statistics:  qm.Statistic,
			Dimensions:  joinFilters(qm.Filters),
			StartMillis: query.TimeRange.From.UnixMilli(),
			EndMillis:   query.TimeRange.To.UnixMilli(),
			GroupBy:     joinGroupBy(qm.GroupBy),
			Limit:       qm.limitSpec(),
		})
	} else {
		series, err = d.client.GetStatisticsV2(ctx, vmonitor.V2Params{
			Graph:       buildGraph(qm),
			StartMillis: query.TimeRange.From.UnixMilli(),
			EndMillis:   query.TimeRange.To.UnixMilli(),
			Period:      periodForInterval(query.Interval),
		})
	}
	if err != nil {
		return errorResponse(err)
	}

	series, err = applyReduce(qm.Reduce, series)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, err.Error())
	}
	for i, s := range series {
		response.Frames = append(response.Frames, seriesToFrame(qm, s, i))
	}
	return response
}

func (d *Datasource) queryExpression(ctx context.Context, query backend.DataQuery, qm queryModel, graphs map[string]vmonitor.Graph, rejected map[string]bool) backend.DataResponse {
	var response backend.DataResponse
	if qm.Expression == "" {
		return response
	}

	tokenGraphs := map[string]vmonitor.Graph{}
	var missing []string
	var rateRefs []string
	var broken []string
	expr := strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			ref := string(r)
			if g, ok := graphs[ref]; ok {
				token := r - 'A' + 'a'
				tokenGraphs[string(token)] = g
				if isRateStatistic(g.Statistics) {
					rateRefs = append(rateRefs, ref)
				}
				return token
			}
			if rejected[ref] {
				broken = append(broken, ref)
			} else {
				missing = append(missing, ref)
			}
		}
		return r
	}, qm.Expression)

	if len(broken) > 0 {
		sort.Strings(broken)
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf(
			"expression references %s, which %s invalid — fix %s first.",
			strings.Join(broken, ", "), pluralIsAre(len(broken)), pluralItThem(len(broken))))
	}
	if len(missing) > 0 {
		return backend.ErrDataResponse(backend.StatusBadRequest,
			fmt.Sprintf("expression references %s, which %s not a metric query in the same request. In a panel, add %s as a metric query. In an alert rule, Grafana sends every query separately, so use a Grafana Math expression instead of this plugin's expression mode.",
				strings.Join(missing, ", "), pluralIsAre(len(missing)), pluralItThem(len(missing))))
	}
	if len(tokenGraphs) == 0 {
		return backend.ErrDataResponse(backend.StatusBadRequest,
			"expression must reference at least one metric query by its letter (A, B, …)")
	}
	if len(rateRefs) > 0 {
		sort.Strings(rateRefs)
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf(
			"expression references %s. vMonitor's formula endpoint rejects rate_* statistics, so switch %s to avg, min, max, sum or count.",
			strings.Join(rateRefs, ", "), pluralItThem(len(rateRefs))))
	}

	series, err := d.client.GetFormula(ctx, vmonitor.FormulaParams{
		Expression:  expr,
		Graphs:      tokenGraphs,
		StartMillis: query.TimeRange.From.UnixMilli(),
		EndMillis:   query.TimeRange.To.UnixMilli(),
		Offset:      qm.offset(),
		Limit:       qm.limitSpec(),
	})
	if err != nil {
		return errorResponse(err)
	}

	series, err = applyReduce(qm.Reduce, series)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, err.Error())
	}
	for i, s := range series {
		if s.Name == "" || s.Name == expr {
			s.Name = qm.Expression
		}
		response.Frames = append(response.Frames, seriesToFrame(qm, s, i))
	}
	return response
}

func pluralIsAre(n int) string {
	if n == 1 {
		return "is"
	}
	return "are"
}

func pluralItThem(n int) string {
	if n == 1 {
		return "it"
	}
	return "them"
}

func buildGraph(qm queryModel) vmonitor.Graph {
	statistic := qm.Statistic
	if statistic == "" {
		statistic = "avg"
	}

	g := vmonitor.Graph{
		Name:       qm.MetricName,
		Dimensions: joinFilters(qm.Filters),
		Statistics: statistic,
		GroupBy:    joinGroupBy(qm.GroupBy),
		Rate:       rateSeconds(qm.Rate),
	}
	if qm.Rank != nil && qm.Rank.Type != "" && qm.Rank.Value != "" {
		g.Limit = qm.Rank.Type + ":" + qm.Rank.Value
	}
	if qm.Rollup != nil && qm.Rollup.Type != "" {
		if secs := unitSeconds(qm.Rollup.Value, qm.Rollup.Unit); secs > 0 {
			g.Rollup = fmt.Sprintf("%s:%d", qm.Rollup.Type, secs)
		}
	}
	if qm.TimeShift != nil {
		g.Offset = unitSeconds(qm.TimeShift.Value, qm.TimeShift.Unit)
	}
	return g
}

func joinFilters(filters []dimensionFilter) string {
	parts := make([]string, 0, len(filters))
	for _, f := range filters {
		if f.Key == "" || f.Value == "" {
			continue
		}
		parts = append(parts, f.Key+":"+f.Value)
	}
	return strings.Join(parts, ",")
}

func joinGroupBy(keys []string) string {
	cleaned := make([]string, 0, len(keys))
	for _, k := range keys {
		if k = strings.TrimSpace(k); k != "" && k != "none" {
			cleaned = append(cleaned, k)
		}
	}
	if len(cleaned) == 0 {
		return "none"
	}
	return strings.Join(cleaned, ",")
}

func unitSeconds(value float64, unit string) int {
	mult := map[string]float64{"minutes": 60, "hours": 3600, "days": 86400, "weeks": 604800}
	m, ok := mult[unit]
	if !ok {
		m = 60
	}
	return int(value * m)
}

func rateSeconds(rate string) int {
	switch rate {
	case "per_second":
		return 1
	case "per_minute":
		return 60
	case "per_hour":
		return 3600
	}
	return 0
}

func periodForInterval(interval time.Duration) int {
	secs := int(interval.Seconds())
	if secs < 60 {
		return 60
	}
	return secs
}

// seriesToFrame converts one vMonitor series into a wide-format Grafana frame.
func seriesToFrame(qm queryModel, s vmonitor.Series, idx int) *data.Frame {
	times := make([]time.Time, 0, len(s.Points))
	values := make([]float64, 0, len(s.Points))
	for _, pt := range s.Points {
		times = append(times, pt.Time)
		values = append(values, pt.Value)
	}

	labels := data.Labels{}
	for k, v := range s.Dimensions {
		labels[k] = v
	}

	name := displayName(qm, s, idx)
	valueField := data.NewField(name, labels, values)
	valueField.Config = &data.FieldConfig{DisplayNameFromDS: name}

	frameName := qm.MetricName
	if frameName == "" {
		frameName = qm.Expression
	}
	return data.NewFrame(frameName,
		data.NewField("time", nil, times),
		valueField,
	)
}

func displayName(qm queryModel, s vmonitor.Series, idx int) string {
	if len(s.Dimensions) > 0 {
		keys := make([]string, 0, len(s.Dimensions))
		for k := range s.Dimensions {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s(%s)", k, s.Dimensions[k]))
		}
		grouped := strings.Join(parts, ",")
		if qm.Alias != "" {
			return qm.Alias + " " + grouped
		}
		return grouped
	}

	if qm.Alias != "" {
		return qm.Alias
	}

	metric := s.Name
	if metric == "" {
		metric = qm.MetricName
	}
	if qm.isExpression() {
		return metric // the expression string itself
	}
	stat := s.ValueColumn
	if stat == "" {
		stat = qm.Statistic
	}
	if stat != "" {
		metric = fmt.Sprintf("%s (%s)", metric, stat)
	}
	if idx > 0 {
		metric = fmt.Sprintf("%s #%d", metric, idx)
	}
	return metric
}

func errorResponse(err error) backend.DataResponse {
	var apiErr *vmonitor.APIError
	if errors.As(err, &apiErr) {
		status := backend.StatusUnknown
		switch apiErr.StatusCode {
		case 401, 403:
			status = backend.StatusUnauthorized
		case 400:
			status = backend.StatusBadRequest
		}
		return backend.ErrDataResponse(status, apiErr.Error())
	}
	return backend.ErrDataResponse(backend.StatusInternal, err.Error())
}

func (d *Datasource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	u, err := url.Parse(req.URL)
	if err != nil {
		return sendJSON(sender, http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	q := u.Query()
	start, end := q.Get("start"), q.Get("end")

	switch strings.Trim(u.Path, "/") {
	case "metric-names":
		names, err := d.client.ListMetricNames(ctx, start, end)
		if err != nil {
			return sendResourceError(sender, err)
		}
		return sendJSON(sender, http.StatusOK, names)

	case "dimension-names":
		names, err := d.client.ListDimensionNames(ctx)
		if err != nil {
			return sendResourceError(sender, err)
		}
		return sendJSON(sender, http.StatusOK, names)

	case "dimensions":
		metric := q.Get("metric")
		if metric == "" {
			return sendJSON(sender, http.StatusBadRequest, map[string]string{"error": "metric is required"})
		}
		dims, err := d.client.GetMetricDimensions(ctx, metric, start, end)
		if err != nil {
			return sendResourceError(sender, err)
		}
		return sendJSON(sender, http.StatusOK, dims)

	case "infrastructure":
		product := q.Get("product")
		if err := checkProduct(product); err != nil {
			return sendJSON(sender, http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		resources, err := d.client.ListInfrastructure(ctx, product, q.Get("search"), q.Get("region"))
		if err != nil {
			return sendResourceError(sender, err)
		}
		return sendJSON(sender, http.StatusOK, resources)

	case "infrastructure-regions":
		product := q.Get("product")
		if err := checkProduct(product); err != nil {
			return sendJSON(sender, http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		regions, err := d.client.ListInfraRegions(ctx, product)
		if err != nil {
			return sendResourceError(sender, err)
		}
		return sendJSON(sender, http.StatusOK, regions)

	case "log-projects":
		projects, err := d.client.ListLogProjects(ctx)
		if err != nil {
			return sendResourceError(sender, err)
		}
		return sendJSON(sender, http.StatusOK, projects)

	case "log-mappings":
		project := q.Get("project")
		if project == "" {
			return sendJSON(sender, http.StatusBadRequest, map[string]string{"error": "project is required"})
		}
		fields, err := d.client.GetLogMappings(ctx, project)
		if err != nil {
			return sendResourceError(sender, err)
		}
		return sendJSON(sender, http.StatusOK, fields)

	case "log-values":
		project := q.Get("project")
		field := q.Get("field")
		if project == "" || field == "" {
			return sendJSON(sender, http.StatusBadRequest, map[string]string{"error": "project and field are required"})
		}
		values, err := d.client.SuggestLogValues(ctx, project, field, q.Get("query"))
		if err != nil {
			return sendResourceError(sender, err)
		}
		return sendJSON(sender, http.StatusOK, values)

	default:
		return sender.Send(&backend.CallResourceResponse{Status: http.StatusNotFound})
	}
}

func checkProduct(product string) error {
	if product == "" {
		return errors.New("product is required")
	}
	if _, ok := vmonitor.LookupInfraProduct(product); !ok {
		return fmt.Errorf("unknown product %q; expected one of %s", product, strings.Join(vmonitor.InfraProducts(), ", "))
	}
	return nil
}

func sendJSON(sender backend.CallResourceResponseSender, status int, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return sender.Send(&backend.CallResourceResponse{
		Status:  status,
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Body:    body,
	})
}

func sendResourceError(sender backend.CallResourceResponseSender, err error) error {
	status := http.StatusBadGateway
	var apiErr *vmonitor.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode > 0 {
		status = apiErr.StatusCode
	}
	return sendJSON(sender, status, map[string]string{"error": err.Error()})
}

func (d *Datasource) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if d.settings.ClientID == "" || d.settings.Secrets == nil || d.settings.Secrets.ClientSecret == "" {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Access key is missing. Enter the IAM Client ID and Client Secret in the datasource settings.",
		}, nil
	}

	if err := d.client.Ping(ctx); err != nil {
		var apiErr *vmonitor.APIError
		if errors.As(err, &apiErr) {
			msg := "vMonitor rejected the request"
			switch apiErr.StatusCode {
			case 400, 401:
				if apiErr.Code == "AUTHENTICATION_FAILED" {
					msg = "Authentication failed: the IAM Client ID / Client Secret is invalid."
				} else {
					msg = "Authentication failed: invalid or expired credentials."
				}
			case 403:
				msg = "Authorization failed: the access key lacks permission (check the API URL / tenant)."
			}
			return &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: msg}, nil
		}
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Could not reach vMonitor: %v", err),
		}, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Connected to vMonitor successfully.",
	}, nil
}
