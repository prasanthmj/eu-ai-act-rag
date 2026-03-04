package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prasanthmj/eu-ai-act-rag/pipeline"
)

// NewRouter creates an HTTP router with all API endpoints.
func NewRouter(p *pipeline.Pipeline) http.Handler {
	r := chi.NewRouter()

	r.Post("/api/classify", classifyHandler(p))
	r.Post("/api/obligations", obligationsHandler(p))
	r.Post("/api/prohibited", prohibitedHandler(p))
	r.Get("/api/article/{reference}", articleHandler(p))
	r.Post("/api/checklist", checklistHandler(p))

	return r
}

func classifyHandler(p *pipeline.Pipeline) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Description string `json:"description"`
			DomainHint  string `json:"domain_hint"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.Description == "" {
			writeError(w, http.StatusBadRequest, "description is required")
			return
		}

		result, err := p.ClassifySystem(r.Context(), req.Description, req.DomainHint)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, result)
	}
}

func obligationsHandler(p *pipeline.Pipeline) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RiskTier string `json:"risk_tier"`
			Domain   string `json:"domain"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.RiskTier == "" {
			writeError(w, http.StatusBadRequest, "risk_tier is required")
			return
		}

		result, err := p.GetObligations(r.Context(), req.RiskTier, req.Domain)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, result)
	}
}

func prohibitedHandler(p *pipeline.Pipeline) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.Description == "" {
			writeError(w, http.StatusBadRequest, "description is required")
			return
		}

		result, err := p.CheckProhibited(r.Context(), req.Description)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, map[string]string{"result": result})
	}
}

func articleHandler(p *pipeline.Pipeline) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reference := chi.URLParam(r, "reference")
		if reference == "" {
			writeError(w, http.StatusBadRequest, "reference is required")
			return
		}

		result, err := p.LookupArticle(r.Context(), reference)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, result)
	}
}

func checklistHandler(p *pipeline.Pipeline) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Description  string `json:"description"`
			OutputFormat string `json:"output_format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.Description == "" {
			writeError(w, http.StatusBadRequest, "description is required")
			return
		}

		result, err := p.RunFull(r.Context(), req.Description, "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if req.OutputFormat == "json" {
			writeJSON(w, result)
		} else {
			writeJSON(w, map[string]string{"checklist": result.Checklist})
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
