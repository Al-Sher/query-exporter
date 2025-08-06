package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// Config конфигурация приложения
type Config struct {
	Dsn         string `yaml:"dsn"`
	QueriesPath string `yaml:"queries_path"`
	Timeout     int    `yaml:"timeout"`
	Interval    int    `yaml:"interval"`
	PoolSize    int    `yaml:"pool_size"`
	MetricsPath string `yaml:"metrics_path"`
}

// LoadConfig Загрузка конфига из файла
func LoadConfig(configPath string) (Config, error) {
	c := defaultConfig()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return *c, fmt.Errorf("не вышло открыть файл конфигурации %s: %w", configPath, err)
	}

	err = yaml.Unmarshal(data, c)
	if err != nil {
		return *c, fmt.Errorf("не вышло распарсить файл конфигурации: %v", err)
	}

	return *c, nil
}

// defaultConfig установка значений по умолчанию в конфиг
func defaultConfig() *Config {
	return &Config{
		Dsn:         "",
		QueriesPath: "./queries/",
		Timeout:     30,
		Interval:    5,
		PoolSize:    2,
		MetricsPath: "/metrics",
	}
}
