package vmonitor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	logBaseURL string
	tokens     *TokenSource
	http       *http.Client
}

func NewClient(baseURL, logBaseURL string, tokens *TokenSource) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		logBaseURL: strings.TrimRight(logBaseURL, "/"),
		tokens:     tokens,
		http:       &http.Client{Timeout: 60 * time.Second},
	}
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != "" || e.Message != "" {
		return fmt.Sprintf("vMonitor API %d: %s %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("vMonitor API returned HTTP %d", e.StatusCode)
}

func (c *Client) get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.request(ctx, c.baseURL, http.MethodGet, path, query, nil)
}

func (c *Client) post(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.request(ctx, c.baseURL, http.MethodPost, path, nil, body)
}

func (c *Client) logGet(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.request(ctx, c.logBaseURL, http.MethodGet, path, query, nil)
}

func (c *Client) logPost(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.request(ctx, c.logBaseURL, http.MethodPost, path, nil, body)
}

func (c *Client) request(ctx context.Context, baseURL, method, path string, query url.Values, body []byte) ([]byte, error) {
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("obtain token: %w", err)
	}

	respBody, err := c.doRequest(ctx, baseURL, method, path, query, body, token)
	if !isUnauthorized(err) {
		return respBody, err
	}

	c.tokens.Invalidate(token)
	token, err = c.tokens.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("obtain token: %w", err)
	}
	return c.doRequest(ctx, baseURL, method, path, query, body, token)
}

func isUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized
}

func (c *Client) doRequest(ctx context.Context, baseURL, method, path string, query url.Values, body []byte, token string) ([]byte, error) {
	u := baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call vMonitor: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}
	return respBody, nil
}

func parseAPIError(status int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: status}

	var arr []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &arr); err == nil && len(arr) > 0 {
		apiErr.Code = arr[0].Code
		apiErr.Message = arr[0].Message
		return apiErr
	}

	var obj struct {
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &obj); err == nil {
		if len(obj.Errors) > 0 {
			apiErr.Code = obj.Errors[0].Code
			apiErr.Message = obj.Errors[0].Message
		} else if obj.Message != "" {
			apiErr.Message = obj.Message
		}
	}
	return apiErr
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.get(ctx, "/v1/metrics/metric-name", nil)
	return err
}

func (c *Client) ListMetricNames(ctx context.Context, start, end string) ([]string, error) {
	q := url.Values{}
	if start != "" {
		q.Set("start_time", start)
	}
	if end != "" {
		q.Set("end_time", end)
	}
	body, err := c.get(ctx, "/v1/metrics/metric-name", q)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode metric names: %w", err)
	}
	names := make([]string, 0, len(raw))
	for _, m := range raw {
		names = append(names, m.Name)
	}
	return names, nil
}

func (c *Client) ListDimensionNames(ctx context.Context) ([]string, error) {
	body, err := c.get(ctx, "/v1/metrics/dimensions-names", nil)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		DimensionName string `json:"dimension_name"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode dimension names: %w", err)
	}
	names := make([]string, 0, len(raw))
	for _, d := range raw {
		names = append(names, d.DimensionName)
	}
	return names, nil
}

type Dimension struct {
	Key    string   `json:"key"`
	Values []string `json:"value"`
}

func (c *Client) GetMetricDimensions(ctx context.Context, name, start, end string) ([]Dimension, error) {
	q := url.Values{}
	q.Set("name", name)
	if start != "" {
		q.Set("start_time", start)
	}
	if end != "" {
		q.Set("end_time", end)
	}
	body, err := c.get(ctx, "/v1/metrics/dimensions", q)
	if err != nil {
		return nil, err
	}
	var dims []Dimension
	if err := json.Unmarshal(body, &dims); err != nil {
		return nil, fmt.Errorf("decode dimensions: %w", err)
	}
	return dims, nil
}

type StatisticsParams struct {
	Name        string
	Statistics  string
	Dimensions  string
	StartMillis int64
	EndMillis   int64
	GroupBy     string
	Limit       string
}

type Series struct {
	ID          string
	Name        string
	Dimensions  map[string]string
	ValueColumn string
	Points      []DataPoint
}

type DataPoint struct {
	Time  time.Time
	Value float64
}

type statElem struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Dimensions map[string]string   `json:"dimensions"`
	Columns    []string            `json:"columns"`
	Statistics [][]json.RawMessage `json:"statistics"`
}

func (c *Client) GetStatistics(ctx context.Context, p StatisticsParams) ([]Series, error) {
	q := url.Values{}
	q.Set("name", p.Name)
	q.Set("statistics", p.Statistics)
	q.Set("dimensions", p.Dimensions)
	q.Set("start_time", strconv.FormatInt(p.StartMillis, 10))
	q.Set("end_time", strconv.FormatInt(p.EndMillis, 10))
	groupBy := p.GroupBy
	if groupBy == "" {
		groupBy = "none"
	}
	q.Set("group_by", groupBy)
	q.Set("alarm", "false")
	q.Set("limit", p.Limit)

	body, err := c.get(ctx, "/v1/statistics", q)
	if err != nil {
		return nil, err
	}
	return decodeSeries(body, p.Name)
}

type Graph struct {
	Name       string `json:"name"`
	Dimensions string `json:"dimensions"`
	Statistics string `json:"statistics"`
	GroupBy    string `json:"group_by"`
	Offset     int    `json:"offset"`
	Limit      string `json:"limit"`
	Rollup     string `json:"rollup"`
	Rate       int    `json:"rate"`
}

type V2Params struct {
	Graph       Graph
	StartMillis int64
	EndMillis   int64
	Period      int
}

func (c *Client) GetStatisticsV2(ctx context.Context, p V2Params) ([]Series, error) {
	period := p.Period
	if period <= 0 {
		period = 60
	}
	if p.Graph.GroupBy == "" {
		p.Graph.GroupBy = "none"
	}

	payload := map[string]any{
		"type": "SIMPLE",
		"data": map[string]any{
			"graph":      p.Graph,
			"start_time": p.StartMillis,
			"end_time":   p.EndMillis,
			"period":     period,
			"alarm":      false,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode statistics request: %w", err)
	}

	body, err := c.post(ctx, "/v1/statistics", raw)
	if err != nil {
		return nil, err
	}
	return decodeSeries(body, p.Graph.Name)
}

type FormulaParams struct {
	Expression  string
	Graphs      map[string]Graph
	StartMillis int64
	EndMillis   int64
	Offset      int
	Limit       string
}

func (c *Client) GetFormula(ctx context.Context, p FormulaParams) ([]Series, error) {
	payload := map[string]any{
		"type": "CUSTOM",
		"data": map[string]any{
			"expression": p.Expression,
			"graphs":     p.Graphs,
			"start_time": p.StartMillis,
			"end_time":   p.EndMillis,
			"alarm":      false,
			"offset":     p.Offset,
			"limit":      p.Limit,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode formula request: %w", err)
	}
	body, err := c.post(ctx, "/v1/statistics", raw)
	if err != nil {
		return nil, err
	}
	return decodeSeries(body, p.Expression)
}

func decodeSeries(body []byte, fallbackName string) ([]Series, error) {
	var raw []statElem
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode statistics: %w", err)
	}
	series := make([]Series, 0, len(raw))
	for _, elem := range raw {
		s := Series{ID: elem.ID, Name: elem.Name, Dimensions: elem.Dimensions}
		if s.Name == "" {
			s.Name = fallbackName
		}
		if s.Dimensions == nil {
			s.Dimensions = map[string]string{}
		}
		if len(elem.Columns) >= 2 {
			s.ValueColumn = elem.Columns[1]
		}
		pts, err := parsePoints(elem.Statistics)
		if err != nil {
			return nil, err
		}
		s.Points = pts
		series = append(series, s)
	}
	return series, nil
}

func parsePoints(pairs [][]json.RawMessage) ([]DataPoint, error) {
	points := make([]DataPoint, 0, len(pairs))
	for _, pair := range pairs {
		if len(pair) < 2 {
			continue
		}
		ts, ok := toTime(pair[0])
		if !ok {
			continue
		}
		val, ok := toFloat(pair[1])
		if !ok {
			continue
		}
		points = append(points, DataPoint{Time: ts, Value: val})
	}
	return points, nil
}

func toTime(raw json.RawMessage) (time.Time, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t.UTC(), true
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return epochToTime(f), true
		}
		return time.Time{}, false
	}
	if f, ok := toFloat(raw); ok {
		return epochToTime(f), true
	}
	return time.Time{}, false
}

func toFloat(raw json.RawMessage) (float64, bool) {
	if string(raw) == "null" {
		return 0, false
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f, true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func epochToTime(v float64) time.Time {
	switch {
	case v >= 1e15:
		return time.UnixMicro(int64(v)).UTC()
	case v >= 1e12:
		return time.UnixMilli(int64(v)).UTC()
	default:
		sec := int64(v)
		nsec := int64((v - float64(sec)) * 1e9)
		return time.Unix(sec, nsec).UTC()
	}
}
