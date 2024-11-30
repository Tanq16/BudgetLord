package api

import (
	"encoding/json"
	"net/http"
	"path"
	"time"

	"github.com/tanq16/budgetlord/internal/models"
	"github.com/tanq16/budgetlord/internal/storage"
	"github.com/tanq16/budgetlord/internal/web"
)

type Handler struct {
	storage storage.Storage
}

func NewHandler(s storage.Storage) *Handler {
	return &Handler{
		storage: s,
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ExpenseRequest struct {
	Name     string    `json:"name"`
	Category string    `json:"category"`
	Amount   float64   `json:"amount"`
	Date     time.Time `json:"date"`
}

func (h *Handler) AddExpense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	expense := &models.Expense{
		Name:     req.Name,
		Category: req.Category,
		Amount:   req.Amount,
		Date:     req.Date,
	}

	if err := expense.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.storage.SaveExpense(expense); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to save expense"})
		return
	}

	writeJSON(w, http.StatusOK, expense)
}

func (h *Handler) GetExpenses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	expenses, err := h.storage.GetAllExpenses()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve expenses"})
		return
	}

	writeJSON(w, http.StatusOK, expenses)
}

func (h *Handler) ServeTableView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := web.ServeTemplate(w, "table.html"); err != nil {
		http.Error(w, "Failed to serve template", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "ID parameter is required"})
		return
	}

	if err := h.storage.DeleteExpense(id); err != nil {
		if err == storage.ErrExpenseNotFound {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Expense not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete expense"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// PWA handlers
func (h *Handler) ServeManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := web.ServeTemplate(w, "manifest.json"); err != nil {
		http.Error(w, "Failed to serve manifest", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) ServeServiceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	if err := web.ServeTemplate(w, "sw.js"); err != nil {
		http.Error(w, "Failed to serve service worker", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) ServePWAIcon(w http.ResponseWriter, r *http.Request) {
	iconName := path.Base(r.URL.Path)
	w.Header().Set("Content-Type", "image/png")
	if err := web.ServeTemplate(w, "pwa/"+iconName); err != nil {
		http.Error(w, "Failed to serve icon", http.StatusInternalServerError)
		return
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
