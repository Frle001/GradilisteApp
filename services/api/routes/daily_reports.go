package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/handlers"
)

func RegisterDailyReportRoutes(
	api *gin.RouterGroup,
	h *handlers.DailyReportHandler,
	attachH *handlers.DailyReportAttachmentsHandler,
	authRequired gin.HandlerFunc,
	requireRoles func(...string) gin.HandlerFunc,
) {
	// Allowed for: poslovoda, direktor, inzenjer, administracija
	allRoles := requireRoles("direktor", "inzenjer", "administracija", "poslovoda")
	// Create/edit: poslovoda is primary; direktor/inzenjer allowed
	writeRoles := requireRoles("direktor", "inzenjer", "poslovoda")
	// Approve/reject: direktor/inzenjer only
	manageRoles := requireRoles("direktor", "inzenjer")

	dr := api.Group("/daily-reports", authRequired)
	{
		dr.GET("", allRoles, h.List)
		// form-data must come before /:id so Gin's router matches it as a literal path
		dr.GET("/form-data", writeRoles, h.GetFormData)
		dr.GET("/:id", allRoles, h.GetByID)
		dr.POST("", writeRoles, h.Create)
		dr.PUT("/:id", writeRoles, h.Update)
		dr.DELETE("/:id", h.Delete)
		dr.PATCH("/:id/approve", manageRoles, h.Approve)
		dr.PATCH("/:id/reject", manageRoles, h.Reject)

		// Attachments (photos)
		dr.GET("/:id/attachments", allRoles, attachH.List)
		dr.POST("/:id/attachments", writeRoles, attachH.Upload)
		dr.GET("/:id/attachments/:attachId/download", allRoles, attachH.Download)
		dr.DELETE("/:id/attachments/:attachId", writeRoles, attachH.Delete)
	}
}
