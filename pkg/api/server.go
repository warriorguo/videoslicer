package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/warriorguo/videoslicer/internal/database"
)

type Server struct {
	handler    *Handler
	server     *http.Server
	httpHandler http.Handler
}

type Config struct {
	Port        string
	TaskDir     string
	MaxFileSize int64
}

func NewServer(db *database.DB, config Config) *Server {
	handler := NewHandler(db, config.TaskDir, config.MaxFileSize)
	
	r := mux.NewRouter()
	
	// API routes
	api := r.PathPrefix("/v1").Subrouter()
	api.HandleFunc("/tasks", handler.CreateTask).Methods("POST")
	api.HandleFunc("/tasks/{task_id}", handler.GetTask).Methods("GET")
	api.HandleFunc("/tasks/{task_id}/result", handler.DownloadResult).Methods("GET")
	
	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")
	
	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	
	// Logging middleware
	loggedRouter := loggingMiddleware(c.Handler(r))
	
	server := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      loggedRouter,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	
	return &Server{
		handler:     handler,
		server:      server,
		httpHandler: loggedRouter,
	}
}

func (s *Server) Start() error {
	log.Printf("Starting server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return s.server.Shutdown(ctx)
}

func (s *Server) Handler() http.Handler {
	return s.httpHandler
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap the ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}