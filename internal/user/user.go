package user

import "encoding/json"

// Profile - структура для анкеты пользователя
type Profile struct {
	UserID    int      // Идентификатор пользователя
	FirstName string   // Имя
	LastName  string   // Фамилия
	Age       int      // Возраст
	Interests string   // Интересы
	Photos    [][]byte // Фотографии пользователя
	IsDriver  bool     // Является ли пользователь водителем
	MessageID int      // Номер сообщения размещения анкеты в группе
}

// Step constants - шаги создания анкеты
const (
	StepFirstName = iota
	StepLastName
	StepAge
	StepInterests
	StepIsDriver
	StepPhoto
	StepCompleted // Завершено создание анкеты
)

// ToJSON преобразует профиль пользователя в JSON формат
func (p *Profile) ToJSON() (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
