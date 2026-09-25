package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/dev-labs-ai/tasks/apps/api/internal/services"
)

// TaskService is the part of services.TaskService the handlers use.
type TaskService interface {
	List(ctx context.Context, userID int64, status string) ([]services.Task, error)
	Create(ctx context.Context, userID int64, title string) (services.Task, error)
	Update(ctx context.Context, userID, taskID int64, u services.TaskUpdate) (services.Task, error)
	Delete(ctx context.Context, userID, taskID int64) error
}

type taskResponse struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newTaskResponse(t services.Task) taskResponse {
	return taskResponse{
		ID:        t.ID,
		Title:     t.Title,
		Completed: t.Completed,
		CreatedAt: t.CreatedAt.UTC(),
		UpdatedAt: t.UpdatedAt.UTC(),
	}
}

type taskHandlers struct {
	svc TaskService
}

func (h *taskHandlers) list(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.List(r.Context(), userID(r.Context()), r.URL.Query().Get("status"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	out := make([]taskResponse, len(tasks))
	for i, t := range tasks {
		out[i] = newTaskResponse(t)
	}
	WriteData(w, http.StatusOK, out)
}

func (h *taskHandlers) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
	}
	if e := DecodeJSON(w, r, &req); e != nil {
		WriteError(w, e)
		return
	}
	task, err := h.svc.Create(r.Context(), userID(r.Context()), req.Title)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	WriteData(w, http.StatusCreated, newTaskResponse(task))
}

func (h *taskHandlers) update(w http.ResponseWriter, r *http.Request) {
	id, ok := taskID(r)
	if !ok {
		WriteError(w, MalformedRequest())
		return
	}
	var req struct {
		Title     *string `json:"title"`
		Completed *bool   `json:"completed"`
	}
	if e := DecodeJSON(w, r, &req); e != nil {
		WriteError(w, e)
		return
	}
	task, err := h.svc.Update(r.Context(), userID(r.Context()), id, services.TaskUpdate{Title: req.Title, Completed: req.Completed})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	WriteData(w, http.StatusOK, newTaskResponse(task))
}

func (h *taskHandlers) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := taskID(r)
	if !ok {
		WriteError(w, MalformedRequest())
		return
	}
	if err := h.svc.Delete(r.Context(), userID(r.Context()), id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// taskID reads the {id} path segment; IDs are positive integers.
func taskID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}
