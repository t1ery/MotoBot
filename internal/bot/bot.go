package bot

import (
	"fmt"
	"github.com/t1ery/MotoBot/config"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/t1ery/MotoBot/internal/storage"
	"github.com/t1ery/MotoBot/internal/user"
)

// Bot представляет интерфейс для взаимодействия с ботом.
type Bot interface {
	CreateProfile(userID int, chatID int64, updates <-chan tgbotapi.Update) error // Создание анкеты
	EditProfile(userID int, chatID int64, updates <-chan tgbotapi.Update) error   // Редактирование анкеты
	DeleteProfile(userID int) error                                               // Удаление анкеты
	SendProfile(userID int, chatID int64, profile *user.Profile) error            // Отправка анкеты в соответствующую тему
	GetProjectInfo(chatID int64) error                                            // Предоставление информации о проекте пользователю
	Run()                                                                         // Запуск бота
}

// MotoBot представляет реализацию интерфейса Bot.
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

	// Здесь мы запрашиваем ChatID из файла конфигурации
	configValues, err := config.GetConfigValuesFromConfig("ChatID")
	if err != nil {
		log.Panic(err)
	}

	chatID, ok := configValues["ChatID"].(int64)
	if !ok {
		log.Panic("ChatID не является корректным int64 значением")
	}

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
					err := b.CreateProfile(update.Message.From.ID, chatID, updates)
					if err != nil {
						log.Printf("Ошибка при создании анкеты: %v", err)
					}
				case "edit":
					// Обработка команды "/edit"
					err := b.EditProfile(update.Message.From.ID, chatID, updates)
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
				err := b.CreateProfile(update.CallbackQuery.From.ID, chatID, updates)
				if err != nil {
					log.Printf("Ошибка при создании анкеты: %v", err)
				}
			case "/edit":
				// Обработка команды "Редактирование анкеты"
				err := b.EditProfile(update.CallbackQuery.From.ID, chatID, updates)
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

func (b *MotoBot) CreateProfile(userID int, chatID int64, updates <-chan tgbotapi.Update) error {
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
				message = tgbotapi.NewMessage(int64(userID), "Шаг 1: Введите ваше имя:")
			case user.StepLastName:
				message = tgbotapi.NewMessage(int64(userID), "Шаг 2: Введите вашу фамилию:")
			case user.StepAge:
				message = tgbotapi.NewMessage(int64(userID), "Шаг 3: Введите ваш возраст:")
			case user.StepIsDriver:
				inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Да", "is_driver_yes"),
						tgbotapi.NewInlineKeyboardButtonData("Нет", "is_driver_no"),
					),
				)
				message = tgbotapi.NewMessage(int64(userID), "Шаг 4: Вы являетесь водителем?")
				message.ReplyMarkup = inlineKeyboard
			case user.StepInterests:
				var messageText string
				if profile.IsDriver {
					messageText = "Шаг 5: Какие у вас будут пожелания к пассажиру?"
				} else {
					messageText = "Шаг 5: Какие у вас будут пожелания к водителю?"
				}
				message = tgbotapi.NewMessage(int64(userID), messageText)
			case user.StepPhoto:
				message = tgbotapi.NewMessage(int64(userID), "Шаг 6: Загрузите фотографию на ваш выбор:")
			case user.StepContacts:
				message = tgbotapi.NewMessage(int64(userID), "Шаг 7: Укажите по желанию контакты для связи с вами, например - номер телефона:")
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
			if userUpdate.Message != nil && (userUpdate.Message.Text != "" || len(*userUpdate.Message.Photo) > 0) {
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
					creationState = user.StepIsDriver
				case user.StepIsDriver:
					if userUpdate.Message.Text == "Да" {
						profile.IsDriver = true
					} else {
						profile.IsDriver = false
					}
					creationState = user.StepInterests
				case user.StepInterests:
					profile.Interests = userUpdate.Message.Text
					creationState = user.StepPhoto
				case user.StepPhoto:
					// Проверяем, есть ли фотографии в сообщении
					if userUpdate.Message != nil {
						if userUpdate.Message.Photo != nil && len(*userUpdate.Message.Photo) > 0 {
							// Выбираем самую большую по размеру фотографию из всех отправленных
							largestPhoto := (*userUpdate.Message.Photo)[0]
							for _, photo := range *userUpdate.Message.Photo {
								if photo.FileSize > largestPhoto.FileSize {
									largestPhoto = photo
								}
							}

							// Получаем информацию о файле фотографии
							fileConfig := tgbotapi.FileConfig{FileID: largestPhoto.FileID}
							photoFile, err := b.bot.GetFile(fileConfig)
							if err != nil {
								log.Printf("Ошибка при получении файла фотографии: %v", err)
								// Обработка ошибки - возможно, стоит уведомить пользователя
								// В данном случае, можно просто отправить сообщение, что не удалось получить фотографию.
							} else {
								// Загружаем фотографию
								photoBytes, err := b.downloadPhoto(photoFile.FilePath)
								if err != nil {
									log.Printf("Ошибка при загрузке файла фотографии: %v", err)
								} else {
									// Добавляем фотографию в структуру пользователя
									profile.Photo = photoBytes

									// Выводим информацию о фотографии в лог
									log.Printf("Сохранена фотография размером %d байт", len(photoBytes))

									// Переходим к следующему шагу
									creationState = user.StepContacts
								}
							}
						} else {
							log.Println("Нет фотографий в сообщении")
						}
					} else {
						// Если нет фотографий в сообщении, отправляем сообщение пользователю
						message := tgbotapi.NewMessage(int64(userID), "На данном шаге необходимо загрузить фотографию.")
						_, err := b.bot.Send(message)
						if err != nil {
							log.Printf("Ошибка при отправке сообщения: %v", err)
							// Обработка ошибки
						}
					}
				case user.StepContacts:
					profile.Contacts = userUpdate.Message.Text
					creationState = user.StepCompleted
				}
			} else if userUpdate.CallbackQuery != nil {
				if userUpdate.CallbackQuery.Data == "is_driver_yes" {
					profile.IsDriver = true
				} else if userUpdate.CallbackQuery.Data == "is_driver_no" {
					profile.IsDriver = false
				}
				creationState = user.StepInterests
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
		err := b.SendProfile(userID, chatID, profile)
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
func (b *MotoBot) EditProfile(userID int, chatID int64, updates <-chan tgbotapi.Update) error {
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
				message := tgbotapi.NewMessage(int64(userID), "Редактирование: Введите новое имя:")
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
				message := tgbotapi.NewMessage(int64(userID), "Редактирование: Введите новую фамилию:")
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
				message := tgbotapi.NewMessage(int64(userID), "Редактирование: Введите новый возраст:")
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
				message := tgbotapi.NewMessage(int64(userID), "Редактирование: Напишите о своих новых интересах и увлечениях:")
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
				err = b.SendProfile(userID, chatID, profile)
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
func (b *MotoBot) SendProfile(userID int, chatID int64, profile *user.Profile) error {
	// Извлеките username из полученной информации, если он доступен
	username, err := b.getUsername(userID, chatID)

	// Подготовьте текст анкеты
	messageText := "Анкета пользователя: " + "@" + username + "\n"
	messageText += "Имя: " + profile.FirstName + "\n"
	messageText += "Фамилия: " + profile.LastName + "\n"
	messageText += "Возраст: " + strconv.Itoa(profile.Age) + "\n"
	messageText += "Интересы: " + profile.Interests + "\n"
	messageText += "Водитель: "
	if profile.IsDriver {
		messageText += "🏍️\n"
	} else {
		messageText += "🚶\n"
	}
	messageText += "Контакты: " + profile.Contacts + "\n"

	// Отправьте сообщение с фотографией и текстом
	msg := tgbotapi.NewPhotoUpload(b.chatID, tgbotapi.FileBytes{
		Bytes: profile.Photo,
	})
	msg.Caption = messageText

	sentMsg, err := b.bot.Send(msg)
	if err != nil {
		return err
	}

	// Сохраните MessageID в профиле анкеты
	profile.MessageID = sentMsg.MessageID

	// Обновите профиль в хранилище с новым MessageID
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

	username, err := b.getUsername(userID, chatID)

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
func (b *MotoBot) downloadPhoto(filePath string) ([]byte, error) {
	fileURL := "https://api.telegram.org/file/bot" + b.token + "/" + filePath
	response, err := http.Get(fileURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP статус: %s", response.Status)
	}

	fileBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}

// Функция getUsername возвращает имя пользователя
func (b *MotoBot) getUsername(userID int, chatID int64) (string, error) {
	// Получение информации о пользователе по userID
	chatConfig := tgbotapi.ChatConfigWithUser{
		ChatID: chatID, // ID чата, в котором вы хотите проверить членство пользователя
		UserID: userID, // ID пользователя, членство которого вы хотите проверить
	}

	user, err := b.bot.GetChatMember(chatConfig)
	if err != nil {
		// Обработка ошибки
		return "", err
	}

	return user.User.UserName, nil
}
