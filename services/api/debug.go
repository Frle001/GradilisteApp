package main

import (
	"context"
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
		"companies":               &summary.Companies,
		"users":                   &summary.Users,
		"employees":               &summary.Employees,
		"projects":                &summary.Projects,
		"project_assignments":     &summary.ProjectAssignments,
		"project_materials":       &summary.ProjectMaterials,
		"daily_reports":           &summary.DailyReports,
		"employee_assets":         &summary.EmployeeAssets,
		"material_purchase_sessions": &summary.MaterialPurchases,
		"audit_logs":              &summary.AuditLogs,
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
