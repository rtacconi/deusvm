package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/riccardotacconi/deusvm/internal/config"
	"github.com/riccardotacconi/deusvm/internal/kvm"
	"github.com/riccardotacconi/deusvm/internal/storage"
	"go.uber.org/zap"
)

type Server struct {
	logger  *zap.Logger
	manager kvm.Manager
	cfg     config.Config
	router  *chi.Mux
	store   storage.Manager
}

func NewServer(logger *zap.Logger, manager kvm.Manager, store storage.Manager, cfg config.Config) *Server {
	s := &Server{logger: logger, manager: manager, store: store, cfg: cfg}
	s.router = chi.NewRouter()
	s.router.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	if cfg.API.AuthToken != "" {
		s.router.Use(s.authMiddleware(cfg.API.AuthToken))
	}
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Route("/vms", func(r chi.Router) {
			r.Post("/", s.createVM)
			r.Get("/", s.listVMs)
			r.Get("/{id}", s.getVM)
			r.Put("/{id}/start", s.startVM)
			r.Put("/{id}/stop", s.stopVM)
			r.Delete("/{id}", s.deleteVM)
		})

		r.Route("/images", func(r chi.Router) {
			r.Post("/", s.createImage)
			r.Get("/", s.listImages)
			r.Delete("/{name}", s.deleteImage)
		})
	})
	return s
}

func (s *Server) Router() http.Handler { return s.router }

type createVMRequest struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	CPU    int    `json:"cpu"`
	Memory string `json:"memory"` // human string like 4GB
	Disk   string `json:"disk"`   // human string like 20GB
}

type vmResponse struct{ kvm.VM }

func (s *Server) createVM(w http.ResponseWriter, r *http.Request) {
	var req createVMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	mem, err := parseSize(req.Memory)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid memory")
		return
	}
	disk, err := parseSize(req.Disk)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid disk")
		return
	}
	vm, err := s.manager.CreateVM(r.Context(), kvm.CreateVMRequest{
		Name: req.Name, CPU: req.CPU, MemoryBytes: mem, DiskBytes: disk, Image: req.Image,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, vmResponse{vm})
}

func (s *Server) listVMs(w http.ResponseWriter, r *http.Request) {
	vms, err := s.manager.ListVMs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, vms)
}

func (s *Server) getVM(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	vm, err := s.manager.GetVM(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, vmResponse{vm})
}

func (s *Server) startVM(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.manager.StartVM(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (s *Server) stopVM(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.manager.StopVM(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Server) deleteVM(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.manager.DeleteVM(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) authMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			h := r.Header.Get("Authorization")
			const prefix = "Bearer "
			if len(h) > len(prefix) && h[:len(prefix)] == prefix && h[len(prefix):] == token {
				next.ServeHTTP(w, r)
				return
			}
			writeError(w, http.StatusUnauthorized, "unauthorized")
		})
	}
}

func parseSize(s string) (int64, error) {
	// minimal parser: expect numbers ending with GB or MB
	if len(s) < 3 {
		return 0, errSize()
	}
	unit := s[len(s)-2:]
	numStr := s[:len(s)-2]
	var v int64
	for i := 0; i < len(numStr); i++ {
		if numStr[i] < '0' || numStr[i] > '9' {
			return 0, errSize()
		}
		v = v*10 + int64(numStr[i]-'0')
	}
	switch unit {
	case "GB":
		return v * 1024 * 1024 * 1024, nil
	case "MB":
		return v * 1024 * 1024, nil
	default:
		return 0, errSize()
	}
}

func errSize() error {
	return &sizeErr{}
}

type sizeErr struct{}

func (e *sizeErr) Error() string { return "invalid size" }

// Images

type createImageRequest struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

func (s *Server) createImage(w http.ResponseWriter, r *http.Request) {
	var req createImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Name == "" || req.Source == "" {
		writeError(w, http.StatusBadRequest, "name and source required")
		return
	}
	img, err := s.store.SaveImageFromURL(r.Context(), req.Name, req.Source)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, img)
}

func (s *Server) listImages(w http.ResponseWriter, r *http.Request) {
	imgs, err := s.store.ListImages(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, imgs)
}

func (s *Server) deleteImage(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	if err := s.store.DeleteImage(r.Context(), name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
