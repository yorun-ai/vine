package redis

const (
	RevisionKey     = "meta:revision"
	EventKindUpsert = "upsert"
	EventKindDelete = "delete"
)

type Event struct {
	Revision uint64 `json:"revision"`
	Kind     string `json:"kind"`
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
}

type NotifyOperation struct {
	Key    string
	Value  string
	Delete bool
}
