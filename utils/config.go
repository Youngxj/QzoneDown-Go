package utils

import (
	"encoding/json"
	"os"
)

var ConfigFile = "config.json"

// Configs 配置项
type Configs struct {
	Cookie string `json:"cookie"`
	GTk    string `json:"gtk"`
	Uin    string `json:"uin"`
}

// LoadConfig 从文件加载配置
func LoadConfig() (*Configs, error) {
	if _, err := os.Stat(ConfigFile); os.IsNotExist(err) {
		return &Configs{}, nil
	}
	data, err := os.ReadFile(ConfigFile)
	if err != nil {
		return nil, err
	}
	var config Configs
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// SaveConfig 写入配置
func SaveConfig(config *Configs) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFile, data, 0644)
}
