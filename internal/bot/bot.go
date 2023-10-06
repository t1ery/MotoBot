package bot

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/t1ery/MotoBot/internal/storage"
	"github.com/t1ery/MotoBot/internal/user"
)

// Bot представляет интерфейс для взаимодействия с ботом.
type Bot interface {
	CreateProfile(userID int, updates <-chan tgbotapi.Update) error // Создание анкеты
	EditProfile(userID int, updates <-chan tgbotapi.Update) error   // Редактирование анкеты
	DeleteProfile(userID int) error                                 // Удаление анкеты
	SendProfile(profile *user.Profile) error                        // Отправка анкеты в соответствующую тему
	GetProjectInfo(chatID int64) error                              // Предоставление информации о проекте пользователю
	Run()                                                           // Запуск бота
}

// BotImpl представляет реализацию интерфейса Bot.
type MotoBot struct {
	bot         *tgbotapi.BotAPI
	dataStorage storage.Storage
	chatID      int64
	token       string
}

// NewBot создает новый экземпляр бота.
func NewBot(token string, dataStorage storage.Storage, chatID int64) (Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &MotoBot{
		bot:         bot,
		dataStorage: dataStorage,
		chatID:      chatID,
		token:       token,
	}, nil
}

// Run запускает бота и начинает обработку обновлений.
func (b *MotoBot) Run() {

	log.Printf("Бот подписан на обновления к чату - ChatID: %d\n", b.chatID)

	// Настроим обработку обновлений
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates, err := b.bot.GetUpdatesChan(updateConfig)
	if err != nil {
		log.Panic(err)
	}

	for update := range updates {
		// Обработка каждого обновления
		if update.Message != nil {
			// Проверяем событие вступления новых участников
			if update.Message.NewChatMembers != nil {
				for _, newUser := range *update.Message.NewChatMembers {
					// Приветствуем нового участника и отправляем инлайн клавиатуру
					err := b.welcomeNewUser(newUser.ID, update.Message.Chat.ID)
					if err != nil {
						log.Printf("Ошибка при приветствии нового участника: %v", err)
					}
				}
			} else if update.Message.IsCommand() {
				// Обработка текстовых команд
				switch update.Message.Command() {
				case "info":
					// Обработка команды "/info"
					err := b.GetProjectInfo(update.Message.Chat.ID)
					if err != nil {
						log.Printf("Ошибка при отправке информации: %v", err)
					}
				case "start":
					// Обработка команды "/start"
					err := b.CreateProfile(update.Message.From.ID, updates)
					if err != nil {
						log.Printf("Ошибка при создании анкеты: %v", err)
					}
				case "edit":
					// Обработка команды "/edit"
					err := b.EditProfile(update.Message.From.ID, updates)
					if err != nil {
						log.Printf("Ошибка при попытке редактирования анкеты: %v", err)
					}
				case "delete":
					// Обработка команды "/delete"
					err := b.DeleteProfile(update.Message.From.ID)
					if err != nil {
						log.Printf("Ошибка при попытке удаления анкеты: %v", err)
					}
				default:
					// Обработка неизвестных команд
					err := b.sendUnknownCommandMessage(update.Message.Chat.ID)
					if err != nil {
						log.Printf("Ошибка при отправке сообщения с неизвестной командой: %v", err)
					}
				}
			}
		}

		// Обработка текстовых команд
		if update.CallbackQuery != nil {
			// Получаем данные, связанные с CallbackQuery
			callbackData := update.CallbackQuery.Data
			switch callbackData {
			case "/info":
				// Обработка команды "Информация"
				err := b.GetProjectInfo(update.CallbackQuery.Message.Chat.ID)
				if err != nil {
					log.Printf("Ошибка при отправке информации: %v", err)
				}
			case "/start":
				// Обработка команды "Создание анкеты"
				log.Printf("Отправка сообщения пользователю с ID: %d", update.CallbackQuery.From.ID)
				err := b.CreateProfile(update.CallbackQuery.From.ID, updates)
				if err != nil {
					log.Printf("Ошибка при создании анкеты: %v", err)
				}
			case "/edit":
				// Обработка команды "Редактирование анкеты"
				err := b.EditProfile(update.CallbackQuery.From.ID, updates)
				if err != nil {
					log.Printf("Ошибка при попытке редактирования анкеты: %v", err)
				}
			case "/delete":
				// Обработка команды "Удаление анкеты"
				err := b.DeleteProfile(update.CallbackQuery.From.ID)
				if err != nil {
					log.Printf("Ошибка при попытке удаления анкеты: %v", err)
				}
			}
		}
	}
}

func (b *MotoBot) CreateProfile(userID int, updates <-chan tgbotapi.Update) error {
	// Получаем профиль пользователя из хранилища
	profile, err := b.dataStorage.GetProfile(userID)
	if err != nil {
		// Если профиль не найден, создаем новую анкету
		if profile == nil {
			profile = &user.Profile{
				UserID: userID,
			}
		}

		// Переменная состояния для отслеживания текущего шага создания анкеты
		creationState := user.StepFirstName

		for {
			// Отправляем запрос данных в зависимости от текущего шага
			var message tgbotapi.MessageConfig
			switch creationState {
			case user.StepFirstName:
				message = tgbotapi.NewMessage(int64(userID), "Шаг 1: Пожалуйста, предоставьте ваше имя:")
			case user.StepLastName:
				message = tgbotapi.NewMessage(int64(userID), "Шаг 2: Пожалуйста, предоставьте вашу фамилию:")
			case user.StepAge:
				message = tgbotapi.NewMessage(int64(userID), "Шаг 3: Пожалуйста, предоставьте ваш возраст:")
			case user.StepInterests:
				message = tgbotapi.NewMessage(int64(userID), "Шаг 4: Пожалуйста, предоставьте ваши интересы:")
			case user.StepPhoto:
				message = tgbotapi.NewMessage(int64(userID), "Шаг 5: Пожалуйста, загрузите 2-3 фотографии на ваш выбор::")
			case user.StepIsDriver:
				inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Да", "is_driver_yes"),
						tgbotapi.NewInlineKeyboardButtonData("Нет", "is_driver_no"),
					),
				)
				message = tgbotapi.NewMessage(int64(userID), "Шаг 6: Вы являетесь водителем?")
				message.ReplyMarkup = inlineKeyboard
			case user.StepCompleted:
				break
			}

			// Отправляем сообщение
			_, err := b.bot.Send(message)
			if err != nil {
				return err
			}

			// Ожидаем ответ от пользователя
			userUpdate, ok := <-updates
			if !ok {
				// Канал закрыт, завершаем выполнение
				return nil
			}
			log.Printf("userUpdate: %+v", userUpdate)
			log.Printf("userUpdate.Message: %+v", userUpdate.Message)
			if userUpdate.Message != nil && userUpdate.Message.Text != "" {
				switch creationState {
				case user.StepFirstName:
					profile.FirstName = userUpdate.Message.Text
					creationState = user.StepLastName
				case user.StepLastName:
					profile.LastName = userUpdate.Message.Text
					creationState = user.StepAge
				case user.StepAge:
					age, err := strconv.Atoi(userUpdate.Message.Text)
					if err == nil {
						profile.Age = age
					}
					creationState = user.StepInterests
				case user.StepInterests:
					profile.Interests = userUpdate.Message.Text
					creationState = user.StepPhoto
				case user.StepPhoto:
					// Проверяем, есть ли фотографии в сообщении
					if userUpdate.Message != nil {
						log.Printf("userUpdate: %+v", userUpdate)
						log.Printf("userUpdate.Message: %+v", userUpdate.Message)
						log.Println("Обработка текстовых сообщений")

						if *userUpdate.Message.Photo != nil {
							log.Printf("Фотографии в сообщении: %d", len(*userUpdate.Message.Photo))
							if len(*userUpdate.Message.Photo) > 0 {
								// Получаем информацию о файле фотографии
								photo := (*userUpdate.Message.Photo)[len(*userUpdate.Message.Photo)-1] // Берем последнюю (самую большую) фотографию
								log.Printf("FileID фотографии: %s", photo.FileID)
								log.Printf("Размер фотографии: %d", photo.FileSize)

								fileConfig := tgbotapi.FileConfig{FileID: photo.FileID}
								photoFile, err := b.bot.GetFile(fileConfig)
								if err != nil {
									log.Printf("Ошибка при получении файла фотографии: %v", err)
									// Обработка ошибки - возможно, стоит уведомить пользователя
								} else {
									log.Printf("FileID файла фотографии: %s", photoFile.FileID)
									log.Printf("Путь файла фотографии: %s", photoFile.FilePath)

									// Загружаем фотографию
									photoBytes, err := b.DownloadPhoto(photoFile.FilePath)
									if err != nil {
										log.Printf("Ошибка при загрузке файла фотографии: %v", err)
										// Обработка ошибки - возможно, стоит уведомить пользователя
									} else {
										// Добавляем []byte фотографии в срез Photos
										profile.Photos = append(profile.Photos, photoBytes)

										log.Printf("Фотография успешно загружена и добавлена в профиль")
									}
								}
							} else {
								log.Println("Нет фотографий в сообщении")
							}
						} else {
							log.Println("Сообщение не содержит фотографий")
						}
					} else {
						log.Println("Сообщение пустое")
					}

					creationState = user.StepIsDriver
				case user.StepIsDriver:
					if userUpdate.Message.Text == "Да" {
						profile.IsDriver = true
					} else {
						profile.IsDriver = false
					}
					creationState = user.StepCompleted
				}
			} else if userUpdate.CallbackQuery != nil {
				if userUpdate.CallbackQuery.Data == "is_driver_yes" {
					profile.IsDriver = true
				} else if userUpdate.CallbackQuery.Data == "is_driver_no" {
					profile.IsDriver = false
				}
				creationState = user.StepCompleted
			}

			// Выход из цикла, если текущий шаг завершен
			if creationState == user.StepCompleted {
				break
			}
		}

		// Сохраняем профиль в хранилище
		err = b.dataStorage.SaveProfile(profile)
		if err != nil {
			return err
		}

		// После завершения всех шагов, отправляем анкету в группу
		err := b.SendProfile(profile)
		if err != nil {
			return err
		}

		// Отправляем сообщение об успешном создании анкеты
		message := tgbotapi.NewMessage(int64(userID), "Ваша анкета успешно создана и отправлена в группу.")
		_, err = b.bot.Send(message)
		if err != nil {
			return err
		}
	} else {
		message := tgbotapi.NewMessage(int64(userID), "Вы уже создали анкету.")
		_, err := b.bot.Send(message)
		if err != nil {
			return err
		}
	}

	return nil
}

// Редактирование анкеты
func (b *MotoBot) EditProfile(userID int, updates <-chan tgbotapi.Update) error {
	// Получите профиль пользователя из хранилища
	profile, err := b.dataStorage.GetProfile(userID)
	if err != nil {
		// Если профиль не найден, отправьте сообщение пользователю
		message := tgbotapi.NewMessage(int64(userID), "Ваш профиль не найден. Создайте анкету с помощью команды /start.")
		_, sendErr := b.bot.Send(message)
		if sendErr != nil {
			log.Printf("Ошибка отправки сообщения: %v", sendErr)
		}
		return err
	}
	// Отправите инлайн клавиатуру для выбора раздела анкеты

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Имя", "edit_name"),
			tgbotapi.NewInlineKeyboardButtonData("Фамилия", "edit_last_name"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Возраст", "edit_age"),
			tgbotapi.NewInlineKeyboardButtonData("Интересы", "edit_interests"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Завершить редактирование", "finish_editing"),
		),
	)
	message := tgbotapi.NewMessage(int64(userID), "Выберите раздел анкеты для редактирования:")
	message.ReplyMarkup = inlineKeyboard

	_, err = b.bot.Send(message)
	if err != nil {
		return err
	}

	// Ожидайте выбора пользователя
	for {
		userUpdate, ok := <-updates
		if !ok {
			// Канал закрыт, завершите выполнение
			return nil
		}

		if userUpdate.CallbackQuery != nil {
			callbackData := userUpdate.CallbackQuery.Data
			switch callbackData {
			case "edit_name":
				// Редактирование имени
				message := tgbotapi.NewMessage(int64(userID), "Редактирование: Пожалуйста, предоставьте новое имя:")
				_, err := b.bot.Send(message)
				if err != nil {
					return err
				}
				userUpdate, ok = <-updates
				if !ok || userUpdate.Message == nil || userUpdate.Message.Text == "" {
					continue
				}
				profile.FirstName = userUpdate.Message.Text

			case "edit_last_name":
				// Редактирование фамилии
				message := tgbotapi.NewMessage(int64(userID), "Редактирование: Пожалуйста, предоставьте новую фамилию:")
				_, err := b.bot.Send(message)
				if err != nil {
					return err
				}
				userUpdate, ok = <-updates
				if !ok || userUpdate.Message == nil || userUpdate.Message.Text == "" {
					continue
				}
				profile.LastName = userUpdate.Message.Text

			case "edit_age":
				// Редактирование возраста
				message := tgbotapi.NewMessage(int64(userID), "Редактирование: Пожалуйста, предоставьте новый возраст:")
				_, err := b.bot.Send(message)
				if err != nil {
					return err
				}
				userUpdate, ok = <-updates
				if !ok || userUpdate.Message == nil || userUpdate.Message.Text == "" {
					continue
				}
				age, err := strconv.Atoi(userUpdate.Message.Text)
				if err != nil {
					continue
				}
				profile.Age = age

			case "edit_interests":
				// Редактирование интересов
				message := tgbotapi.NewMessage(int64(userID), "Редактирование: Пожалуйста, предоставьте новые интересы:")
				_, err := b.bot.Send(message)
				if err != nil {
					return err
				}
				userUpdate, ok = <-updates
				if !ok || userUpdate.Message == nil || userUpdate.Message.Text == "" {
					continue
				}
				profile.Interests = userUpdate.Message.Text

			case "finish_editing":
				// Удаление старой анкеты из группы, если она существует
				if profile.MessageID != 0 {
					deleteMessage := tgbotapi.NewDeleteMessage(b.chatID, profile.MessageID)
					_, err = b.bot.DeleteMessage(deleteMessage)
					if err != nil {
						return err
					}
				}

				// Отправление обновленной анкеты в группу
				err = b.SendProfile(profile)
				if err != nil {
					return err
				}

				// Завершение редактирования
				message := tgbotapi.NewMessage(int64(userID), "Редактирование завершено.")
				_, err := b.bot.Send(message)
				if err != nil {
					return err
				}

				// Обновление профиля в хранилище
				err = b.dataStorage.SaveProfile(profile)
				if err != nil {
					return err
				}

				return nil
			}
		}
	}
}

// Удаление анкеты
func (b *MotoBot) DeleteProfile(userID int) error {
	// Получите профиль пользователя из хранилища
	profile, err := b.dataStorage.GetProfile(userID)
	if err != nil {
		// Если профиль не найден, отправьте сообщение пользователю
		message := tgbotapi.NewMessage(int64(userID), "Ваш профиль не найден. Создайте анкету с помощью команды /start.")
		_, sendErr := b.bot.Send(message)
		if sendErr != nil {
			log.Printf("Ошибка отправки сообщения: %v", sendErr)
		}
		return err
	}

	// Удалите профиль из хранилища
	err = b.dataStorage.DeleteProfile(userID)
	if err != nil {
		return err
	}

	// Удалите сообщение с анкетой из группы, используя MessageID
	if profile.MessageID != 0 {
		deleteMessage := tgbotapi.NewDeleteMessage(b.chatID, profile.MessageID)
		_, err = b.bot.DeleteMessage(deleteMessage)
		if err != nil {
			return err
		}
	}

	// Отправьте сообщение об успешном удалении
	message := tgbotapi.NewMessage(int64(userID), "Анкета успешно удалена.")
	_, err = b.bot.Send(message)
	if err != nil {
		return err
	}

	return nil
}

// Отправляет анкету пользователя в группу и сохраняет MessageID в хранилище
func (b *MotoBot) SendProfile(profile *user.Profile) error {
	// Преобразуйте анкету в JSON
	profileJSON, err := profile.ToJSON()
	if err != nil {
		return err
	}

	// Отправьте JSON в чат группы
	messageText := "А я всё ждал, когда же ты появишься:\n" + profileJSON
	message := tgbotapi.NewMessage(b.chatID, messageText)

	// Отправляем сообщение
	msg, err := b.bot.Send(message)
	if err != nil {
		return err
	}

	// Сохраняем MessageID в профиле анкеты
	profile.MessageID = msg.MessageID

	// Обновляем профиль в хранилище с новым MessageID
	err = b.dataStorage.SaveProfile(profile)
	if err != nil {
		return err
	}

	return nil
}

// GetProjectInfo отправляет информацию пользователю.
func (b *MotoBot) GetProjectInfo(chatID int64) error {
	// Ваш код для отправки информации
	informationMessage := "Ебэрис Гузеев представляет новый проект мото-покатушек и знакомств «Давай прокатимся». Мальчики катают девочек, девочки катают мальчиков… Все просто) Организовываем массовые покатушки с моим участием, в которых я буду в качестве ператора и свахи)."
	message := tgbotapi.NewMessage(chatID, informationMessage)
	_, err := b.bot.Send(message)
	return err
}

// welcomeNewUser отправляет приветственное сообщение в личку пользователю и, если невозможно, то приветствует его в группе без инлайн клавиатуры.
func (b *MotoBot) welcomeNewUser(userID int, chatID int64) error {
	// Получение информации о пользователе по userID
	chatConfig := tgbotapi.ChatConfigWithUser{
		ChatID: chatID, // ID чата, в котором вы хотите проверить членство пользователя
		UserID: userID, // ID пользователя, членство которого вы хотите проверить
	}

	user, err := b.bot.GetChatMember(chatConfig)
	if err != nil {
		// Обработка ошибки
		return err
	}

	// Извлеките username из полученной информации, если он доступен
	username := user.User.UserName

	// Приветственное сообщение
	welcomeMessage := fmt.Sprintf("Добро пожаловать, @%s! Чем я могу вам помочь?", username)
	welcomeMessageGroupe := fmt.Sprintf("Добро пожаловать, @%s! Для создания анкеты и получения информации - напиши мне!", username)

	// Создание инлайн клавиатуры
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Информация", "/info"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Создание анкеты", "/start"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Редактирование анкеты", "/edit"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удаление анкеты", "/delete"),
		),
	)

	// Попытка отправить сообщение в личку
	message := tgbotapi.NewMessage(int64(userID), welcomeMessage)
	message.ReplyMarkup = inlineKeyboard
	_, err = b.bot.Send(message)
	if err != nil {
		// Если не удалось отправить в личку, отправляем только приветствие в группу
		message = tgbotapi.NewMessage(chatID, welcomeMessageGroupe)
		_, err = b.bot.Send(message)
		if err != nil {
			log.Printf("Ошибка при отправке приветственного сообщения: %v", err)
		}
	}

	return nil
}

// sendUnknownCommandMessage отправляет сообщение о неизвестной команде и инлайн клавиатуру приветствия.
func (b *MotoBot) sendUnknownCommandMessage(chatID int64) error {
	// Приветственное сообщение
	welcomeMessage := "Ваша команда не опознана, выберите что вы хотите сделать:"

	// Создание инлайн клавиатуры
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Информация", "/info"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Создание анкеты", "/start"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Редактирование анкеты", "/edit"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удаление анкеты", "/delete"),
		),
	)

	// Отправка сообщения с инлайн клавиатурой
	message := tgbotapi.NewMessage(chatID, welcomeMessage)
	message.ReplyMarkup = inlineKeyboard

	_, err := b.bot.Send(message)
	return err
}

// DownloadFile загружает файл по его пути и возвращает []byte с содержимым файла.
func (b *MotoBot) DownloadPhoto(filePath string) ([]byte, error) {
	fileURL := "https://api.telegram.org/file/bot" + b.token + "/" + filePath
	response, err := http.Get(fileURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP статус: %s", response.Status)
	}

	fileBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}
