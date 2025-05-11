package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv" 
	"strings" // для разделения списка backends

	"gopkg.in/yaml.v2"
)

type Config struct {
    Port         int      `yaml:"port"`
    Backends     []string `yaml:"backends"`
    RateLimit    struct {
        Capacity   int `yaml:"capacity"`
        RefillRate int `yaml:"refill_rate"`
    } `yaml:"rate_limit"`
    DatabasePath string `yaml:"databasePath"` 
    Strategy     string `yaml:"strategy"` // добавляем стратегию
}

// Load загружает конфигурацию из файла и переменных окружения
func Load(path string) (*Config, error) {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }

    // Дополнительное использование переменных окружения, если они заданы
    if port := os.Getenv("PORT"); port != "" {
        // Преобразуем строку в int
        if p, err := strconv.Atoi(port); err == nil {
            cfg.Port = p
        } else {
            return nil, fmt.Errorf("invalid PORT value: %v", err)
        }
    }

    if backends := os.Getenv("BACKENDS"); backends != "" {
        // Разделяем список бэкендов по запятой и добавляем их
        cfg.Backends = strings.Split(backends, ",")
    }

    if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
        cfg.DatabasePath = dbPath
    }

    if strategy := os.Getenv("STRATEGY"); strategy != "" {
        cfg.Strategy = strategy
    }

    return &cfg, nil
}
