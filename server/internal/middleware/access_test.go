package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

// access_test.go covers RequirePathTenantMatch, the single exported
// helper in access.go: the URL :id on a /tenants/:id route must match
// the caller's active tenant context, with no per-user bypass.

// ---------- RequirePathTenantMatch ----------

func runPathTenantHandler(ctxTenantID uint64, urlPath string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// ErrorHandler renders c.Error() into the standard envelope and
	// sets the status from AppError.HTTPCode. Mounting it here mirrors
	// the production router so the tests assert what real clients see.
	r.Use(ErrorHandler())
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		if ctxTenantID != 0 {
			ctx = context.WithValue(ctx, types.TenantIDContextKey, ctxTenantID)
		}
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.GET("/tenants/:id", RequirePathTenantMatch(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, urlPath, nil)
	r.ServeHTTP(w, req)
	return w
}

func TestRequirePathTenantMatch_MatchAllows(t *testing.T) {
	w := runPathTenantHandler(7, "/tenants/7")
	if w.Code != http.StatusOK {
		t.Fatalf("matching :id must allow, got %d", w.Code)
	}
}

func TestRequirePathTenantMatch_MismatchRejects(t *testing.T) {
	w := runPathTenantHandler(7, "/tenants/9")
	if w.Code != http.StatusForbidden {
		t.Fatalf("mismatch must 403, got %d", w.Code)
	}
}

func TestRequirePathTenantMatch_MissingCtxTenantRejects(t *testing.T) {
	// If the auth middleware never set TenantIDContextKey we fail
	// closed — silently treating "no ctx" as a match would be a footgun
	// the next time someone refactors the auth chain.
	w := runPathTenantHandler(0, "/tenants/7")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing tenant ctx must 401, got %d", w.Code)
	}
}

func TestRequirePathTenantMatch_NonNumericIDRejects(t *testing.T) {
	w := runPathTenantHandler(7, "/tenants/abc")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("non-numeric :id must 400, got %d", w.Code)
	}
}

func TestRequirePathTenantMatch_ZeroIDRejects(t *testing.T) {
	w := runPathTenantHandler(7, "/tenants/0")
	if w.Code != http.StatusBadRequest {
		t.Fatalf(":id=0 must 400, got %d", w.Code)
	}
}

// TestAccessMiddleware_ResponseEnvelope pins the wire format produced
// by RequirePathTenantMatch when it rejects. It used to live in a
// handler that called c.Error(apperrors.NewForbiddenError(...)) and
// rendered through ErrorHandler — clients (frontend axios interceptor,
// Go SDK) key off `success` and `error.code`. If a future change drops
// c.Error in favour of a raw c.JSON, this test catches the regression.
func TestAccessMiddleware_ResponseEnvelope(t *testing.T) {
	t.Run("RequirePathTenantMatch_Mismatch", func(t *testing.T) {
		w := runPathTenantHandler(7, "/tenants/9")
		assertEnvelope(t, w, http.StatusForbidden, apperrors.ErrForbidden)
	})
	t.Run("RequirePathTenantMatch_NonNumeric", func(t *testing.T) {
		w := runPathTenantHandler(7, "/tenants/abc")
		// NewValidationError uses ErrValidation, not ErrBadRequest — both
		// surface as HTTP 400 but the code in the body is ErrValidation.
		assertEnvelope(t, w, http.StatusBadRequest, apperrors.ErrValidation)
	})
	t.Run("RequirePathTenantMatch_NoCtxTenant", func(t *testing.T) {
		w := runPathTenantHandler(0, "/tenants/7")
		assertEnvelope(t, w, http.StatusUnauthorized, apperrors.ErrUnauthorized)
	})
}

func assertEnvelope(t *testing.T, w *httptest.ResponseRecorder, wantStatus int, wantCode apperrors.ErrorCode) {
	t.Helper()
	if w.Code != wantStatus {
		t.Fatalf("status: got %d, want %d (body=%s)", w.Code, wantStatus, w.Body.String())
	}
	var body struct {
		Success bool `json:"success"`
		Error   struct {
			Code    apperrors.ErrorCode `json:"code"`
			Message string              `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not the standard envelope: %v (body=%s)", err, w.Body.String())
	}
	if body.Success {
		t.Fatalf("envelope success must be false on rejection, got body=%s", w.Body.String())
	}
	if body.Error.Code != wantCode {
		t.Fatalf("envelope error.code: got %d, want %d (body=%s)",
			body.Error.Code, wantCode, w.Body.String())
	}
	if body.Error.Message == "" {
		t.Fatalf("envelope error.message must be non-empty, got body=%s", w.Body.String())
	}
}
