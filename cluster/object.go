// Generic object interface
package cluster

type Object interface {
	Bytes() []byte
	Hash() []byte
}