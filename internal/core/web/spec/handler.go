package spec

type HandleFunc func()

type Handler interface {
	Routes(r *Router)
}
