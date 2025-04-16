package traefik_ratelimit

type Rule struct {
	UrlPathPattern string `json:"urlpathpattern"`
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
	return h.key + "_" + h.val
}
