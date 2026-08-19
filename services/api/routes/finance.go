package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/handlers"
)

func RegisterFinanceRoutes(
	api *gin.RouterGroup,
	h *handlers.FinanceHandler,
	authRequired gin.HandlerFunc,
	requireRoles func(...string) gin.HandlerFunc,
) {
	// Računi: direktor + administracija only
	racuni := api.Group("/finance/racuni", authRequired, requireRoles("direktor", "administracija"))
	{
		racuni.POST("", h.CreateInvoice)
		racuni.GET("", h.ListInvoices)
		racuni.GET("/:invoiceId/download", h.DownloadInvoice)
	}

	// R1 create: administracija FORBIDDEN
	r1Write := api.Group("/finance/r1", authRequired, requireRoles("direktor", "inzenjer", "poslovoda", "radnik"))
	r1Write.POST("", h.CreateR1Receipt)

	// R1 read: administracija allowed (service enforces visibility — non-mgmt sees only own)
	r1Read := api.Group("/finance/r1", authRequired, requireRoles("direktor", "inzenjer", "poslovoda", "radnik", "administracija"))
	r1Read.GET("", h.ListR1Receipts)
	r1Read.GET("/:receiptId/download", h.DownloadR1Receipt)
}
