package http

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
)

const (
	cookieName = "refres_session"
	ctxUser    = "user"
)

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if len(id) == 0 || len(id) > 64 {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		c.Set("rid", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func accessLog(l *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.FullPath() == "/healthz" {
			return
		}
		l.Info("http", "rid", c.GetString("rid"), "method", c.Request.Method, "route", c.FullPath(), "status", c.Writer.Status(),
			"dur_ms", time.Since(start).Milliseconds(), "ip", c.ClientIP())
	}
}

func recovery(l *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		l.Error("panic", "rid", c.GetString("rid"), "err", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal", "message": "Erro interno."}})
	})
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cache-Control", "no-store")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		c.Next()
	}
}

func bodyLimit(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		c.Next()
	}
}

// csrfGuard rejects state-changing requests that are not JSON or come from a foreign origin.
// Combined with the SameSite=Strict session cookie this blocks cross-site form posts.
func csrfGuard(allowed map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		if o := c.GetHeader("Origin"); o != "" && !allowed[o] {
			abort(c, http.StatusForbidden, "bad_origin", "Origem não permitida.")
			return
		}
		if c.GetHeader("Sec-Fetch-Site") == "cross-site" {
			abort(c, http.StatusForbidden, "bad_origin", "Origem não permitida.")
			return
		}
		if c.Request.ContentLength != 0 && !strings.HasPrefix(c.ContentType(), "application/json") {
			abort(c, http.StatusUnsupportedMediaType, "json_required", "Usa application/json.")
			return
		}
		c.Next()
	}
}

func cors(allowed map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		o := c.GetHeader("Origin")
		if o != "" && allowed[o] {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", o)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
			h.Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID, Idempotency-Key")
			h.Set("Access-Control-Max-Age", "600")
			h.Add("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

type limiter struct {
	mu    sync.Mutex
	r     rate.Limit
	burst int
	m     map[string]*entry
}

type entry struct {
	l    *rate.Limiter
	seen time.Time
}

func newLimiter(perMinute float64, burst int) *limiter {
	l := &limiter{r: rate.Limit(perMinute / 60), burst: burst, m: map[string]*entry{}}
	go func() {
		for range time.Tick(5 * time.Minute) {
			l.mu.Lock()
			for k, e := range l.m {
				if time.Since(e.seen) > 10*time.Minute {
					delete(l.m, k)
				}
			}
			l.mu.Unlock()
		}
	}()
	return l
}

func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.m[key]
	if !ok {
		e = &entry{l: rate.NewLimiter(l.r, l.burst)}
		l.m[key] = e
	}
	e.seen = time.Now()
	return e.l.Allow()
}

func (l *limiter) mw(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if p := net.ParseIP(ip); p == nil {
			ip = "unknown"
		}
		if !l.allow(scope + "|" + ip) {
			c.Header("Retry-After", "60")
			abort(c, http.StatusTooManyRequests, "rate_limited", "Demasiados pedidos. Tenta daqui a pouco.")
			return
		}
		c.Next()
	}
}

func abort(c *gin.Context, status int, code, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": msg}})
}

func (s *Server) fail(c *gin.Context, err error) {
	if e, ok := d.AsError(err); ok {
		status := map[d.Kind]int{
			d.KindInvalid: http.StatusUnprocessableEntity, d.KindNotFound: http.StatusNotFound, d.KindConflict: http.StatusConflict,
			d.KindUnauthorized: http.StatusUnauthorized, d.KindForbidden: http.StatusForbidden, d.KindUnavailable: http.StatusServiceUnavailable,
			d.KindTooMany: http.StatusTooManyRequests,
		}[e.Kind]
		if status == 0 {
			status = http.StatusBadRequest
		}
		abort(c, status, e.Code, e.Message)
		return
	}
	var mbe *http.MaxBytesError
	if ok := asMaxBytes(err, &mbe); ok {
		abort(c, http.StatusRequestEntityTooLarge, "too_large", "Pedido demasiado grande.")
		return
	}
	s.log.Error("request failed", "rid", c.GetString("rid"), "route", c.FullPath(), "err", err)
	abort(c, http.StatusInternalServerError, "internal", "Erro interno. Tenta de novo.")
}

func (s *Server) auth(c *gin.Context) {
	tok, err := c.Cookie(cookieName)
	if err != nil || tok == "" {
		abort(c, http.StatusUnauthorized, "unauthorized", "Entra para continuar.")
		return
	}
	u, err := s.app.Authenticate(c.Request.Context(), tok)
	if err != nil {
		s.clearCookie(c)
		abort(c, http.StatusUnauthorized, "unauthorized", "Sessão inválida ou expirada.")
		return
	}
	c.Set(ctxUser, u)
	c.Next()
}

func (s *Server) superAdmin(c *gin.Context) {
	if u := user(c); u == nil || u.Role != d.RoleSuperAdmin {
		abort(c, http.StatusForbidden, "forbidden", "Sem permissão.")
		return
	}
	c.Next()
}

func user(c *gin.Context) *d.User {
	v, _ := c.Get(ctxUser)
	u, _ := v.(*d.User)
	return u
}
