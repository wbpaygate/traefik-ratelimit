package traefik_ratelimit

import (
	"github.com/wbpaygate/traefik-ratelimit/internal/pattern"
)

type Rule struct {
	URLPathPattern string `json:"urlpathpattern"`
	HeaderKey      string `json:"headerkey"`
	HeaderVal      string `json:"headerval"`
}

type Limit struct {
	Limit int    `json:"limit"`
	Rules []Rule `json:"rules"`
}

type Limits struct {
	Limits []Limit `json:"limits"`
}

type Header struct {
	key string
	val string
}

func (h *Header) String() string {
	return h.key + ": " + h.val
}

type RuleImpl struct {
	URLPathPattern *pattern.Pattern
	Header         *Header
}

func (ri *RuleImpl) String() string {
	if ri.Header != nil {
		return "[" + ri.URLPathPattern.String() + ", " + ri.Header.String() + "]"

	}

	return "[" + ri.URLPathPattern.String() + "]"
}
