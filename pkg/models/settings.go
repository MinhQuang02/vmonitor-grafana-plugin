package models

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

const (
	DefaultVMonitorURL    = "https://vmonitorapis.vngcloud.vn/vmonitor-api/api"
	DefaultVMonitorLogURL = "https://vmonitorapis.vngcloud.vn/log-api/v1"
	metricAPIPath         = "/vmonitor-api/api"
	logAPIPath            = "/log-api/v1"
)

type PluginSettings struct {
	URL      string                `json:"url"`
	TokenURL string                `json:"tokenUrl"`
	ClientID string                `json:"clientId"`
	Secrets  *SecretPluginSettings `json:"-"`
	LogURL   string                `json:"-"`
}

type SecretPluginSettings struct {
	ClientSecret string `json:"clientSecret"`
}

func LoadPluginSettings(source backend.DataSourceInstanceSettings) (*PluginSettings, error) {
	settings := PluginSettings{}
	err := json.Unmarshal(source.JSONData, &settings)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal PluginSettings json: %w", err)
	}

	if settings.URL == "" {
		settings.URL = DefaultVMonitorURL
	}
	settings.LogURL = deriveLogURL(settings.URL)

	settings.Secrets = loadSecretPluginSettings(source.DecryptedSecureJSONData)

	return &settings, nil
}

func deriveLogURL(metricURL string) string {
	if strings.Contains(metricURL, metricAPIPath) {
		return strings.Replace(metricURL, metricAPIPath, logAPIPath, 1)
	}
	return DefaultVMonitorLogURL
}

func loadSecretPluginSettings(source map[string]string) *SecretPluginSettings {
	return &SecretPluginSettings{
		ClientSecret: source["clientSecret"],
	}
}
