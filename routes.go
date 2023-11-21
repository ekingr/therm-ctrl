package main

import (
	"mime"
	"net/http"
)

//
// Logging middleware

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
func (s *server) logMid(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.logger.Println(">", r.RemoteAddr, r.Method, r.URL, ">")
		lrw := NewLoggingResponseWriter(w)
		handler.ServeHTTP(lrw, r)
		s.logger.Println("<", r.RemoteAddr, r.Method, r.URL, "<", lrw.statusCode)
	})
}

//
// Routes

func (s *server) routes() {
	// hdlTherm.go
	s.router.HandleFunc(s.domain+"/status.json", s.method(http.MethodGet,
		s.getStatus()))
	s.router.HandleFunc(s.domain+"/set", s.method(http.MethodPost,
		s.inJson(s.postSet())))
}

//
// Method enforcement middleware

func (s *server) method(method string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			s.logger.Println("Error:", r.URL.Path, "wrong method", r.Method, "expected", method)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}

//
// Json-header checking middleware
func (s *server) inJson(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctype, ok := r.Header["Content-Type"]
		if !ok || len(ctype) < 1 {
			s.logger.Println("Invalid request no content-type")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		if mimet, _, err := mime.ParseMediaType(ctype[0]); err != nil || mimet != "application/json" {
			if err != nil {
				s.logger.Println("Error parsing content type: ", err)
			} else {
				s.logger.Println("Invalid request content-type: ", mimet)
			}
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		h(w, r)
	}
}

//
// Utils

// Cap a string length

const capLen = 100

func capStr(str string) string {
	if len(str) < capLen {
		return str
	} else {
		return str[:(capLen - 1)]
	}
}
