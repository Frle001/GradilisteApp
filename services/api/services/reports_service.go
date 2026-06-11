package services

import (
	"bytes"
	"context"
	"errors"
	"math"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

const (
	ExportRowLimit = 10_000
	DefaultPerPage = 25
	MaxPerPage     = 100
)

var ErrExportLimitExceeded = errors.New("rezultati premašuju limit od 10 000 redova; suzite filtere i pokušajte ponovo")

type ReportsService struct {
	repo  *repositories.ReportsRepository
	excel *ExcelExportService
}

func NewReportsService(repo *repositories.ReportsRepository) *ReportsService {
	return &ReportsService{
		repo:  repo,
		excel: NewExcelExportService(),
	}
}

// ── Permission enforcement ────────────────────────────────────────────────────

// enforcePoslovodaFilter forces the poslovoda_id filter to the caller's own employee ID
// for the poslovoda role. For other roles, the filter passes through unchanged.
func enforcePoslovodaFilter(callerRole, callerEmpID string, f *dto.ReportFilter) {
	if callerRole == "poslovoda" {
		f.PoslovodaID = &callerEmpID
	}
}

func clampPagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	return page, perPage
}

// ── Građevinski dnevnik ───────────────────────────────────────────────────────

func (s *ReportsService) ListDnevnik(
	ctx context.Context,
	companyID, callerRole, callerEmpID string,
	f dto.ReportFilter,
	page, perPage int,
) (*dto.GradevinskiDnevnikResponse, error) {
	enforcePoslovodaFilter(callerRole, callerEmpID, &f)
	page, perPage = clampPagination(page, perPage)

	total, err := s.repo.CountDnevnik(ctx, companyID, f)
	if err != nil {
		return nil, err
	}

	data, err := s.repo.ListDnevnik(ctx, companyID, f, page, perPage)
	if err != nil {
		return nil, err
	}

	summary, err := s.repo.SummaryDnevnik(ctx, companyID, f)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &dto.GradevinskiDnevnikResponse{
		Data: data,
		Pagination: dto.ReportPagination{
			Page: page, PerPage: perPage,
			Total: total, TotalPages: totalPages,
		},
		Summary: summary,
	}, nil
}

func (s *ReportsService) ExportDnevnik(
	ctx context.Context,
	companyID, callerRole, callerEmpID string,
	f dto.ReportFilter,
) (*bytes.Buffer, error) {
	enforcePoslovodaFilter(callerRole, callerEmpID, &f)

	rows, err := s.repo.ExportDnevnik(ctx, companyID, f, ExportRowLimit)
	if err != nil {
		return nil, err
	}
	if len(rows) > ExportRowLimit {
		return nil, ErrExportLimitExceeded
	}

	summary, err := s.repo.SummaryDnevnik(ctx, companyID, f)
	if err != nil {
		return nil, err
	}

	return s.excel.BuildDnevnikXLSX(rows, summary)
}

// ── Građevinska knjiga ────────────────────────────────────────────────────────

func (s *ReportsService) ListKnjiga(
	ctx context.Context,
	companyID, callerRole, callerEmpID string,
	f dto.ReportFilter,
	page, perPage int,
) (*dto.GradevinskaKnjigaResponse, error) {
	enforcePoslovodaFilter(callerRole, callerEmpID, &f)
	page, perPage = clampPagination(page, perPage)

	total, err := s.repo.CountKnjiga(ctx, companyID, f)
	if err != nil {
		return nil, err
	}

	data, err := s.repo.ListKnjiga(ctx, companyID, f, page, perPage)
	if err != nil {
		return nil, err
	}

	summary, err := s.repo.SummaryKnjiga(ctx, companyID, f)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &dto.GradevinskaKnjigaResponse{
		Data: data,
		Pagination: dto.ReportPagination{
			Page: page, PerPage: perPage,
			Total: total, TotalPages: totalPages,
		},
		Summary: summary,
	}, nil
}

func (s *ReportsService) ExportKnjiga(
	ctx context.Context,
	companyID, callerRole, callerEmpID string,
	f dto.ReportFilter,
) (*bytes.Buffer, error) {
	enforcePoslovodaFilter(callerRole, callerEmpID, &f)

	rows, err := s.repo.ExportKnjiga(ctx, companyID, f, ExportRowLimit)
	if err != nil {
		return nil, err
	}
	if len(rows) > ExportRowLimit {
		return nil, ErrExportLimitExceeded
	}

	summary, err := s.repo.SummaryKnjiga(ctx, companyID, f)
	if err != nil {
		return nil, err
	}

	return s.excel.BuildKnjigaXLSX(rows, summary)
}

// ── Filter options ────────────────────────────────────────────────────────────

func (s *ReportsService) GetFilterOptions(
	ctx context.Context,
	companyID, callerRole, callerEmpID string,
) (dto.ReportFilterOptions, error) {
	if callerRole == "poslovoda" {
		return s.repo.FilterOptionsForPoslovoda(ctx, companyID, callerEmpID)
	}
	return s.repo.FilterOptionsForAll(ctx, companyID)
}
