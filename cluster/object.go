// Generic object interface
package cluster

import "github.com/reusee/mmh3"

type Object interface {
	Bytes() []byte
	Hash() []byte
}

type StringObject struct {
	Data *string
}

func (s StringObject) Bytes() []byte {
	return []byte(*s.Data)
}

func (s StringObject) Hash() []byte {
	return mmh3.Sum128([]byte(*s.Data))
}
