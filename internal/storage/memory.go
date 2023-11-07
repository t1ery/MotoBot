package storage

import (
	"errors"
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
	return nil
}

func (s *MemoryStorage) GetProfile(userID int) (*user.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, found := s.data[userID]
	if !found {
		return nil, errors.New("profile not found")
	}
	return profile, nil
}

func (s *MemoryStorage) DeleteProfile(userID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, userID)
	return nil
}
