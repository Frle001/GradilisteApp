package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

var (
	ErrPurchaseNotFound     = errors.New("purchase not found")
	ErrReceiptNotFound      = errors.New("receipt not found")
	ErrProjectNotActive     = errors.New("project not found or not active")
	ErrProjectNotAssigned   = errors.New("project is not assigned to you")
	ErrNoItems              = errors.New("purchase must contain at least one item")
	ErrInvalidItem          = errors.New("invalid item: quantity must be positive and unit required")
	ErrMaterialNotInProject = errors.New("one or more materials do not belong to the selected project")
)

const (
	defaultPerPage = 25
	maxPerPage     = 100
)

type MaterialPurchasesService struct {
	repo    *repositories.MaterialPurchasesRepository
	audit   *repositories.AuditRepository
	storage StorageService
}

func NewMaterialPurchasesService(
	repo *repositories.MaterialPurchasesRepository,
	audit *repositories.AuditRepository,
	storage StorageService,
) *MaterialPurchasesService {
	return &MaterialPurchasesService{repo: repo, audit: audit, storage: storage}
}

// ── Form data ─────────────────────────────────────────────────────────────────

func (s *MaterialPurchasesService) GetFormData(ctx context.Context, companyID, role, employeeID, projectID string) (*dto.PurchaseFormDataResponse, error) {
	projects, err := s.repo.GetFormProjects(ctx, companyID, role, employeeID)
	if err != nil {
		return nil, fmt.Errorf("get form projects: %w", err)
	}

	var materials []dto.PurchaseFormMaterial
	if projectID != "" {
		// If poslovoda, verify project is actually theirs before returning materials
		if role == "poslovoda" {
			assigned, err := s.repo.IsProjectAssigned(ctx, companyID, projectID, employeeID)
			if err != nil {
				return nil, err
			}
			if !assigned {
				return nil, ErrForbidden
			}
		}
		materials, err = s.repo.GetFormMaterials(ctx, companyID, projectID)
		if err != nil {
			return nil, fmt.Errorf("get form materials: %w", err)
		}
	}
	if materials == nil {
		materials = []dto.PurchaseFormMaterial{}
	}

	return &dto.PurchaseFormDataResponse{
		Projects:  projects,
		Materials: materials,
	}, nil
}

// ── List ──────────────────────────────────────────────────────────────────────

func (s *MaterialPurchasesService) List(ctx context.Context, companyID, role, employeeID string, f dto.PurchaseFilter, page, perPage int) (*dto.PurchaseListResponse, error) {
	// Poslovoda: restrict to own purchases only
	if role == "poslovoda" {
		f.BuyerID = &employeeID
	}

	page, perPage = clampPage(page, perPage)
	limit := perPage
	offset := (page - 1) * perPage

	total, err := s.repo.Count(ctx, companyID, f)
	if err != nil {
		return nil, fmt.Errorf("count purchases: %w", err)
	}

	data, err := s.repo.List(ctx, companyID, f, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list purchases: %w", err)
	}

	totalPages := 1
	if perPage > 0 && total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	return &dto.PurchaseListResponse{
		Data: data,
		Pagination: dto.PurchasePagination{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// ── Detail ────────────────────────────────────────────────────────────────────

func (s *MaterialPurchasesService) GetByID(ctx context.Context, companyID, role, employeeID, id string) (*dto.PurchaseDetail, error) {
	d, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, ErrPurchaseNotFound
	}
	// Poslovoda may only see own purchases
	if role == "poslovoda" && d.Buyer.ID != employeeID {
		return nil, ErrForbidden
	}
	return d, nil
}

// ── Receipt ───────────────────────────────────────────────────────────────────

func (s *MaterialPurchasesService) GetReceipt(ctx context.Context, companyID, role, employeeID, id string) (reader io.ReadCloser, contentType, originalFilename string, err error) {
	fileKey, origName, err := s.repo.GetFileKey(ctx, companyID, id)
	if err != nil {
		return nil, "", "", err
	}
	if fileKey == "" {
		// Check if purchase exists at all
		d, err := s.repo.GetByID(ctx, companyID, id)
		if err != nil {
			return nil, "", "", err
		}
		if d == nil {
			return nil, "", "", ErrPurchaseNotFound
		}
		if role == "poslovoda" && d.Buyer.ID != employeeID {
			return nil, "", "", ErrForbidden
		}
		return nil, "", "", ErrReceiptNotFound
	}

	// Permission check
	d, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, "", "", err
	}
	if d == nil {
		return nil, "", "", ErrPurchaseNotFound
	}
	if role == "poslovoda" && d.Buyer.ID != employeeID {
		return nil, "", "", ErrForbidden
	}

	r, ct, err := s.storage.ReadReceiptFile(ctx, fileKey)
	if err != nil {
		return nil, "", "", fmt.Errorf("read receipt file: %w", err)
	}
	return r, ct, origName, nil
}

// ── Create ────────────────────────────────────────────────────────────────────

func (s *MaterialPurchasesService) Create(
	ctx context.Context,
	companyID, role, employeeID, userID string,
	projectID, itemsJSON, notesStr, purchasedAtStr string,
	receiptFile multipart.File,
	receiptHeader *multipart.FileHeader,
) (string, error) {
	// ── Parse & validate inputs ───────────────────────────────────────────────

	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return "", fmt.Errorf("project_id is required")
	}

	var items []dto.CreatePurchaseItemRequest
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return "", fmt.Errorf("invalid items JSON: %w", err)
	}
	if len(items) == 0 {
		return "", ErrNoItems
	}
	for _, item := range items {
		if item.Quantity <= 0 || strings.TrimSpace(item.Unit) == "" || strings.TrimSpace(item.ProjectMaterialID) == "" {
			return "", ErrInvalidItem
		}
	}

	var purchasedAt time.Time
	if purchasedAtStr != "" {
		t, err := time.Parse("2006-01-02", purchasedAtStr)
		if err != nil {
			t, err = time.Parse(time.RFC3339, purchasedAtStr)
			if err != nil {
				return "", fmt.Errorf("invalid purchased_at: use YYYY-MM-DD or ISO 8601")
			}
		}
		purchasedAt = t
	} else {
		purchasedAt = time.Now()
	}

	var notes *string
	if n := strings.TrimSpace(notesStr); n != "" {
		notes = &n
	}

	// ── Validate receipt file ──────────────────────────────────────────────────
	if receiptHeader != nil {
		if err := ValidateReceiptFile(receiptHeader); err != nil {
			return "", err
		}
	}

	// ── Validate project ──────────────────────────────────────────────────────
	projectName, err := s.repo.GetActiveProject(ctx, companyID, projectID)
	if err != nil {
		return "", err
	}
	if projectName == "" {
		return "", ErrProjectNotActive
	}

	if role == "poslovoda" {
		assigned, err := s.repo.IsProjectAssigned(ctx, companyID, projectID, employeeID)
		if err != nil {
			return "", err
		}
		if !assigned {
			return "", ErrProjectNotAssigned
		}
	}

	// ── Validate materials ────────────────────────────────────────────────────
	materialIDs := make([]string, len(items))
	for i, item := range items {
		materialIDs[i] = item.ProjectMaterialID
	}
	validMaterials, err := s.repo.ValidateMaterials(ctx, companyID, projectID, materialIDs)
	if err != nil {
		return "", err
	}
	for _, item := range items {
		if _, ok := validMaterials[item.ProjectMaterialID]; !ok {
			return "", ErrMaterialNotInProject
		}
	}

	// ── Save receipt file (before DB transaction) ─────────────────────────────
	var fileKey, origFilename *string
	if receiptFile != nil && receiptHeader != nil {
		key, name, err := s.storage.SaveReceiptFile(ctx, receiptFile, receiptHeader, companyID, projectID)
		if err != nil {
			return "", fmt.Errorf("save receipt file: %w", err)
		}
		fileKey = &key
		origFilename = &name
	}

	// ── DB transaction ────────────────────────────────────────────────────────
	params := repositories.CreatePurchaseParams{
		CompanyID:    companyID,
		ProjectID:    projectID,
		BuyerID:      employeeID,
		UserID:       userID,
		PurchasedAt:  purchasedAt,
		Notes:        notes,
		FileKey:      fileKey,
		OrigFilename: origFilename,
		Items:        items,
	}

	sessionID, err := s.repo.Create(ctx, params)
	if err != nil {
		// Clean up file if transaction failed
		if fileKey != nil {
			if cleanErr := s.storage.DeleteReceiptFile(ctx, *fileKey); cleanErr != nil {
				log.Printf("[material_purchases] failed to cleanup receipt file %s after tx error: %v", *fileKey, cleanErr)
			}
		}
		return "", fmt.Errorf("create purchase: %w", err)
	}

	// ── Audit log ─────────────────────────────────────────────────────────────
	newData := map[string]interface{}{
		"session_id":   sessionID,
		"project_id":   projectID,
		"project_name": projectName,
		"buyer_id":     employeeID,
		"items_count":  len(items),
		"purchased_at": purchasedAt,
		"has_receipt":  fileKey != nil,
	}
	s.audit.Log(ctx, repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &userID,
		EmployeeID: &employeeID,
		Action:     "material_purchase.create",
		EntityType: "material_purchase_sessions",
		EntityID:   &sessionID,
		NewData:    newData,
	})

	return sessionID, nil
}

func clampPage(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	return page, perPage
}
