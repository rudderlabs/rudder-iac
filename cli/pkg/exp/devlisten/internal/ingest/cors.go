package ingest

import "net/http"

// CORS copies rs/cors v1.11.1 with the SVC options (middleware_cors.go):
// any origin reflected, credentials allowed, any request header, max age 900 s,
// the default methods GET, POST and HEAD.
const (
	corsMaxAge       = "900"
	corsPreflightVar = "Origin, Access-Control-Request-Method, Access-Control-Request-Headers"
)

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

// preflight answers 204 before auth, as rs/cors does with OptionsPassthrough off.
func preflight(r *http.Request) reply {
	rep := reply{kind: "control", transport: "http", status: http.StatusNoContent, header: http.Header{}}
	rep.header["Vary"] = []string{corsPreflightVar}

	origin := r.Header.Get("Origin")
	if origin == "" || !corsMethodAllowed(r.Header.Get("Access-Control-Request-Method")) {
		return rep
	}
	rep.header["Access-Control-Allow-Origin"] = r.Header["Origin"]
	rep.header["Access-Control-Allow-Methods"] = r.Header["Access-Control-Request-Method"]
	if reqHeaders, ok := r.Header["Access-Control-Request-Headers"]; ok && len(reqHeaders[0]) > 0 {
		rep.header["Access-Control-Allow-Headers"] = reqHeaders
	}
	rep.header.Set("Access-Control-Allow-Credentials", "true")
	rep.header.Set("Access-Control-Max-Age", corsMaxAge)
	return rep
}

// withActualCORS adds the headers rs/cors puts on a non-preflight response.
func withActualCORS(r *http.Request, h http.Header) {
	h.Add("Vary", "Origin")
	if r.Header.Get("Origin") == "" || !corsMethodAllowed(r.Method) {
		return
	}
	h["Access-Control-Allow-Origin"] = r.Header["Origin"]
	h.Set("Access-Control-Allow-Credentials", "true")
}
