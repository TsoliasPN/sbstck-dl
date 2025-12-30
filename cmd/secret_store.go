package cmd

import (
	"errors"
	"fmt"
	"sync"

	"github.com/zalando/go-keyring"
)

const secretServiceName = "sbstck-dl"

var errSecretNotFound = errors.New("secret not found")

type secretStoreInterface interface {
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
}

type keyringStore struct {
	service string
}

func newKeyringStore() secretStoreInterface {
	return keyringStore{service: secretServiceName}
}

func (k keyringStore) Set(key, value string) error {
	return keyring.Set(k.service, key, value)
}

func (k keyringStore) Get(key string) (string, error) {
	value, err := keyring.Get(k.service, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", errSecretNotFound
	}
	return value, err
}

func (k keyringStore) Delete(key string) error {
	err := keyring.Delete(k.service, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return errSecretNotFound
	}
	return err
}

type memorySecretStore struct {
	mu    sync.Mutex
	items map[string]string
}

func newMemorySecretStore() *memorySecretStore {
	return &memorySecretStore{
		items: make(map[string]string),
	}
}

func (m *memorySecretStore) Set(key, value string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key] = value
	return nil
}

func (m *memorySecretStore) Get(key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.items[key]
	if !ok {
		return "", errSecretNotFound
	}
	return value, nil
}

func (m *memorySecretStore) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[key]; !ok {
		return errSecretNotFound
	}
	delete(m.items, key)
	return nil
}

var secretStore secretStoreInterface = newKeyringStore()

func setSecretStoreForTest(store secretStoreInterface) func() {
	previous := secretStore
	secretStore = store
	return func() {
		secretStore = previous
	}
}
