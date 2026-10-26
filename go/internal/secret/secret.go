package secret

import (
	"errors"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	defaultService = "app.smartpaste.mac.typesafe"
	account        = "direct-jev-api-key"
	envKey         = "SMARTPASTE_DIRECT_JEV_KEY"
)

var ErrUnavailable = errors.New("system credential store is unavailable")

type Store struct {
	service string
}

func New() *Store {
	service := os.Getenv("SMARTPASTE_KEYCHAIN_SERVICE")
	if service == "" {
		service = defaultService
	}
	return &Store{service: service}
}

func (s *Store) Get() (string, error) {
	if value := strings.TrimSpace(os.Getenv(envKey)); value != "" {
		return value, nil
	}
	value, err := keyring.Get(s.service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func (s *Store) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("empty secret")
	}
	if err := keyring.Set(s.service, account, value); err != nil {
		return errors.Join(ErrUnavailable, err)
	}
	return nil
}

func (s *Store) Delete() error {
	err := keyring.Delete(s.service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

func (s *Store) FromEnvironment() bool {
	return strings.TrimSpace(os.Getenv(envKey)) != ""
}
