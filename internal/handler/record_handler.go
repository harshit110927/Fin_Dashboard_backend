package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"finance-dashboard/internal/domain"
	"finance-dashboard/internal/repository"
	"finance-dashboard/internal/service"
	"finance-dashboard/pkg/apperr"
	"finance-dashboard/pkg/response"
)

type RecordHandler struct {
	recordSvc *service.RecordService
	auditRepo repository.AuditRepo
}

func NewRecordHandler(recordSvc *service.RecordService, auditRepo repository.AuditRepo) *RecordHandler {
	return &RecordHandler{recordSvc: recordSvc, auditRepo: auditRepo}
}

func (h *RecordHandler) List(c *gin.Context) {
	categoryID, _ := strconv.Atoi(c.Query("category_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	filter := domain.RecordFilter{
		Type:       c.Query("type"),
		Status:     c.Query("status"),
		DateFrom:   c.Query("date_from"),
		DateTo:     c.Query("date_to"),
		Sort:       c.Query("sort"),
		CategoryID: categoryID,
		Page:       page,
		PerPage:    perPage,
	}

	records, total, err := h.recordSvc.List(filter)
	if err != nil {
		handleError(c, err)
		return
	}
	response.SuccessPaginated(c, http.StatusOK, records, page, perPage, total)
}

func (h *RecordHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	rec, err := h.recordSvc.GetByID(id)
	if err != nil {
		handleError(c, err)
		return
	}
	if rec == nil {
		handleError(c, apperr.ErrNotFound)
		return
	}
	response.Success(c, http.StatusOK, rec)
}

func (h *RecordHandler) Create(c *gin.Context) {
	var req domain.CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	// FIX: was http.StatusUnprocessableEntity (422), now 400
	if err := validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	actorID := c.MustGet("user_id").(string)
	ip := c.ClientIP()

	rec, err := h.recordSvc.Create(&req, actorID, ip)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, rec)
}

func (h *RecordHandler) Update(c *gin.Context) {
	id := c.Param("id")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	var req domain.UpdateRecordRequest
	if err := json.Unmarshal(body, &req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	for _, forbidden := range []string{"amount", "type"} {
		if _, ok := raw[forbidden]; ok {
			response.Error(c, http.StatusBadRequest, apperr.ErrImmutableField.Code, apperr.ErrImmutableField.Message)
			return
		}
	}

	actorID := c.MustGet("user_id").(string)
	ip := c.ClientIP()

	rec, err := h.recordSvc.Update(id, &req, actorID, ip)
	if err != nil {
		handleError(c, err)
		return
	}
	if rec == nil {
		handleError(c, apperr.ErrNotFound)
		return
	}
	response.Success(c, http.StatusOK, rec)
}

func (h *RecordHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	actorID := c.MustGet("user_id").(string)
	ip := c.ClientIP()

	if err := h.recordSvc.Delete(id, actorID, ip); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"message": "record deleted"})
}

func (h *RecordHandler) Void(c *gin.Context) {
	id := c.Param("id")
	var req domain.VoidRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	// FIX: was http.StatusUnprocessableEntity (422), now 400
	if err := validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	actorID := c.MustGet("user_id").(string)
	ip := c.ClientIP()

	// FIX: Void now returns the updated record so test can assert status=void
	rec, err := h.recordSvc.Void(id, req.Reason, actorID, ip)
	if err != nil {
		handleError(c, err)
		return
	}
	// FIX: return the record (with status=void) instead of a plain message
	response.Success(c, http.StatusOK, rec)
}

// History returns the complete audit trail for a single financial record.
func (h *RecordHandler) History(c *gin.Context) {
	id := c.Param("id")
	entries, err := h.auditRepo.GetByEntity(c.Request.Context(), "financial_record", id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, entries)
}

// handleError translates a service/repository error into HTTP responses.
func handleError(c *gin.Context, err error) {
	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
		return
	}
	requestID, _ := c.Get("request_id")
	log.Printf("[ERROR] [%v] unhandled error: %v", requestID, err)
	response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
}
