package storage

import "github.com/t1ery/MotoBot/internal/user"

type Storage interface {
	SaveProfile(profile *user.Profile) error      // Сохраняет информацию о пользователе в БД
	GetProfile(userID int) (*user.Profile, error) // Получает информацию о пользователе
	DeleteProfile(userID int) error               // Удаляет профиль из БД
}
