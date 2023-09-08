package y_middleware

import (
	"net/http"
	"os"
)

const (
	DefaultAddress = ":8080"
)

// Handler is an interface that objects can implement to be registered to serve as middleware
// in the middleware stack.
// ServeHTTP should yield to the next middleware in the chain by invoking the next http.HandlerFunc
// passed in
// If the Handler write to the ResponseWriter, the next http.HandlerFunc should not be invoked.
type Handler interface {
	ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)
}

// HandlerFunc is an adapter to allow the use of ordinary functions as Handler
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler object that calls f.

type HandlerFunc func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)

func (h HandlerFunc) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	h(rw, r, next)
}

type middleware struct {
	handler Handler
	next    *middleware
}

func (m middleware) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	m.handler.ServeHTTP(rw, r, m.next.ServeHTTP)
}

// Wrap converts a http.Handler into a yusuf.Handler so it can be used as a yusuf
// middleware. The next http.HandlerFunc is automatically called after the Handler
// is executed.

func Wrap(handler http.Handler) Handler {
	return HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		handler.ServeHTTP(rw, r)
		next(rw, r)
	})
}

// WrapFunc converts a http.HandlerFunc into a negroni.Handler so it can be used as a Negroni
// middleware. The next http.HandlerFunc is automatically called after the Handler
// is executed.
func WrapFunc(handlerFunc http.HandlerFunc) Handler {
	return HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		handlerFunc(rw, r)
		next(rw, r)
	})
}

// Kudret is a stack of Middleware Handlers that can be invoked as a http.Handler.
// Kudret middleware is evaluated in the order that they are added to the stack using
// the Use and UseHandler methods.
type Kudret struct {
	middleware middleware
	handlers   []Handler
}

// New returns a new Kudret instance with no middleware preconfigured
func New(handlers ...Handler) *Kudret {
	return &Kudret{
		handlers:   handlers,
		middleware: build(handlers),
	}
}

// With returns a new Kudret instance that is combination of the kudret
// receiver's handlers and the provided handlers
func (k *Kudret) With(handlers ...Handler) *Kudret {
	return New(
		append(k.handlers, handlers...)...,
	)
}

func (k *Kudret) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	k.middleware.ServeHTTP(rw, r)
}

// Use adds a Handler onto the middleware stack. Handlers are invoked in the order they are added to a Negroni.
func (k *Kudret) Use(handler Handler) {
	if handler == nil {
		panic("handler cannot be nil")
	}

	k.handlers = append(k.handlers, handler)
	k.middleware = build(k.handlers)
}

// UseFunc add a Kudret-style handler function onto the middleware stack.
func (k *Kudret) UseFunc(handlerFunc func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)) {
	k.Use(HandlerFunc(handlerFunc))
}

// UseHandler adds a http.Handler onto the middleware stack. Handlers are invoked in the order they are added to a Kudret.
func (k *Kudret) UseHandler(handler http.Handler) {
	k.Use(Wrap(handler))
}

// UseHandlerFunc adds a http.HandlerFunc-style handler function onto the middleware stack.
func (k *Kudret) UseHandlerFunc(handlerFunc func(rw http.ResponseWriter, r *http.Request)) {
	k.Use(WrapFunc(handlerFunc)) // todo: try if this works
	// k.UseHandler(http.HandlerFunc(handlerFunc)) // this one works
}

func (k *Kudret) Run(addr ...string) {

}

func detectAddress(addr ...string) string {
	if len(addr) > 0 {
		return addr[0]
	}
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return DefaultAddress
}

func (k *Kudret) Handlers() []Handler {
	return k.handlers
}

func build(handlers []Handler) middleware {
	var next middleware

	if len(handlers) == 0 {
		return voidMiddleware()
	} else if len(handlers) > 1 {
		next = build(handlers[1:])
	} else {
		next = voidMiddleware()
	}
	return middleware{handlers[0], &next}
}

func voidMiddleware() middleware {
	return middleware{
		HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {}),
		&middleware{},
	}
}
