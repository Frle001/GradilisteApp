package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/handlers"
)

func RegisterInventoryRoutes(
	api *gin.RouterGroup,
	h *handlers.InventoryHandler,
	authRequired gin.HandlerFunc,
	requireRoles func(...string) gin.HandlerFunc,
) {
	allRoles := requireRoles("direktor", "inzenjer", "administracija", "poslovoda")
	managementRoles := requireRoles("direktor", "inzenjer", "administracija")
	canTransfer := requireRoles("direktor", "inzenjer", "administracija", "poslovoda")

	rg := api.Group("/inventory", authRequired, allRoles)
	{
		rg.GET("/me", h.GetMyInventory)
		rg.GET("/employees/:employeeId", managementRoles, h.GetEmployeeInventory)
		rg.GET("/transfer-targets", canTransfer, h.GetTransferTargets)
		rg.GET("/employee-projects", canTransfer, h.GetEmployeeProjects)
		rg.POST("/transfers", canTransfer, h.CreateTransfer)
		rg.GET("/transfers", h.ListTransfers)
	}
}
