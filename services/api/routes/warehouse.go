package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/handlers"
)

func RegisterWarehouseRoutes(
	api *gin.RouterGroup,
	h *handlers.WarehouseHandler,
	authRequired gin.HandlerFunc,
	requireRoles func(...string) gin.HandlerFunc,
) {
	allRoles := requireRoles("direktor", "inzenjer", "administracija", "poslovoda", "radnik")
	writeRoles := requireRoles("direktor", "inzenjer", "administracija")
	issueRoles := requireRoles("direktor", "inzenjer", "administracija", "poslovoda")

	rg := api.Group("/warehouse", authRequired)
	{
		rg.GET("/overview", allRoles, h.GetOverview)
		rg.GET("/stock", allRoles, h.ListStock)
		rg.GET("/stock/:balanceId", allRoles, h.GetStock)
		rg.POST("/receipts", writeRoles, h.CreateReceipt)
		rg.POST("/issues", issueRoles, h.IssueToProject)
		rg.POST("/returns", issueRoles, h.ReturnFromProject)
		rg.POST("/adjustments", writeRoles, h.AdjustInventory)
		rg.GET("/movements", allRoles, h.ListMovements)
	}
}
