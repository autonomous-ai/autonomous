package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// hardwareProxy is a wildcard reverse proxy from /api/hardware/* to HAL on
// loopback (127.0.0.1:5001). It exists so the web UI never touches /hw/*
// directly — adminAuthMiddleware gates the bearer here, and the upstream
// HAL call is loopback so its local_only_middleware lets it through.
//
// MJPEG streams (/api/hardware/camera/stream) work because
// httputil.ReverseProxy disables response buffering for chunked / multipart
// content out of the box. Long-running endpoints reuse the default 300s
// proxy timeout configured at the http.Server level.
// openapiProxy serves the in-iframe Swagger UI's `/openapi.json` fetch by
// forwarding straight to HAL on loopback. Path stays as-is — FastAPI
// generates the spec at /openapi.json on HAL, no rewrite needed.
var openapiProxy = func() http.Handler {
	target, _ := url.Parse("http://127.0.0.1:5001")
	proxy := httputil.NewSingleHostReverseProxy(target)
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		req.Header.Del("X-Forwarded-For")
		req.Header.Del("X-Real-IP")
	}
	return proxy
}()

var hardwareProxy = func() http.Handler {
	target, _ := url.Parse("http://127.0.0.1:5001")
	proxy := httputil.NewSingleHostReverseProxy(target)
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// Gin's wildcard match leaves /api/hardware/<path> in req.URL.Path.
		// Strip the prefix so HAL sees its original path.
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/hardware")
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		origDirector(req)
		// Stop leaking the original LAN client IP downstream: HAL's
		// same-origin/local check trusts loopback, so we present as one.
		req.Header.Del("X-Forwarded-For")
		req.Header.Del("X-Real-IP")
	}
	return proxy
}()
