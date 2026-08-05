package vmonitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type LogProject struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	ProjectType string           `json:"projectType"`
	Extra       *LogProjectExtra `json:"extra"`
}

type LogProjectExtra struct {
	PreRenderSystemFields []string           `json:"preRenderSystemFields"`
	CustomMappings        []LogCustomMapping `json:"customMappings"`
}

type LogCustomMapping struct {
	SystemField  string   `json:"systemField"`
	PortalFields []string `json:"portalFields"`
}

type LogField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	ElasticType string `json:"elasticType"`
	IsAgg       bool   `json:"isAgg"`
}

type logProjectPage struct {
	Content       []LogProject `json:"content"`
	TotalPages    int          `json:"totalPages"`
	TotalElements int          `json:"totalElements"`
	CurrentPage   int          `json:"currentPage"`
}

const logProjectPageSize = 50

func (c *Client) ListLogProjects(ctx context.Context) ([]LogProject, error) {
	var out []LogProject
	for page := 0; ; page++ {
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("size", strconv.Itoa(logProjectPageSize))
		q.Set("billing_status", "ACTIVE")

		body, err := c.logGet(ctx, "/projects", q)
		if err != nil {
			return nil, err
		}
		var pg logProjectPage
		if err := json.Unmarshal(body, &pg); err != nil {
			return nil, fmt.Errorf("decode log projects: %w", err)
		}
		out = append(out, pg.Content...)
		if len(pg.Content) < logProjectPageSize || page+1 >= pg.TotalPages {
			break
		}
	}
	return out, nil
}

func (c *Client) GetLogMappings(ctx context.Context, projectID string) ([]LogField, error) {
	q := url.Values{}
	q.Set("refresh", "false")
	body, err := c.logGet(ctx, "/projects/"+projectID+"/mappings", q)
	if err != nil {
		return nil, err
	}
	var fields []LogField
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("decode log mappings: %w", err)
	}
	return fields, nil
}

type LogSearchRequest struct {
	From         int              `json:"from"`
	Size         int              `json:"size"`
	Aggregations []map[string]any `json:"aggregations"`
	Query        map[string]any   `json:"query"`
	Sorts        []map[string]any `json:"sorts"`
}

type LogHit struct {
	ID     string          `json:"_id"`
	Index  string          `json:"_index"`
	Source json.RawMessage `json:"_source"`
	Sort   []json.Number   `json:"sort"`
}

type LogAggregation struct {
	Buckets []LogBucket `json:"buckets"`
}

type LogBucket struct {
	Key          json.RawMessage
	KeyAsString  string
	DocCount     int64
	Aggregations map[string]LogAggregation
}

func (b *LogBucket) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*b = LogBucket{Aggregations: map[string]LogAggregation{}}
	for name, value := range raw {
		switch name {
		case "key":
			b.Key = value
		case "key_as_string":
			if err := json.Unmarshal(value, &b.KeyAsString); err != nil {
				return fmt.Errorf("decode bucket key_as_string: %w", err)
			}
		case "doc_count":
			if err := json.Unmarshal(value, &b.DocCount); err != nil {
				return fmt.Errorf("decode bucket doc_count: %w", err)
			}
		default:
			var sub LogAggregation
			if err := json.Unmarshal(value, &sub); err == nil && sub.Buckets != nil {
				b.Aggregations[name] = sub
			}
		}
	}
	return nil
}

// KeyMillis reads the bucket key of a date_histogram, which is epoch millis.
func (b LogBucket) KeyMillis() (int64, bool) {
	var millis int64
	if err := json.Unmarshal(b.Key, &millis); err != nil {
		return 0, false
	}
	return millis, true
}

func (b LogBucket) KeyLabel() string {
	var text string
	if err := json.Unmarshal(b.Key, &text); err == nil {
		return text
	}
	if b.KeyAsString != "" {
		return b.KeyAsString
	}
	return string(b.Key)
}

type LogSearchResult struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []LogHit `json:"hits"`
	} `json:"hits"`
	Aggregations map[string]LogAggregation `json:"aggregations"`
}

func (c *Client) SearchLogs(ctx context.Context, projectID string, req LogSearchRequest) (*LogSearchResult, error) {
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode log search request: %w", err)
	}
	body, err := c.logPost(ctx, "/projects/"+projectID+"/search-logs", raw)
	if err != nil {
		return nil, err
	}
	var result LogSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode log search result: %w", err)
	}
	return &result, nil
}

func (c *Client) SuggestLogValues(ctx context.Context, projectID, field, contains string) ([]string, error) {
	aggValue := map[string]any{
		"name":        "suggestions",
		"field":       field,
		"size":        50,
		"minDocCount": 1,
	}
	if contains != "" {
		aggValue["includeRegExp"] = ".*" + contains + ".*"
	}
	req := LogSearchRequest{
		From:  0,
		Size:  0,
		Query: map[string]any{"type": "match_all", "value": map[string]any{}},
		Aggregations: []map[string]any{{
			"aggregationQuery": map[string]any{"type": "terms", "value": aggValue},
			"subAggregation":   nil,
		}},
		Sorts: []map[string]any{},
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode value suggestion request: %w", err)
	}
	body, err := c.logPost(ctx, "/projects/"+projectID+"/search-logs", raw)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Aggregations struct {
			Suggestions struct {
				Buckets []struct {
					Key         json.RawMessage `json:"key"`
					KeyAsString string          `json:"key_as_string"`
				} `json:"buckets"`
			} `json:"suggestions"`
		} `json:"aggregations"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode value suggestions: %w", err)
	}
	values := make([]string, 0, len(parsed.Aggregations.Suggestions.Buckets))
	for _, bucket := range parsed.Aggregations.Suggestions.Buckets {
		var text string
		if err := json.Unmarshal(bucket.Key, &text); err == nil {
			values = append(values, text)
			continue
		}
		if bucket.KeyAsString != "" {
			values = append(values, bucket.KeyAsString)
			continue
		}
		values = append(values, string(bucket.Key))
	}
	return values, nil
}

func logClause(typ string, value map[string]any) map[string]any {
	return map[string]any{"type": typ, "value": value}
}

func nonNilClauses(s []map[string]any) []map[string]any {
	if s == nil {
		return []map[string]any{}
	}
	return s
}

func LogBoolQuery(filter, must, mustNot, should []map[string]any) map[string]any {
	return logClause("bool", map[string]any{
		"filter":  nonNilClauses(filter),
		"must":    nonNilClauses(must),
		"mustNot": nonNilClauses(mustNot),
		"should":  nonNilClauses(should),
	})
}

func LogMatch(field, value string) map[string]any {
	return logClause("match", map[string]any{"field": field, "value": value})
}

func LogExists(field string) map[string]any {
	return logClause("exists", map[string]any{"field": field})
}

func LogRange(field string, bounds map[string]any) map[string]any {
	value := map[string]any{"field": field}
	for k, b := range bounds {
		value[k] = b
	}
	return logClause("range", value)
}

func LogTimeRange(field, gte, lte string) map[string]any {
	return logClause("range", map[string]any{
		"field":  field,
		"gte":    gte,
		"lte":    lte,
		"format": "strict_date_optional_time",
	})
}

func LogFieldSort(field, order string) map[string]any {
	return logClause("field_sort", map[string]any{
		"field":        field,
		"order":        order,
		"mode":         "min",
		"missing":      "_last",
		"unmappedType": "boolean",
	})
}

type LogGroupBy struct {
	Name  string
	Field string
	Size  int
}

func LogCountAggregation(name, field, interval string, buckets int, groups []LogGroupBy) map[string]any {
	return map[string]any{
		"aggregationQuery": map[string]any{
			"type": "date_histogram",
			"value": map[string]any{
				"name":         name,
				"field":        field,
				"timeInterval": interval,
				"minDocCount":  0,
				"size":         buckets,
			},
		},
		"subAggregation": logTermsChain(groups),
	}
}

func logTermsChain(groups []LogGroupBy) []map[string]any {
	if len(groups) == 0 {
		return nil
	}
	group := groups[0]
	return []map[string]any{{
		"aggregationQuery": map[string]any{
			"type": "terms",
			"value": map[string]any{
				"name":  group.Name,
				"field": group.Field,
				"size":  group.Size,
			},
		},
		"subAggregation": logTermsChain(groups[1:]),
	}}
}
