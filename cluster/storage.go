package cluster

type Storage interface {
    Get([]byte) ([]byte, bool)
    Put([]byte, []byte)
}

type InMemoryStorage struct {
    data map[string][]byte
}

func NewInMemoryStorage() Storage {
    return &InMemoryStorage{
        data: make(map[string][]byte, 10),
    }
}

func (m *InMemoryStorage) Get(key []byte) ([]byte, bool) {
    v, ok := m.data[string(key)]
    return v, ok
}

func (m *InMemoryStorage) Put(key, value []byte) {
    m.data[string(key)] = value
}
