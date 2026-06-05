package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/handlers"
)

func RegisterReportsRoutes(
	api *gin.RouterGroup,
	h *handlers.ReportsHandler,
	authRequired gin.HandlerFunc,
	requireRoles func(...string) gin.HandlerFunc,
) {
	allRoles := requireRoles("direktor", "inzenjer", "administracija", "poslovoda")

	rg := api.Group("/reports", authRequired, allRoles)
	{
		rg.GET("/gradevinski-dnevnik", h.ListDnevnik)
		rg.GET("/gradevinski-dnevnik/export", h.ExportDnevnik)
		rg.GET("/gradevinska-knjiga", h.ListKnjiga)
		rg.GET("/gradevinska-knjiga/export", h.ExportKnjiga)
		rg.GET("/filter-options", h.FilterOptions)
	}
}
