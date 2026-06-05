package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// DBSummary represents basic database statistics
type DBSummary struct {
	Companies              int64 `json:"companies"`
	Users                  int64 `json:"users"`
	Employees              int64 `json:"employees"`
	Projects               int64 `json:"projects"`
	ProjectAssignments     int64 `json:"project_assignments"`
	ProjectMaterials       int64 `json:"project_materials"`
	DailyReports           int64 `json:"daily_reports"`
	DailyReportWorkerHours int64 `json:"daily_report_worker_hours"`
	DailyReportActivities  int64 `json:"daily_report_activities"`
	EmployeeAssets         int64 `json:"employee_assets"`
	MaterialPurchases      int64 `json:"material_purchase_sessions"`
	AuditLogs              int64 `json:"audit_logs"`
}

// GetDBSummary queries basic table counts for debugging
func GetDBSummary(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := GetDB()
	if db == nil {
		c.JSON(500, gin.H{
			"status": "error",
			"error":  "database pool is not initialized",
		})
		return
	}

	summary := DBSummary{}

	// Query row counts from key tables
	tables := map[string]*int64{
		"companies":                  &summary.Companies,
		"users":                      &summary.Users,
		"employees":                  &summary.Employees,
		"projects":                   &summary.Projects,
		"project_assignments":        &summary.ProjectAssignments,
		"project_materials":          &summary.ProjectMaterials,
		"daily_reports":              &summary.DailyReports,
		"daily_report_worker_hours":  &summary.DailyReportWorkerHours,
		"daily_report_activities":    &summary.DailyReportActivities,
		"employee_assets":            &summary.EmployeeAssets,
		"material_purchase_sessions": &summary.MaterialPurchases,
		"audit_logs":                 &summary.AuditLogs,
	}

	for table, count := range tables {
		err := db.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(count)
		if err != nil {
			c.JSON(500, gin.H{
				"status": "error",
				"error":  "failed to query " + table + ": " + err.Error(),
			})
			return
		}
	}

	c.JSON(200, gin.H{
		"status": "ok",
		"data":   summary,
	})
}

// ReportsDebug runs the exact Dnevnik queries and surfaces any errors.
// GET /api/debug/reports — dev only, no auth required.
func ReportsDebug(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := GetDB()
	result := gin.H{}

	// 1. Simple count of worker_hours table
	var whCount int64
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM daily_report_worker_hours").Scan(&whCount); err != nil {
		result["worker_hours_count_error"] = err.Error()
	} else {
		result["worker_hours_count"] = whCount
	}

	// 2. Get any company_id for testing
	var companyID string
	if err := db.QueryRow(ctx, "SELECT id::text FROM companies LIMIT 1").Scan(&companyID); err != nil {
		result["company_id_error"] = err.Error()
		c.JSON(200, result)
		return
	}
	result["company_id"] = companyID

	// 3. Run the COUNT query exactly as the repository does
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM daily_report_worker_hours drwh
		JOIN daily_reports dr ON dr.id = drwh.daily_report_id
		JOIN projects p ON p.id = dr.project_id
		JOIN employees pe ON pe.id = dr.poslovoda_id
		JOIN employees we ON we.id = drwh.worker_id
		WHERE drwh.company_id = $1::uuid
	`)
	var countResult int
	if err := db.QueryRow(ctx, countSQL, companyID).Scan(&countResult); err != nil {
		result["count_query_error"] = err.Error()
	} else {
		result["count_query_result"] = countResult
	}

	// 4. Run the SUMMARY query
	summarySQL := `
		SELECT
			COALESCE(SUM(drwh.hours_worked), 0),
			COUNT(DISTINCT drwh.worker_id),
			COUNT(DISTINCT dr.project_id)
		FROM daily_report_worker_hours drwh
		JOIN daily_reports dr ON dr.id = drwh.daily_report_id
		JOIN projects p ON p.id = dr.project_id
		JOIN employees pe ON pe.id = dr.poslovoda_id
		JOIN employees we ON we.id = drwh.worker_id
		WHERE drwh.company_id = $1::uuid
	`
	var totalHours float64
	var workersCount, projectsCount int
	if err := db.QueryRow(ctx, summarySQL, companyID).Scan(&totalHours, &workersCount, &projectsCount); err != nil {
		result["summary_query_error"] = err.Error()
	} else {
		result["summary"] = gin.H{
			"total_hours":    totalHours,
			"workers_count":  workersCount,
			"projects_count": projectsCount,
		}
	}

	// 5. Run the LIST query (first page)
	listSQL := `
		SELECT
			dr.id::text,
			dr.report_date::text,
			dr.project_id::text,
			p.name,
			p.address,
			dr.poslovoda_id::text,
			pe.first_name||' '||pe.last_name,
			drwh.worker_id::text,
			we.first_name||' '||we.last_name,
			drwh.hours_worked,
			drwh.notes,
			dr.status
		FROM daily_report_worker_hours drwh
		JOIN daily_reports dr ON dr.id = drwh.daily_report_id
		JOIN projects p ON p.id = dr.project_id
		JOIN employees pe ON pe.id = dr.poslovoda_id
		JOIN employees we ON we.id = drwh.worker_id
		WHERE drwh.company_id = $1::uuid
		ORDER BY dr.report_date DESC, p.name, we.last_name, we.first_name
		LIMIT $2 OFFSET $3
	`
	rows, err := db.Query(ctx, listSQL, companyID, 5, 0)
	if err != nil {
		result["list_query_error"] = err.Error()
	} else {
		defer rows.Close()
		var scanErrors []string
		var rowCount int
		for rows.Next() {
			var (
				reportID, reportDate, projectID, projectName string
				projectAddress                               *string
				poslovodaID, poslovodaName                  string
				workerID, workerName                        string
				hoursWorked                                 float64
				notes                                       *string
				status                                      string
			)
			if err := rows.Scan(
				&reportID, &reportDate,
				&projectID, &projectName, &projectAddress,
				&poslovodaID, &poslovodaName,
				&workerID, &workerName,
				&hoursWorked, &notes, &status,
			); err != nil {
				scanErrors = append(scanErrors, fmt.Sprintf("row %d: %v", rowCount, err))
			}
			rowCount++
		}
		result["list_query_rows"] = rowCount
		if len(scanErrors) > 0 {
			result["list_scan_errors"] = scanErrors
		}
		if rows.Err() != nil {
			result["list_rows_error"] = rows.Err().Error()
		}
	}

	c.JSON(200, result)
}
