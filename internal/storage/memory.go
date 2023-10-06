package storage

import (
	"errors"
	"fmt"
	"sync"

	"github.com/t1ery/MotoBot/internal/user"
)

type MemoryStorage struct {
	data map[int]*user.Profile
	mu   sync.Mutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[int]*user.Profile),
	}
}

func (s *MemoryStorage) SaveProfile(profile *user.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[profile.UserID] = profile
	fmt.Printf("Профиль сохранен в хранилище: %+v\n", profile)
	return nil
}

func (s *MemoryStorage) GetProfile(userID int) (*user.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, found := s.data[userID]
	if !found {
		return nil, errors.New("profile not found")
	}

	// Вывод информации о профиле в лог
	fmt.Printf("Получен профиль для пользователя %d: %+v\n", userID, profile)

	return profile, nil
}

func (s *MemoryStorage) DeleteProfile(userID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, userID)
	return nil
}

// Другие методы реализации интерфейса DataStorage
