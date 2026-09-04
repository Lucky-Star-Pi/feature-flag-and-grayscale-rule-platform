package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"featureflag/internal/service"
	"featureflag/internal/store"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Svc *service.Service
}

func NewRouter(svc *service.Service) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())
	h := &Handler{Svc: svc}

	api := r.Group("/api/v1")
	{
		api.GET("/flags", h.ListFlags)
		api.POST("/flags", h.CreateFlag)
		api.GET("/flags/:id", h.GetFlag)
		api.PUT("/flags/:id", h.UpdateFlag)
		api.PATCH("/flags/:id/enabled", h.SetEnabled)
		api.POST("/flags/:id/rules", h.CreateRule)
		api.PUT("/flags/:id/rules/:ruleId", h.UpdateRule)
		api.DELETE("/flags/:id/rules/:ruleId", h.DeleteRule)
		api.GET("/flags/:id/histories", h.ListHistories)
		api.POST("/evaluate", h.Evaluate)
	}
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

type errBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidJSON):
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_JSON", Message: "用户属性必须是 JSON 对象"})
	case errors.Is(err, service.ErrDuplicatePriority):
		c.JSON(http.StatusBadRequest, errBody{Code: "DUPLICATE_PRIORITY", Message: "同一 Flag 内优先级不可重复"})
	case errors.Is(err, service.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: err.Error()})
	case errors.Is(err, service.ErrFlagNotFound), errors.Is(err, store.ErrNotFound):
		c.JSON(http.StatusNotFound, errBody{Code: "FLAG_NOT_FOUND", Message: "指定环境未找到该 Flag Key"})
	case errors.Is(err, service.ErrRuleNotFound):
		c.JSON(http.StatusNotFound, errBody{Code: "RULE_NOT_FOUND", Message: "规则不存在"})
	case errors.Is(err, store.ErrFlagKeyConflict):
		c.JSON(http.StatusConflict, errBody{Code: "FLAG_KEY_CONFLICT", Message: "该环境下 Key 已存在"})
	case errors.Is(err, store.ErrRulePriorityConflict):
		c.JSON(http.StatusConflict, errBody{Code: "RULE_PRIORITY_CONFLICT", Message: "规则优先级冲突"})
	default:
		c.JSON(http.StatusInternalServerError, errBody{Code: "INTERNAL_ERROR", Message: err.Error()})
	}
}

func (h *Handler) ListFlags(c *gin.Context) {
	q := c.Query("q")
	env := c.Query("environment")
	var enabled *bool
	if v := c.Query("enabled"); v != "" {
		b := v == "true" || v == "1"
		enabled = &b
		if v != "true" && v != "false" && v != "1" && v != "0" {
			c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: "enabled must be true or false"})
			return
		}
		if v == "false" || v == "0" {
			f := false
			enabled = &f
		}
	}
	flags, err := h.Svc.ListFlags(c.Request.Context(), q, env, enabled)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": flags})
}

func (h *Handler) CreateFlag(c *gin.Context) {
	var in service.CreateFlagInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: err.Error()})
		return
	}
	f, err := h.Svc.CreateFlag(c.Request.Context(), in)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"flag": f})
}

func (h *Handler) GetFlag(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: "invalid id"})
		return
	}
	detail, err := h.Svc.GetDetail(c.Request.Context(), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *Handler) UpdateFlag(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: "invalid id"})
		return
	}
	var in service.UpdateFlagInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: err.Error()})
		return
	}
	f, err := h.Svc.UpdateFlag(c.Request.Context(), id, in)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"flag": f})
}

func (h *Handler) SetEnabled(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: "invalid id"})
		return
	}
	var in service.SetEnabledInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: err.Error()})
		return
	}
	f, err := h.Svc.SetEnabled(c.Request.Context(), id, in)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"flag": f})
}

func (h *Handler) CreateRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: "invalid id"})
		return
	}
	var in service.RuleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: err.Error()})
		return
	}
	r, err := h.Svc.CreateRule(c.Request.Context(), id, in)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"rule": r})
}

func (h *Handler) UpdateRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: "invalid id"})
		return
	}
	ruleID, err := strconv.ParseInt(c.Param("ruleId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: "invalid ruleId"})
		return
	}
	var in service.RuleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: err.Error()})
		return
	}
	r, err := h.Svc.UpdateRule(c.Request.Context(), id, ruleID, in)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rule": r})
}

func (h *Handler) DeleteRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: "invalid id"})
		return
	}
	ruleID, err := strconv.ParseInt(c.Param("ruleId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: "invalid ruleId"})
		return
	}
	if err := h.Svc.DeleteRule(c.Request.Context(), id, ruleID); err != nil {
		writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListHistories(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_INPUT", Message: "invalid id"})
		return
	}
	detail, err := h.Svc.GetDetail(c.Request.Context(), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": detail.Histories})
}

func (h *Handler) Evaluate(c *gin.Context) {
	raw, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_JSON", Message: "用户属性必须是 JSON 对象"})
		return
	}
	var envelope struct {
		Key         string          `json:"key"`
		Environment string          `json:"environment"`
		Attributes  json.RawMessage `json:"attributes"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_JSON", Message: "用户属性必须是 JSON 对象"})
		return
	}
	if len(envelope.Attributes) == 0 || strings.TrimSpace(string(envelope.Attributes)) == "null" {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_JSON", Message: "用户属性必须是 JSON 对象"})
		return
	}
	var attrs map[string]any
	dec := json.NewDecoder(strings.NewReader(string(envelope.Attributes)))
	dec.UseNumber()
	if err := dec.Decode(&attrs); err != nil || attrs == nil {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_JSON", Message: "用户属性必须是 JSON 对象"})
		return
	}
	// Reject JSON arrays disguised via RawMessage that decoded oddly — ensure object
	if envelope.Attributes[0] != '{' {
		c.JSON(http.StatusBadRequest, errBody{Code: "INVALID_JSON", Message: "用户属性必须是 JSON 对象"})
		return
	}

	out, err := h.Svc.Evaluate(c.Request.Context(), service.EvaluateInput{
		Key: envelope.Key, Environment: envelope.Environment, Attributes: attrs,
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}
