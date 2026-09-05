package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"featureflag/internal/db"
	"featureflag/internal/service"
	"featureflag/internal/store"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Svc *service.Service
}

// NewRouter 注入 service（可为 nil，此时仅 /healthz，供单测）。
func NewRouter(svc *service.Service) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())
	r.GET("/healthz", Healthz)

	if svc != nil {
		h := &Handler{Svc: svc}
		api := r.Group("/api/v1")
		{
			api.GET("/flags", h.ListFlags)
			api.POST("/flags", h.CreateFlag)
			api.GET("/flags/:id", h.GetFlag)
			api.PATCH("/flags/:id", h.UpdateFlag)
			api.POST("/flags/:id/enable", h.EnableFlag)
			api.POST("/flags/:id/disable", h.DisableFlag)
			api.GET("/flags/:id/history", h.ListHistory)
			api.POST("/flags/:id/rules", h.CreateRule)
			api.PATCH("/flags/:id/rules/:ruleId", h.UpdateRule)
			api.DELETE("/flags/:id/rules/:ruleId", h.DeleteRule)
		}
	}
	return r
}

func Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeError(c *gin.Context, err error) {
	body := errorBody{}
	switch {
	case errors.Is(err, db.ErrFlagKeyConflict):
		body.Error.Code = "KEY_CONFLICT"
		body.Error.Message = "该环境下 Key 已存在"
		c.JSON(http.StatusConflict, body)
	case errors.Is(err, db.ErrRulePriorityConflict):
		body.Error.Code = "PRIORITY_CONFLICT"
		body.Error.Message = "同一 Flag 内优先级不可重复（数字越小优先级越高）"
		c.JSON(http.StatusBadRequest, body)
	case errors.Is(err, service.ErrNotFound):
		body.Error.Code = "NOT_FOUND"
		body.Error.Message = "资源不存在"
		c.JSON(http.StatusNotFound, body)
	case errors.Is(err, service.ErrInvalidInput):
		body.Error.Code = "INVALID_INPUT"
		body.Error.Message = err.Error()
		c.JSON(http.StatusBadRequest, body)
	default:
		body.Error.Code = "INTERNAL_ERROR"
		body.Error.Message = err.Error()
		c.JSON(http.StatusInternalServerError, body)
	}
}

func (h *Handler) ListFlags(c *gin.Context) {
	filter := store.FlagFilter{
		Key:         strings.TrimSpace(c.Query("key")),
		Environment: strings.TrimSpace(c.Query("environment")),
	}
	if v := c.Query("enabled"); v != "" {
		switch v {
		case "true", "1":
			b := true
			filter.Enabled = &b
		case "false", "0":
			b := false
			filter.Enabled = &b
		default:
			writeError(c, errors.Join(service.ErrInvalidInput, errors.New("enabled 必须是 true 或 false")))
			return
		}
	}
	flags, err := h.Svc.ListFlags(c.Request.Context(), filter)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": flags})
}

func (h *Handler) CreateFlag(c *gin.Context) {
	var in service.CreateFlagInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeError(c, errors.Join(service.ErrInvalidInput, err))
		return
	}
	f, err := h.Svc.CreateFlag(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"flag": f})
}

func (h *Handler) GetFlag(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, errors.Join(service.ErrInvalidInput, errors.New("invalid id")))
		return
	}
	detail, err := h.Svc.GetFlagDetail(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *Handler) UpdateFlag(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, errors.Join(service.ErrInvalidInput, errors.New("invalid id")))
		return
	}
	var in service.UpdateFlagInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeError(c, errors.Join(service.ErrInvalidInput, err))
		return
	}
	f, err := h.Svc.UpdateFlag(c.Request.Context(), id, in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"flag": f})
}

func (h *Handler) EnableFlag(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, errors.Join(service.ErrInvalidInput, errors.New("invalid id")))
		return
	}
	f, err := h.Svc.EnableFlag(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"flag": f})
}

func (h *Handler) DisableFlag(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, errors.Join(service.ErrInvalidInput, errors.New("invalid id")))
		return
	}
	f, err := h.Svc.DisableFlag(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"flag": f})
}

func (h *Handler) ListHistory(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, errors.Join(service.ErrInvalidInput, errors.New("invalid id")))
		return
	}
	items, err := h.Svc.ListHistory(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) CreateRule(c *gin.Context) {
	flagID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, fmt.Errorf("%w: invalid id", service.ErrInvalidInput))
		return
	}
	var in service.RuleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeError(c, fmt.Errorf("%w: %v", service.ErrInvalidInput, err))
		return
	}
	r, err := h.Svc.CreateRule(c.Request.Context(), flagID, in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"rule": r})
}

func (h *Handler) UpdateRule(c *gin.Context) {
	flagID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, fmt.Errorf("%w: invalid id", service.ErrInvalidInput))
		return
	}
	ruleID, err := strconv.ParseInt(c.Param("ruleId"), 10, 64)
	if err != nil {
		writeError(c, fmt.Errorf("%w: invalid ruleId", service.ErrInvalidInput))
		return
	}
	var in service.RuleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeError(c, fmt.Errorf("%w: %v", service.ErrInvalidInput, err))
		return
	}
	r, err := h.Svc.UpdateRule(c.Request.Context(), flagID, ruleID, in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rule": r})
}

func (h *Handler) DeleteRule(c *gin.Context) {
	flagID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, fmt.Errorf("%w: invalid id", service.ErrInvalidInput))
		return
	}
	ruleID, err := strconv.ParseInt(c.Param("ruleId"), 10, 64)
	if err != nil {
		writeError(c, fmt.Errorf("%w: invalid ruleId", service.ErrInvalidInput))
		return
	}
	if err := h.Svc.DeleteRule(c.Request.Context(), flagID, ruleID); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
