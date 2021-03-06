package middleware

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/labstack/echo"
	"github.com/labstack/echo/engine"
)

type (
	// GzipConfig defines config for gzip middleware.
	GzipConfig struct {
		// Level is the gzip level.
		Level int
	}

	gzipResponseWriter struct {
		engine.Response
		io.Writer
	}
)

var (
	// DefaultGzipConfig is the default gzip middleware config.
	DefaultGzipConfig = GzipConfig{
		Level: gzip.DefaultCompression,
	}
)

// Gzip returns a middleware which compresses HTTP response using gzip compression
// scheme.
func Gzip() echo.MiddlewareFunc {
	return GzipFromConfig(DefaultGzipConfig)
}

// GzipFromConfig return gzip middleware from config.
// See `Gzip()`.
func GzipFromConfig(config GzipConfig) echo.MiddlewareFunc {
	pool := gzipPool(config)
	scheme := "gzip"

	return func(next echo.Handler) echo.Handler {
		return echo.HandlerFunc(func(c echo.Context) error {
			c.Response().Header().Add(echo.Vary, echo.AcceptEncoding)
			if strings.Contains(c.Request().Header().Get(echo.AcceptEncoding), scheme) {
				w := pool.Get().(*gzip.Writer)
				w.Reset(c.Response().Writer())
				defer func() {
					w.Close()
					pool.Put(w)
					w.Close()
				}()
				g := gzipResponseWriter{Response: c.Response(), Writer: w}
				c.Response().Header().Set(echo.ContentEncoding, scheme)
				c.Response().SetWriter(g)
			}
			return next.Handle(c)
		})
	}
}

func (g gzipResponseWriter) Write(b []byte) (int, error) {
	if g.Header().Get(echo.ContentType) == "" {
		g.Header().Set(echo.ContentType, http.DetectContentType(b))
	}
	return g.Writer.Write(b)
}

func gzipPool(config GzipConfig) sync.Pool {
	return sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(ioutil.Discard, config.Level)
			return w
		},
	}
}
