package user

// Profile - структура для анкеты пользователя
type Profile struct {
	UserID    int    // Идентификатор пользователя
	FirstName string // Имя
	LastName  string // Фамилия
	Age       int    // Возраст
	Interests string // Интересы
	Photo     []byte // Фотография пользователя
	Contacts  string // Контактная информация пользователя
	IsDriver  bool   // Является ли пользователь водителем
	MessageID int    // Номер сообщения размещения анкеты в группе
}

// Step constants - шаги создания анкеты
const (
	StepFirstName = iota
	StepLastName
	StepAge
	StepIsDriver
	StepInterests
	StepPhoto
	StepContacts
	StepCompleted // Завершено создание анкеты
)
