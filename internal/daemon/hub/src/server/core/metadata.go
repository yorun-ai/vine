package core

type MetadataRepo interface {
	IsSeeded() bool
	MarkSeeded()
}
