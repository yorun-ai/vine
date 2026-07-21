package conf

type Lifecycle string

const (
	LifecycleEternal Lifecycle = "ETERNAL"
	LifecycleInstant Lifecycle = "INSTANT"
)

type Config interface {
	mustBeConfig()
}

type ConfigModel struct{}

func (ConfigModel) mustBeConfig() {}

func ParseLifecycle(s string) (Lifecycle, bool) {
	lifecycle := Lifecycle(s)
	return lifecycle, lifecycle.IsValid()
}

func (lifecycle Lifecycle) String() string {
	return string(lifecycle)
}

func (lifecycle Lifecycle) IsValid() bool {
	switch lifecycle {
	case LifecycleEternal, LifecycleInstant:
		return true
	default:
		return false
	}
}
