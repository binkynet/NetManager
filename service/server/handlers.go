package server

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (s *server) notFound(w http.ResponseWriter, r *http.Request) {
	s.requestLog.Warn().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Msg("path not found")
	http.NotFound(w, r)
}

func (s *server) handleGetWorkerConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := context.Background()
	workerID := ps.ByName("id")
	result, err := s.api.GetWorkerConfig(ctx, workerID)
	if err != nil {
		handleError(w, err)
	} else {
		sendJSON(w, http.StatusOK, result)
	}
}

func (s *server) handleGetWorkers(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := context.Background()
	result, err := s.api.GetWorkers(ctx)
	if err != nil {
		handleError(w, err)
	} else {
		sendJSON(w, http.StatusOK, result)
	}
}
