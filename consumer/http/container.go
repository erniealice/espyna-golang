package http

import (
	"net/http"
)

// Container is the built, ready-to-serve product of the fluent Server.Build /
// Server.MustBuild (A.1 #8). It owns the assembled http.Handler and the listen
// address; the consumer app stores it directly:
//
//	c := consumer.NewServer().WithApp(cfg).WithMiddleware(middleware.StandardAdmin()).
//	    WithBlocks(...).MustBuild()
//	c.Serve()
type Container struct {
	handler http.Handler
	addr    string
}

// Handler returns the fully assembled fixed-order middleware chain over the mux.
func (c *Container) Handler() http.Handler { return c.handler }

// Addr returns the server listen address (e.g. ":8080").
func (c *Container) Addr() string { return c.addr }

// Serve listens on Addr and serves the assembled handler. Blocks until the
// server stops; returns the http.ListenAndServe error.
func (c *Container) Serve() error {
	return http.ListenAndServe(c.addr, c.handler)
}
