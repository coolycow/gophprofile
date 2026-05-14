package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Без trusted subnet — всегда 403; совпадающий IP в CIDR — проход в обработчик.
func TestTrustedSubnetInternalStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(TrustedSubnetInternalStats(nil))
	r.GET("/stats", func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stats", nil))
	assert.Equal(t, http.StatusForbidden, w.Code)

	_, ipnet, err := net.ParseCIDR("203.0.113.0/24")
	require.NoError(t, err)
	r2 := gin.New()
	r2.Use(TrustedSubnetInternalStats(ipnet))
	r2.GET("/stats", func(c *gin.Context) { c.Status(http.StatusTeapot) })
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/stats", nil)
	req2.Header.Set("X-Real-IP", "203.0.113.7")
	r2.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusTeapot, w2.Code)
}
