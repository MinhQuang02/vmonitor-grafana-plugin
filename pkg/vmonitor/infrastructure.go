package vmonitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
)

type InfraProduct struct {
	Product     string
	Path        string
	IDField     string
	NameField   string
	Dimension   string
	SearchParam string
	Regional    bool
}

var infraProducts = map[string]InfraProduct{
	"host":            {Product: "", Path: "hosts", IDField: "name", NameField: "name", Dimension: "host", SearchParam: "searching_text"},
	"vserver":         {Product: "vserver", Path: "vserver/hosts", IDField: "server_id", NameField: "server_name", Dimension: "resource_id", SearchParam: "name"},
	"vlb":             {Product: "vlb", Path: "vlb/hosts", IDField: "load_balancer_id", NameField: "load_balancer_name", Dimension: "loadbalancer_id", SearchParam: "name"},
	"vstorage":        {Product: "vstorage", Path: "vstorage/hosts", IDField: "vstorage_id", NameField: "vstorage_name", Dimension: "project_id", SearchParam: "name"},
	"vstorage-bucket": {Product: "vstorage", Path: "vstorage-ceph/hosts", IDField: "bucket", NameField: "bucket", Dimension: "bucket", SearchParam: "bucket", Regional: true},
	"vdb":             {Product: "vdb", Path: "vdb/hosts", IDField: "database_id", NameField: "database_name", Dimension: "database_id", SearchParam: "name"},
	"vdb-kafka":       {Product: "vkafka", Path: "vdb-kafka/hosts", IDField: "database_id", NameField: "database_name", Dimension: "cluster_id", SearchParam: "name"},
	"vdb-cluster":     {Product: "vdb", Path: "vdb-cluster/hosts", IDField: "database_id", NameField: "database_name", Dimension: "database_id", SearchParam: "name"},
	"vbackup":         {Product: "vbackup", Path: "vbackup/hosts", IDField: "vbackup_id", NameField: "vbackup_name", Dimension: "backup_location_id", SearchParam: "name"},
}

func InfraProducts() []string {
	keys := make([]string, 0, len(infraProducts))
	for k := range infraProducts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func LookupInfraProduct(product string) (InfraProduct, bool) {
	p, ok := infraProducts[product]
	return p, ok
}

type InfraResource struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Region         string `json:"region,omitempty"`
	MonitorEnabled bool   `json:"monitorEnabled"`
}

type infraPage struct {
	Page      int              `json:"page"`
	PageSize  int              `json:"pageSize"`
	TotalItem int              `json:"totalItem"`
	LstData   []map[string]any `json:"lstData"`
}

const maxInfraItems = 2000
const infraPageSize = 200

func (c *Client) ListInfrastructure(ctx context.Context, product, search, region string) ([]InfraResource, error) {
	p, ok := infraProducts[product]
	if !ok {
		return nil, fmt.Errorf("unknown product %q; expected one of %v", product, InfraProducts())
	}

	var out []InfraResource
	for page := 1; len(out) < maxInfraItems; page++ {
		q := url.Values{}
		q.Set(p.SearchParam, search)
		q.Set("page", strconv.Itoa(page))
		q.Set("size", strconv.Itoa(infraPageSize))
		if p.SearchParam == "searching_text" {
			q.Set("searching_field", "")
			q.Set("filter", "")
		}
		if p.Regional && region != "" {
			q.Set("region", region)
		}

		body, err := c.get(ctx, "/v1/infrastructure/"+p.Path, q)
		if err != nil {
			return nil, err
		}
		var pg infraPage
		if err := json.Unmarshal(body, &pg); err != nil {
			return nil, fmt.Errorf("decode %s infrastructure list: %w", product, err)
		}
		for _, item := range pg.LstData {
			id := stringField(item[p.IDField])
			if id == "" {
				continue
			}
			name := stringField(item[p.NameField])
			if name == "" {
				name = id
			}
			enabled, ok := item["monitor_enabled"].(bool)
			if !ok {
				enabled, _ = item["enabled"].(bool)
			}
			out = append(out, InfraResource{
				ID:             id,
				Name:           name,
				Region:         stringField(item["region"]),
				MonitorEnabled: enabled,
			})
		}
		if len(pg.LstData) < infraPageSize || len(out) >= pg.TotalItem {
			break
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (c *Client) ListInfraRegions(ctx context.Context, product string) ([]string, error) {
	p, ok := infraProducts[product]
	if !ok {
		return nil, fmt.Errorf("unknown product %q; expected one of %v", product, InfraProducts())
	}
	if !p.Regional {
		return nil, fmt.Errorf("product %q is not region-scoped", product)
	}
	resources, err := c.ListInfrastructure(ctx, product, "", "")
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(resources))
	var regions []string
	for _, r := range resources {
		if r.Region == "" {
			continue
		}
		if _, dup := seen[r.Region]; dup {
			continue
		}
		seen[r.Region] = struct{}{}
		regions = append(regions, r.Region)
	}
	sort.Strings(regions)
	return regions, nil
}

func stringField(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	default:
		return ""
	}
}
