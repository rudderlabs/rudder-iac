package ingest

import (
	"net/http"
)

// gwError is one error answer RudderStack gives. The messages come from
// rudder-server gateway/response/response.go.
type gwError struct {
	status  int
	message string
	stage   string
}

var (
	errQueryTooLarge         = gwError{http.StatusRequestEntityTooLarge, "Request Entity Too Large", "size"}
	errServiceUnavailable    = gwError{http.StatusServiceUnavailable, "Service Unavailable", "overload"}
	errUncompress            = gwError{http.StatusBadRequest, "failed to uncompress request body", "decode"}
	errNoWriteKeyInBasicAuth = gwError{http.StatusUnauthorized, "failed to read writekey from header", "auth"}
	errNoWriteKeyInQuery     = gwError{http.StatusUnauthorized, "failed to read writekey from query params", "auth"}
	errInvalidWriteKey       = gwError{http.StatusUnauthorized, "invalid write key", "auth"}
	errRequestBodyNil        = gwError{http.StatusBadRequest, "request body is nil", "body"}
	errRequestBodyReadFailed = gwError{http.StatusBadRequest, "failed to read body from request", "body"}
	errInvalidJSON           = gwError{http.StatusBadRequest, "invalid json", "parse"}
	errNotRudderEvent        = gwError{http.StatusBadRequest, "event is not a valid rudder event", "parse"}
	errEmptyBatch            = gwError{http.StatusBadRequest, "empty batch payload", "batch"}
	errNonIdentifiable       = gwError{http.StatusBadRequest, "request neither has anonymousId nor userId", "identity"}
	errUnknownPath           = gwError{http.StatusNotFound, "unknown path", "route"}
	errProxyDisabled         = gwError{http.StatusNotImplemented, "Proxy is disabled", "route"}

	// These are dev listen's own memory caps. The pixel answers replace the
	// GIF, so the sender sees why dev listen refused a pixel query.
	errRequestBodyTooLarge = gwError{http.StatusRequestEntityTooLarge, "request body exceeds 2,048,000 bytes", "size"}
	errTooManyEvents       = gwError{http.StatusRequestEntityTooLarge, "batch has more than 10,000 events", "size"}
	errPixelTooManyKeys    = gwError{http.StatusBadRequest, "pixel query has more than 1000 keys", "size"}
	errPixelKeyTooDeep     = gwError{http.StatusBadRequest, "pixel query key has more than 32 levels", "size"}
)

// pixelGIF is the transparent GIF RudderStack answers to a pixel request.
const pixelGIF = "\x47\x49\x46\x38\x39\x61\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\x00\x00\x00\x21\xF9\x04" +
	"\x01\x00\x00\x00\x00\x2C\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02\x44\x01\x00\x3B"

type reply struct {
	status      int
	body        []byte
	contentType string
	allow       string
	// isError writes the headers http.Error writes.
	isError   bool
	rejection *Rejection
}

func reject(e gwError) reply {
	return reply{
		status:    e.status,
		body:      []byte(e.message + "\n"),
		isError:   true,
		rejection: &Rejection{Stage: e.stage, Reason: e.message},
	}
}

// methodNotAllowed matches chi's default answer.
func methodNotAllowed(allow string) reply {
	return reply{
		status:    http.StatusMethodNotAllowed,
		allow:     allow,
		rejection: &Rejection{Stage: "route"},
	}
}

// respond sets every header net/http would add itself, so the capture holds
// the headers sent. A nil c answers without a capture.
func (h *Handler) respond(w http.ResponseWriter, c *Capture, rep reply) {
	header := w.Header()
	switch {
	case rep.isError:
		header.Del("Content-Length")
		header.Set("Content-Type", "text/plain; charset=utf-8")
		header.Set("X-Content-Type-Options", "nosniff")
	case rep.contentType != "":
		header.Set("Content-Type", rep.contentType)
	case len(rep.body) > 0:
		header.Set("Content-Type", http.DetectContentType(rep.body))
	}
	if rep.allow != "" {
		header.Set("Allow", rep.allow)
	}

	if c != nil {
		c.StatusCode = rep.status
		c.Header = header.Clone()
		c.ResponseBody = rep.body
		c.Rejection = rep.rejection
		h.sink.Capture(c)
	}

	w.WriteHeader(rep.status)
	_, _ = w.Write(rep.body)
}

// The CORS functions copy rs/cors v1.11.1 with the options that give
// RudderStack's CORS answers.

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
}

func corsMethodAllowed(method string) bool {
	switch method {
	case http.MethodOptions, http.MethodGet, http.MethodPost, http.MethodHead:
		return true
	}
	return false
}

// preflight answers before auth and before the concurrency limit, because a
// browser sends a preflight without credentials.
func preflight(h http.Header, r *http.Request) {
	h["Vary"] = []string{"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"}
	if r.Header.Get("Origin") == "" || !corsMethodAllowed(r.Header.Get("Access-Control-Request-Method")) {
		return
	}
	h["Access-Control-Allow-Origin"] = r.Header["Origin"]
	h["Access-Control-Allow-Methods"] = r.Header["Access-Control-Request-Method"]
	if reqHeaders, ok := r.Header["Access-Control-Request-Headers"]; ok && reqHeaders[0] != "" {
		h["Access-Control-Allow-Headers"] = reqHeaders
	}
	h.Set("Access-Control-Allow-Credentials", "true")
	h.Set("Access-Control-Max-Age", "900")
}

func actualCORS(h http.Header, r *http.Request) {
	h.Add("Vary", "Origin")
	if r.Header.Get("Origin") == "" || !corsMethodAllowed(r.Method) {
		return
	}
	h["Access-Control-Allow-Origin"] = r.Header["Origin"]
	h.Set("Access-Control-Allow-Credentials", "true")
}
