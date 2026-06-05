package handlers

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/services"
)

type MaterialPurchasesHandler struct {
	svc *services.MaterialPurchasesService
}

func NewMaterialPurchasesHandler(svc *services.MaterialPurchasesService) *MaterialPurchasesHandler {
	return &MaterialPurchasesHandler{svc: svc}
}

// GET /api/material-purchases/form-data?project_id=...
func (h *MaterialPurchasesHandler) FormData(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	projectID := c.Query("project_id")

	data, err := h.svc.GetFormData(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, projectID)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		log.Printf("[material_purchases] FormData error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /api/material-purchases
func (h *MaterialPurchasesHandler) List(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	f := dto.PurchaseFilter{}
	if v := c.Query("project_id"); v != "" {
		f.ProjectID = &v
	}
	if v := c.Query("buyer_id"); v != "" {
		f.BuyerID = &v
	}
	if v := c.Query("date_from"); v != "" {
		f.DateFrom = &v
	}
	if v := c.Query("date_to"); v != "" {
		f.DateTo = &v
	}
	if v := c.Query("search"); v != "" {
		f.Search = &v
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "25"))

	resp, err := h.svc.List(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, f, page, perPage)
	if err != nil {
		log.Printf("[material_purchases] List error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/material-purchases/:id
func (h *MaterialPurchasesHandler) GetByID(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	id := c.Param("id")

	d, err := h.svc.GetByID(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, id)
	if err != nil {
		if errors.Is(err, services.ErrPurchaseNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase not found"})
			return
		}
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		log.Printf("[material_purchases] GetByID error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, d)
}

// GET /api/material-purchases/:id/receipt
func (h *MaterialPurchasesHandler) GetReceipt(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	id := c.Param("id")

	reader, contentType, origFilename, err := h.svc.GetReceipt(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, id)
	if err != nil {
		if errors.Is(err, services.ErrPurchaseNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase not found"})
			return
		}
		if errors.Is(err, services.ErrReceiptNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no receipt for this purchase"})
			return
		}
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		if errors.Is(err, os.ErrNotExist) {
			c.JSON(http.StatusNotFound, gin.H{"error": "receipt file not found on server"})
			return
		}
		log.Printf("[material_purchases] GetReceipt error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	disposition := "inline"
	if origFilename != "" {
		disposition = `inline; filename="` + origFilename + `"`
	}
	c.DataFromReader(http.StatusOK, -1, contentType, reader, map[string]string{
		"Content-Disposition": disposition,
	})
}

// POST /api/material-purchases
func (h *MaterialPurchasesHandler) Create(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	projectID := c.PostForm("project_id")
	itemsJSON := c.PostForm("items")
	notes := c.PostForm("notes")
	purchasedAt := c.PostForm("purchased_at")

	if itemsJSON == "" {
		itemsJSON = "[]"
	}

	// Optional receipt file
	receiptFile, receiptHeader, _ := c.Request.FormFile("receipt")
	if receiptFile != nil {
		defer receiptFile.Close()
	}

	id, err := h.svc.Create(
		c.Request.Context(),
		u.CompanyID, u.Role, u.EmployeeID, u.UserID,
		projectID, itemsJSON, notes, purchasedAt,
		receiptFile, receiptHeader,
	)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrForbidden) || errors.Is(err, services.ErrProjectNotAssigned) {
			status = http.StatusForbidden
		} else if errors.Is(err, services.ErrProjectNotActive) || errors.Is(err, services.ErrMaterialNotInProject) ||
			errors.Is(err, services.ErrNoItems) || errors.Is(err, services.ErrInvalidItem) {
			status = http.StatusUnprocessableEntity
		} else if status == http.StatusBadRequest {
			// Other errors (json parse, etc.) stay 400; unexpected errors become 500
			// Check if it's an unexpected internal error
			log.Printf("[material_purchases] Create error: %v", err)
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// DELETE /api/material-purchases/:id
func (h *MaterialPurchasesHandler) Delete(c *gin.Context) {
	c.JSON(http.StatusMethodNotAllowed, gin.H{
		"error": "Material purchases are not permanently deleted in this phase.",
	})
}
