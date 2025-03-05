package server

import (
	"errors"

	"github.com/cockroachdb/pebble"
)

type Store struct {
	db *pebble.DB
}

var ErrNotFound = errors.New("not found")

func NewStore(path string) (*Store, error) {

	opts := &pebble.Options{}
	db, err := pebble.Open(path, opts)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Get(key []byte) ([]byte, error) {
	value, closer, err := s.db.Get(key)

	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	v := append([]byte(nil), value...)
	return v, nil
}

func (s *Store) Set(key, value []byte) error {
	return s.db.Set(key, value, pebble.Sync)
}
