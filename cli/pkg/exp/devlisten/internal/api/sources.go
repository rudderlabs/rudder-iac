package api

import (
	"encoding/json"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

// SDK families of summary.bySource.bySdk.
const (
	sdkBrowser = "browser"
	sdkNode    = "node"
	sdkGo      = "go"
	sdkMobile  = "mobile"
	sdkOther   = "other"
)

// sourceCounts tells browser traffic from server traffic: an app whose
// browser SDK never loaded still shows its backend events.
type sourceCounts struct {
	ByChannel map[string]int        `json:"byChannel"`
	BySdk     map[string]*sdkCounts `json:"bySdk"`
}

type sdkCounts struct {
	Requests int `json:"requests"`
	Events   int `json:"events"`
	Control  int `json:"control"`
}

func newSourceCounts() sourceCounts {
	return sourceCounts{ByChannel: map[string]int{}, BySdk: map[string]*sdkCounts{}}
}

func (s sourceCounts) add(rec store.Record) {
	family := sdkFamily(rec)
	c := s.BySdk[family]
	if c == nil {
		c = &sdkCounts{}
		s.BySdk[family] = c
	}
	if rec.Kind == "control" {
		c.Control++
		return
	}
	c.Requests++
	c.Events += len(rec.Events)
	for _, ev := range rec.Events {
		if ch := eventSource(ev.Message).Channel; ch != "" {
			s.ByChannel[ch]++
		}
	}
}

// sdkRequests counts ingestion requests from server and mobile SDKs; a
// curl or a probe is "other" and does not count.
func (s sourceCounts) sdkRequests() int {
	n := 0
	for _, family := range []string{sdkNode, sdkGo, sdkMobile} {
		if c := s.BySdk[family]; c != nil {
			n += c.Requests
		}
	}
	return n
}

func (s sourceCounts) browser() sdkCounts {
	if c := s.BySdk[sdkBrowser]; c != nil {
		return *c
	}
	return sdkCounts{}
}

type messageSource struct {
	Channel string `json:"channel"`
	Context struct {
		Library struct {
			Name string `json:"name"`
		} `json:"library"`
	} `json:"context"`
}

func eventSource(msg json.RawMessage) messageSource {
	var m messageSource
	_ = json.Unmarshal(msg, &m)
	return m
}

// sdkFamily reads context.library.name of the first event, then the
// User-Agent header.
func sdkFamily(rec store.Record) string {
	if len(rec.Events) > 0 {
		if f := libraryFamily(eventSource(rec.Events[0].Message).Context.Library.Name); f != "" {
			return f
		}
	}
	return userAgentFamily(rec.Request.Headers.Get("User-Agent"))
}

var libraryFamilies = []struct{ substr, family string }{
	{"javascript", sdkBrowser}, {"analytics-js", sdkBrowser},
	{"node", sdkNode},
	{"-go", sdkGo},
	{"android", sdkMobile}, {"ios", sdkMobile}, {"flutter", sdkMobile}, {"react-native", sdkMobile},
}

func libraryFamily(name string) string {
	name = strings.ToLower(name)
	if name == "" {
		return ""
	}
	for _, lf := range libraryFamilies {
		if strings.Contains(name, lf.substr) {
			return lf.family
		}
	}
	return sdkOther
}

var userAgentFamilies = []struct{ substr, family string }{
	{"mozilla/", sdkBrowser},
	{"node", sdkNode}, {"axios", sdkNode},
	{"go-http-client", sdkGo},
	{"okhttp", sdkMobile}, {"cfnetwork", sdkMobile}, {"dart", sdkMobile},
}

func userAgentFamily(ua string) string {
	ua = strings.ToLower(ua)
	for _, uf := range userAgentFamilies {
		if strings.Contains(ua, uf.substr) {
			return uf.family
		}
	}
	return sdkOther
}
