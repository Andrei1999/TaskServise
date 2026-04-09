package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	templatedomain "example.com/taskservice/internal/domain/template"
	templateusecase "example.com/taskservice/internal/usecase/template"
)

type TemplateHandler struct {
	usecase templateusecase.Usecase
}

func NewTemplateHandler(usecase templateusecase.Usecase) *TemplateHandler {
	return &TemplateHandler{usecase: usecase}
}

func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTemplateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input := templateusecase.CreateInput{
		Title:         req.Title,
		Description:   req.Description,
		RuleType:      req.RuleType,
		RuleParams:    req.RuleParams,
		ExecutionTime: req.ExecutionTime,
	}
	tmpl, err := h.usecase.Create(r.Context(), input)
	if err != nil {
		writeTemplateError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, newTemplateResponse(tmpl))
}

func (h *TemplateHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := getTemplateID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	tmpl, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeTemplateError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newTemplateResponse(tmpl))
}

func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getTemplateID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req updateTemplateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input := templateusecase.UpdateInput{
		Title:         req.Title,
		Description:   req.Description,
		RuleType:      req.RuleType,
		RuleParams:    req.RuleParams,
		ExecutionTime: req.ExecutionTime,
	}
	tmpl, err := h.usecase.Update(r.Context(), id, input)
	if err != nil {
		writeTemplateError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newTemplateResponse(tmpl))
}

func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := getTemplateID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.usecase.Delete(r.Context(), id); err != nil {
		writeTemplateError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	templates, err := h.usecase.List(r.Context())
	if err != nil {
		writeTemplateError(w, err)
		return
	}
	resp := make([]templateResponse, len(templates))
	for i, tmpl := range templates {
		resp[i] = newTemplateResponse(&tmpl)
	}
	writeJSON(w, http.StatusOK, resp)
}

func getTemplateID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		return 0, errors.New("missing template id")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid template id")
	}
	return id, nil
}

func writeTemplateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, templatedomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, templatedomain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}