package main

import (
	"log"
	"os"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/t1ery/MotoBot/config"
	"github.com/t1ery/MotoBot/internal/bot"
	"github.com/t1ery/MotoBot/internal/storage"
	"gopkg.in/yaml.v2"
)

func main() {
	// Открываем файл конфигурации
	configFile, err := os.Open("/home/tiery/Desktop/GO/GolandProjects/MotoBot/config/config.yaml")
	if err != nil {
		log.Panic("Ошибка при открытии файла конфигурации: ", err)
	}
	defer configFile.Close()

	// Создаем структуру для конфигурации
	var cfg config.Config

	// Декодируем содержимое файла конфигурации
	decoder := yaml.NewDecoder(configFile)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatalf("Ошибка при декодировании файла конфигурации: %v", err)
	}

	botAPI, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Panic(err)
	}

	botAPI.Debug = cfg.Debug

	// Создание хранилища данных (в данном случае, в памяти)
	dataStorage := storage.NewMemoryStorage()

	b, err := bot.NewBot(botAPI.Token, dataStorage, cfg.ChatID)
	if err != nil {
		log.Panic(err)
	}

	// Добавляем лог для сообщения о запуске бота
	log.Println("Бот запущен!")

	b.Run()
}
