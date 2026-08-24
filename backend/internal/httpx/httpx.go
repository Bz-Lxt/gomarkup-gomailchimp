package httpx

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lumen/relay/internal/auth"
	"github.com/lumen/relay/internal/clock"
	"github.com/lumen/relay/internal/domain"
)

type Envelope struct {
	Data  any        `json:"data,omitempty"`
	Meta  any        `json:"meta,omitempty"`
	Error *APIError  `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Data: data})
}

func Page(c *gin.Context, data any, total int64, page, per int) {
	c.JSON(http.StatusOK, Envelope{Data: data, Meta: gin.H{"total": total, "page": page, "per_page": per}})
}

func Fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, Envelope{Error: &APIError{Code: "not_found", Message: "资源不存在"}})
	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, Envelope{Error: &APIError{Code: "unauthorized", Message: "未登录或令牌无效"}})
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, Envelope{Error: &APIError{Code: "forbidden", Message: "权限不足"}})
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, Envelope{Error: &APIError{Code: "conflict", Message: err.Error()}})
	case errors.Is(err, domain.ErrValidation), errors.Is(err, domain.ErrInvalidState):
		c.JSON(http.StatusUnprocessableEntity, Envelope{Error: &APIError{Code: "validation_error", Message: err.Error()}})
	case errors.Is(err, domain.ErrQuotaExceeded):
		c.JSON(http.StatusTooManyRequests, Envelope{Error: &APIError{Code: "quota_exceeded", Message: err.Error()}})
	default:
		c.JSON(http.StatusInternalServerError, Envelope{Error: &APIError{Code: "internal_error", Message: "内部错误"}})
	}
}

func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("trace_id", id)
		c.Writer.Header().Set("X-Request-ID", id)
		c.Next()
	}
}

func AccessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("http",
			"trace_id", c.GetString("trace_id"),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"ms", time.Since(start).Milliseconds(),
			"at", clock.Format(clock.Now()),
		)
	}
}

func JWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		raw := ""
		if strings.HasPrefix(h, "Bearer ") {
			raw = strings.TrimPrefix(h, "Bearer ")
		} else if q := c.Query("access_token"); q != "" {
			raw = q
		}
		if raw == "" {
			Fail(c, domain.ErrUnauthorized)
			c.Abort()
			return
		}
		cl, kind, err := auth.Parse(secret, raw)
		if err != nil || kind != "access" {
			Fail(c, domain.ErrUnauthorized)
			c.Abort()
			return
		}
		c.Set("claims", cl)
		c.Next()
	}
}

func Claims(c *gin.Context) domain.Claims {
	v, _ := c.Get("claims")
	cl, _ := v.(domain.Claims)
	return cl
}

func RequireWrite() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !Claims(c).CanWrite() {
			Fail(c, domain.ErrForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}
