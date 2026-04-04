package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"finance-dashboard/internal/domain"
	"finance-dashboard/internal/service"
	"finance-dashboard/pkg/response"
)

type DashboardHandler struct {
	dashSvc *service.DashboardService
}

func NewDashboardHandler(dashSvc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashSvc: dashSvc}
}

func (h *DashboardHandler) Summary(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")

	summary, err := h.dashSvc.GetSummary(from, to)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.Success(c, http.StatusOK, summary)
}

func (h *DashboardHandler) Trends(c *gin.Context) {
	trends, err := h.dashSvc.GetTrends()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.Success(c, http.StatusOK, trends)
}

func (h *DashboardHandler) Categories(c *gin.Context) {
	breakdown, err := h.dashSvc.GetCategoryBreakdown()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.Success(c, http.StatusOK, breakdown)
}

func (h *DashboardHandler) Recent(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// FIX: reject limit > 50 with 400 (test 19)
	if limit > 50 {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "limit cannot exceed 50")
		return
	}

	records, err := h.dashSvc.GetRecentActivity(limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.Success(c, http.StatusOK, records)
}

func (h *DashboardHandler) ListCategories(c *gin.Context) {
	cats, err := h.dashSvc.GetCategories()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.Success(c, http.StatusOK, cats)
}

func (h *DashboardHandler) CreateCategory(c *gin.Context) {
	var req domain.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	// FIX: was 422, now 400 (tests 7, 8)
	if err := validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	cat, err := h.dashSvc.CreateCategory(&req)
	if err != nil {
		// FIX: catch duplicate name error and return 409 instead of 500 (test 9)
		if strings.HasPrefix(err.Error(), "DUPLICATE:") {
			response.Error(c, http.StatusConflict, "CONFLICT", strings.TrimPrefix(err.Error(), "DUPLICATE:"))
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.Success(c, http.StatusCreated, cat)
}
