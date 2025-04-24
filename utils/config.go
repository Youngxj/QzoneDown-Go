package utils

import (
	"encoding/json"
	"os"
)

// Config 配置结构体
type Config struct {
	GTk     string `json:"g_tk"`
	ResUin  string `json:"res_uin"`
	Cookie  string `json:"cookie"`
	Threads int    `json:"threads"`
}

// LoadConfig 从文件加载配置
func LoadConfig() (*Config, error) {
	configFile := "config.json"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return &Config{}, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("config.json", data, 0644)
}