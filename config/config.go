package config

// Структура для конфигурации
type Config struct {
	BotToken string `yaml:"BotToken"`
	ChatID   int64  `yaml:"ChatID"`
	Debug    bool   `yaml:"Debug"`
}
