package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/logger"
	"github.com/gin-gonic/gin"
)

// RequestGzip — middleware-логер для входящих HTTP-запросов.
func RequestGzip() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Распаковываем данные.
		if strings.Contains(c.GetHeader("Content-Encoding"), "gzip") {
			logger.Log.Info("request with gzip")

			compressedData, _ := io.ReadAll(c.Request.Body)
			reader, err := gzip.NewReader(bytes.NewReader(compressedData))

			if err != nil {
				_ = c.Error(error.CustomError{
					Message:    "Failed to decompress gzip data",
					StatusCode: http.StatusBadRequest,
				})
				c.Abort()
			}

			defer func(reader *gzip.Reader) {
				if reader != nil {
					err = reader.Close()
					if err != nil {
						logger.Log.Error(err.Error())
					}
				}
			}(reader)

			if reader != nil {
				c.Request.Body = io.NopCloser(reader)
			}
		}

		// Передаём управление другим Middleware или Handler
		c.Next()
	}
}
