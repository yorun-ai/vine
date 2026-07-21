package redised

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	configKeyFormat = "config:%s"
)

type ConfigValue struct {
	Name        string          `json:"name"`
	Value       json.RawMessage `json:"value"`
	LastUpdated time.Time       `json:"lastUpdated"`
}

func FormatConfigKey(configName string) string {
	return fmt.Sprintf(configKeyFormat, configName)
}
