package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/jackc/pgx/v5"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Interfaces ────────────────────────────────────────────────────────────────

type financeRepo interface {
	CreateInvoice(ctx context.Context, p repositories.CreateInvoiceParams) (*repositories.CompanyInvoiceRow, error)
	ListInvoices(ctx context.Context, companyID string) ([]repositories.CompanyInvoiceRow, error)
	GetInvoice(ctx context.Context, companyID, invoiceID string) (*repositories.CompanyInvoiceRow, error)
	CreateR1Receipt(ctx context.Context, p repositories.CreateR1Params) (*repositories.R1ReceiptRow, error)
	ListR1Receipts(ctx context.Context, companyID string, submittedBy *string) ([]repositories.R1ReceiptRow, error)
	GetR1Receipt(ctx context.Context, companyID, receiptID string) (*repositories.R1ReceiptRow, error)
}

type financeStorage interface {
	SaveFinanceDocFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, companyID, subdir string) (fileKey, contentType string, fileSize int64, err error)
	ReadFinanceDocFile(ctx context.Context, fileKey string) (io.ReadCloser, string, error)
}

// ── Service ───────────────────────────────────────────────────────────────────

type FinanceService struct {
	repo    financeRepo
	storage financeStorage
}

func NewFinanceService(repo *repositories.FinanceRepository, storage StorageService) *FinanceService {
	return &FinanceService{repo: repo, storage: storage}
}

// ── Access control ────────────────────────────────────────────────────────────

func requireRacuniAccess(role string) error {
	if role != "direktor" && role != "administracija" {
		return ErrForbidden
	}
	return nil
}

func requireR1CreateAccess(role string) error {
	if role == "administracija" {
		return ErrForbidden
	}
	return nil
}

func requireR1ReadAccess(_ string) error {
	return nil
}

// ── Validation ────────────────────────────────────────────────────────────────

func validateInvoiceFields(invoiceType string, supplier, leasingCompany *string) error {
	if !dto.AllowedInvoiceTypes[invoiceType] {
		return validationErr("Neispravna vrsta računa")
	}
	switch invoiceType {
	case dto.InvoiceTypeMaterijal:
		if supplier == nil || *supplier == "" {
			return validationErr("Dobavljač je obavezan za vrstu 'materijal'")
		}
		if !dto.AllowedSuppliers[*supplier] {
			return validationErr("Nepoznati dobavljač: " + *supplier)
		}
		if leasingCompany != nil {
			return validationErr("Polje 'leasing_company' nije dopušteno za vrstu 'materijal'")
		}
	case dto.InvoiceTypeLeasing:
		if leasingCompany == nil || *leasingCompany == "" {
			return validationErr("Firma je obavezna za vrstu 'leasing'")
		}
		if !dto.AllowedLeasingCompanies[*leasingCompany] {
			return validationErr("Nepoznata leasing firma: " + *leasingCompany)
		}
		if supplier != nil {
			return validationErr("Polje 'supplier' nije dopušteno za vrstu 'leasing'")
		}
	default:
		if supplier != nil {
			return validationErr("Polje 'supplier' nije dopušteno za vrstu '" + invoiceType + "'")
		}
		if leasingCompany != nil {
			return validationErr("Polje 'leasing_company' nije dopušteno za vrstu '" + invoiceType + "'")
		}
	}
	return nil
}

// ── Računi ────────────────────────────────────────────────────────────────────

func (s *FinanceService) CreateInvoice(
	ctx context.Context,
	companyID, role, userID string,
	invoiceType string,
	supplier, leasingCompany *string,
	file multipart.File,
	header *multipart.FileHeader,
) (*dto.CompanyInvoice, error) {
	if err := requireRacuniAccess(role); err != nil {
		return nil, err
	}
	if err := validateInvoiceFields(invoiceType, supplier, leasingCompany); err != nil {
		return nil, err
	}
	if file == nil || header == nil {
		return nil, validationErr("Dokument je obavezan")
	}

	fileKey, contentType, fileSize, err := s.storage.SaveFinanceDocFile(ctx, file, header, companyID, "racuni")
	if err != nil {
		return nil, fmt.Errorf("finance.CreateInvoice upload: %w", err)
	}

	row, err := s.repo.CreateInvoice(ctx, repositories.CreateInvoiceParams{
		CompanyID:        companyID,
		InvoiceType:      invoiceType,
		Supplier:         supplier,
		LeasingCompany:   leasingCompany,
		StorageKey:       fileKey,
		OriginalFilename: header.Filename,
		MimeType:         contentType,
		FileSize:         fileSize,
		CreatedBy:        userID,
	})
	if err != nil {
		return nil, fmt.Errorf("finance.CreateInvoice: %w", err)
	}
	return toInvoiceDTO(*row), nil
}

func (s *FinanceService) ListInvoices(ctx context.Context, companyID, role string) ([]dto.CompanyInvoice, error) {
	if err := requireRacuniAccess(role); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListInvoices(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("finance.ListInvoices: %w", err)
	}
	out := make([]dto.CompanyInvoice, len(rows))
	for i, r := range rows {
		out[i] = *toInvoiceDTO(r)
	}
	return out, nil
}

func (s *FinanceService) DownloadInvoice(ctx context.Context, companyID, role, invoiceID string) (io.ReadCloser, string, string, error) {
	if err := requireRacuniAccess(role); err != nil {
		return nil, "", "", err
	}
	row, err := s.repo.GetInvoice(ctx, companyID, invoiceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", "", ErrNotFound
		}
		return nil, "", "", fmt.Errorf("finance.DownloadInvoice get: %w", err)
	}
	reader, ct, err := s.storage.ReadFinanceDocFile(ctx, row.StorageKey)
	if err != nil {
		return nil, "", "", fmt.Errorf("finance.DownloadInvoice read: %w", err)
	}
	return reader, ct, row.OriginalFilename, nil
}

// ── R1 Receipts ───────────────────────────────────────────────────────────────

func (s *FinanceService) CreateR1Receipt(
	ctx context.Context,
	companyID, role, userID string,
	price float64,
	file multipart.File,
	header *multipart.FileHeader,
) (*dto.R1Receipt, error) {
	if err := requireR1CreateAccess(role); err != nil {
		return nil, err
	}
	if price <= 0 {
		return nil, validationErr("Cijena mora biti veća od nule")
	}
	if file == nil || header == nil {
		return nil, validationErr("Dokument je obavezan")
	}

	fileKey, contentType, fileSize, err := s.storage.SaveFinanceDocFile(ctx, file, header, companyID, "r1")
	if err != nil {
		return nil, fmt.Errorf("finance.CreateR1Receipt upload: %w", err)
	}

	row, err := s.repo.CreateR1Receipt(ctx, repositories.CreateR1Params{
		CompanyID:        companyID,
		SubmittedBy:      userID,
		Price:            price,
		StorageKey:       fileKey,
		OriginalFilename: header.Filename,
		MimeType:         contentType,
		FileSize:         fileSize,
	})
	if err != nil {
		return nil, fmt.Errorf("finance.CreateR1Receipt: %w", err)
	}
	return toR1DTO(*row), nil
}

func (s *FinanceService) ListR1Receipts(ctx context.Context, companyID, role, userID string) ([]dto.R1Receipt, error) {
	if err := requireR1ReadAccess(role); err != nil {
		return nil, err
	}
	var filter *string
	if role != "direktor" && role != "administracija" {
		filter = &userID
	}
	rows, err := s.repo.ListR1Receipts(ctx, companyID, filter)
	if err != nil {
		return nil, fmt.Errorf("finance.ListR1Receipts: %w", err)
	}
	out := make([]dto.R1Receipt, len(rows))
	for i, r := range rows {
		out[i] = *toR1DTO(r)
	}
	return out, nil
}

func (s *FinanceService) DownloadR1Receipt(ctx context.Context, companyID, role, userID, receiptID string) (io.ReadCloser, string, string, error) {
	if err := requireR1ReadAccess(role); err != nil {
		return nil, "", "", err
	}
	row, err := s.repo.GetR1Receipt(ctx, companyID, receiptID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", "", ErrNotFound
		}
		return nil, "", "", fmt.Errorf("finance.DownloadR1Receipt get: %w", err)
	}
	// Direktor and administracija may download any same-company receipt.
	if role != "direktor" && role != "administracija" && row.SubmittedBy != userID {
		return nil, "", "", ErrForbidden
	}
	reader, ct, err := s.storage.ReadFinanceDocFile(ctx, row.StorageKey)
	if err != nil {
		return nil, "", "", fmt.Errorf("finance.DownloadR1Receipt read: %w", err)
	}
	return reader, ct, row.OriginalFilename, nil
}

// ── Converters ────────────────────────────────────────────────────────────────

func toInvoiceDTO(r repositories.CompanyInvoiceRow) *dto.CompanyInvoice {
	return &dto.CompanyInvoice{
		ID:             r.ID,
		InvoiceType:    r.InvoiceType,
		Supplier:       r.Supplier,
		LeasingCompany: r.LeasingCompany,
		Document: dto.FinanceDocMeta{
			OriginalFilename: r.OriginalFilename,
			MimeType:         r.MimeType,
			FileSize:         r.FileSize,
			UploadedAt:       r.CreatedAt,
		},
		CreatedBy:     r.CreatedBy,
		CreatedByName: r.CreatedByName,
		CreatedAt:     r.CreatedAt,
	}
}

func toR1DTO(r repositories.R1ReceiptRow) *dto.R1Receipt {
	return &dto.R1Receipt{
		ID:            r.ID,
		SubmittedBy:   r.SubmittedBy,
		SubmitterName: r.SubmitterName,
		Price:         r.Price,
		Document: dto.FinanceDocMeta{
			OriginalFilename: r.OriginalFilename,
			MimeType:         r.MimeType,
			FileSize:         r.FileSize,
			UploadedAt:       r.CreatedAt,
		},
		CreatedAt: r.CreatedAt,
	}
}
