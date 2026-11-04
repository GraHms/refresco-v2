package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCSRFGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(csrfGuard(map[string]bool{"https://refres.co": true}))
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	cases := []struct {
		name, origin, site, ctype string
		want                      int
	}{
		{"same origin json", "https://refres.co", "same-origin", "application/json", 204},
		{"foreign origin", "https://evil.example", "cross-site", "application/json", 403},
		{"cross-site without origin", "", "cross-site", "application/json", 403},
		{"form post", "https://refres.co", "same-origin", "application/x-www-form-urlencoded", 415},
		{"non-browser client", "", "", "application/json", 204},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", c.ctype)
		if c.origin != "" {
			req.Header.Set("Origin", c.origin)
		}
		if c.site != "" {
			req.Header.Set("Sec-Fetch-Site", c.site)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != c.want {
			t.Errorf("%s: got %d, want %d", c.name, w.Code, c.want)
		}
	}
}
