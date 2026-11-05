package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/services"
)

type ColdOpsHandler struct {
	svc *services.ColdOpsService
}

func NewColdOpsHandler(svc *services.ColdOpsService) *ColdOpsHandler {
	return &ColdOpsHandler{svc: svc}
}

func respondColdOpsError(c *gin.Context, err error) {
	var ve *services.ValidationError
	switch {
	case errors.Is(err, services.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "Pristup nije dozvoljen"})
	case errors.Is(err, services.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Nije pronaÄ‘eno"})
	case errors.As(err, &ve):
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Interna greĹˇka"})
	}
}

// â”€â”€ Summary â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (h *ColdOpsHandler) GetSummary(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	summary, err := h.svc.GetSummary(c.Request.Context(), u.CompanyID, u.Role)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// â”€â”€ Categories â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (h *ColdOpsHandler) ListCategories(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	cats, err := h.svc.ListCategories(c.Request.Context(), u.CompanyID, u.Role)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": cats})
}

func (h *ColdOpsHandler) CreateCategory(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var req dto.CreateCategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}
	cat, err := h.svc.CreateCategory(c.Request.Context(), u.CompanyID, u.UserID, u.Role, req)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"category": cat})
}

func (h *ColdOpsHandler) UpdateCategory(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")
	var req dto.UpdateCategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}
	cat, err := h.svc.UpdateCategory(c.Request.Context(), u.CompanyID, u.Role, id, req)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"category": cat})
}

func (h *ColdOpsHandler) DeleteCategory(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")
	if err := h.svc.DeleteCategory(c.Request.Context(), u.CompanyID, u.Role, id); err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// â”€â”€ Expenses â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (h *ColdOpsHandler) ListExpenses(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	f := dto.ListExpensesFilter{
		Page:     queryInt(c, "page", 1),
		PageSize: queryInt(c, "page_size", 50),
	}
	if v := c.Query("category_id"); v != "" {
		f.CategoryID = &v
	}
	if v := c.Query("expense_type"); v != "" {
		f.ExpenseType = &v
	}
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			f.From = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			f.To = &t
		}
	}
	expenses, err := h.svc.ListExpenses(c.Request.Context(), u.CompanyID, u.Role, f)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"expenses": expenses})
}

func (h *ColdOpsHandler) CreateExpense(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var req dto.CreateExpenseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}
	exp, err := h.svc.CreateExpense(c.Request.Context(), u.CompanyID, u.UserID, u.Role, req)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"expense": exp})
}

func (h *ColdOpsHandler) GetExpense(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	exp, err := h.svc.GetExpense(c.Request.Context(), u.CompanyID, u.Role, c.Param("id"))
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"expense": exp})
}

func (h *ColdOpsHandler) UpdateExpense(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var req dto.UpdateExpenseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}
	exp, err := h.svc.UpdateExpense(c.Request.Context(), u.CompanyID, u.Role, c.Param("id"), req)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"expense": exp})
}

func (h *ColdOpsHandler) DeleteExpense(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if err := h.svc.DeleteExpense(c.Request.Context(), u.CompanyID, u.Role, c.Param("id")); err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// â”€â”€ Recurring definitions â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (h *ColdOpsHandler) ListRecurringDefs(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	activeOnly := c.Query("active_only") == "true"
	defs, err := h.svc.ListRecurringDefs(c.Request.Context(), u.CompanyID, u.Role, activeOnly)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"definitions": defs})
}

func (h *ColdOpsHandler) CreateRecurringDef(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var req dto.CreateRecurringReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}
	def, err := h.svc.CreateRecurringDef(c.Request.Context(), u.CompanyID, u.UserID, u.Role, req)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"definition": def})
}

func (h *ColdOpsHandler) UpdateRecurringDef(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var req dto.UpdateRecurringReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}
	def, err := h.svc.UpdateRecurringDef(c.Request.Context(), u.CompanyID, u.UserID, u.Role, c.Param("id"), req)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"definition": def})
}

// â”€â”€ Occurrences â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (h *ColdOpsHandler) ListOccurrences(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	defID := c.Param("defId")
	var from, to *time.Time
	var status *string
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			from = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			to = &t
		}
	}
	if v := c.Query("status"); v != "" {
		status = &v
	}
	occs, err := h.svc.ListOccurrences(c.Request.Context(), u.CompanyID, u.Role, defID, from, to, status)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"occurrences": occs})
}

func (h *ColdOpsHandler) ConfirmOccurrence(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var req dto.ConfirmOccurrenceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}
	occ, err := h.svc.ConfirmOccurrence(c.Request.Context(), u.CompanyID, u.UserID, u.Role, c.Param("id"), req)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"occurrence": occ})
}

func (h *ColdOpsHandler) SkipOccurrence(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var req dto.SkipOccurrenceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}
	occ, err := h.svc.SkipOccurrence(c.Request.Context(), u.CompanyID, u.UserID, u.Role, c.Param("id"), req)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"occurrence": occ})
}

// â”€â”€ Budgets â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (h *ColdOpsHandler) ListBudgets(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	year, month := parseYearMonth(c)
	budgets, err := h.svc.ListBudgets(c.Request.Context(), u.CompanyID, u.Role, year, month)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"budgets": budgets})
}

func (h *ColdOpsHandler) UpsertBudget(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var req dto.UpsertBudgetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}
	budget, err := h.svc.UpsertBudget(c.Request.Context(), u.CompanyID, u.UserID, u.Role, req)
	if err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"budget": budget})
}

func (h *ColdOpsHandler) DeleteBudget(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if err := h.svc.DeleteBudget(c.Request.Context(), u.CompanyID, u.Role, c.Param("id")); err != nil {
		respondColdOpsError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// â”€â”€ Query helper â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func queryInt(c *gin.Context, key string, def int) int {
	if v := c.Query(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
