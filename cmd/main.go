package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/t1ery/MotoBot/config"
	"github.com/t1ery/MotoBot/internal/bot"
	"github.com/t1ery/MotoBot/internal/storage"
	"log"
)

func main() {

	// Здесь мы запрашиваем токены и другие значения из файла конфигурации
	configValues, err := config.GetConfigValuesFromConfig("BotToken", "ChatID", "Debug")
	if err != nil {
		log.Panic(err)
	}

	// Создаем бота с использованием значений из конфигурации
	botAPI, err := tgbotapi.NewBotAPI(configValues["BotToken"].(string))
	if err != nil {
		log.Panic(err)
	}

	chatID, ok := configValues["ChatID"].(int64)
	if !ok {
		log.Panic("ChatID не является корректным int64 значением")
	}

	botAPI.Debug = configValues["Debug"].(bool)

	// Создание хранилища данных (в данном случае, в памяти)
	dataStorage := storage.NewMemoryStorage()

	b, err := bot.NewBot(botAPI.Token, dataStorage, chatID)
	if err != nil {
		log.Panic(err)
	}

	// Добавляем лог для сообщения о запуске бота
	log.Println("Бот запущен!")

	b.Run()
}
