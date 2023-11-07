package config

import (
	"errors"
	"gopkg.in/yaml.v2"
	"os"
)

// Структура для конфигурации
type Config struct {
	BotToken string `yaml:"BotToken"`
	ChatID   int64  `yaml:"ChatID"`
	Debug    bool   `yaml:"Debug"`
}

// GetConfigValuesFromConfig функция для извлечения нескольких значений из config.yaml
func GetConfigValuesFromConfig(keys ...string) (map[string]interface{}, error) {
	// Открываем файл конфигурации
	configFile, err := os.Open("/home/tiery/Desktop/GO/GolandProjects/MotoBot/config/config.yaml")
	if err != nil {
		return nil, err
	}
	defer configFile.Close()

	// Создаём структуру для конфигурации
	var cfg Config

	// Декодируем содержимое файла конфигурации
	decoder := yaml.NewDecoder(configFile)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}

	configValues := make(map[string]interface{})

	// Перебираем запрошенные ключи и добавляем соответствующие значения в карту
	for _, key := range keys {
		switch key {
		case "BotToken":
			configValues[key] = cfg.BotToken
		case "ChatID":
			configValues[key] = cfg.ChatID
		case "Debug":
			configValues[key] = cfg.Debug
		default:
			return nil, errors.New("Неизвестный ключ конфигурации: " + key)
		}
	}

	return configValues, nil
}
