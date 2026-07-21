package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes caps the request body size for state-changing requests. It wraps
// the body in an http.MaxBytesReader so that any handler reading it (gin's
// ShouldBind, manual io.ReadAll, etc.) receives an error once the limit is
// exceeded, which the existing bind-failure path reports as a 400 rather than
// allocating an unbounded buffer or starting a long DB transaction.
//
// Methods without a body (GET/HEAD/OPTIONS/TRACE) and a non-positive limit are
// passed through untouched. Paths ending in one of skipSuffixes are also passed
// through uncapped — these are routes that legitimately accept a large upload
// (e.g. database restore, which streams a multi-MiB SQLite file).
func MaxBodyBytes(limit int64, skipSuffixes ...string) gin.HandlerFunc {
	overrides := make(map[string]int64, len(skipSuffixes))
	for _, suffix := range skipSuffixes {
		overrides[suffix] = 0
	}
	return MaxBodyBytesBySuffix(limit, overrides)
}

func MaxBodyBytesBySuffix(defaultLimit int64, overrides map[string]int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := defaultLimit
		for suffix, override := range overrides {
			if matchesPathSuffix(c.Request.URL.Path, suffix) {
				limit = override
				break
			}
		}
		if limit > 0 {
			switch c.Request.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			default:
				if c.Request.ContentLength > limit {
					c.AbortWithStatus(http.StatusRequestEntityTooLarge)
					return
				}
				if c.Request.Body != nil {
					c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
				}
			}
		}
		c.Next()
	}
}

func matchesPathSuffix(path, pattern string) bool {
	star := strings.IndexByte(pattern, '*')
	if star < 0 {
		return pattern != "" && strings.HasSuffix(path, pattern)
	}
	prefix := pattern[:star]
	suffix := pattern[star+1:]
	if !strings.HasSuffix(path, suffix) {
		return false
	}
	end := len(path) - len(suffix)
	start := strings.LastIndex(path[:end], prefix)
	if start < 0 {
		return false
	}
	middle := path[start+len(prefix) : end]
	return middle != "" && !strings.Contains(middle, "/")
}
