package main

import "net/http"

type Server struct{}

func (s *Server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, world!"))
}

func (s *Server) PingHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello Pong!!"))
}
