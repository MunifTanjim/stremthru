package stremio_userdata

type UserData[T any] interface {
	HasRequiredValues() bool

	GetEncoded() string
	SetEncoded(encoded string)
	Ptr() *T
}
