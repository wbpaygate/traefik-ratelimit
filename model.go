package traefik_ratelimit

import (
	"fmt"
	"strings"

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

func (l *Limits) validate() error {
	if l == nil {
		return fmt.Errorf("limits is nil")
	}

	if len(l.Limits) == 0 {
		return fmt.Errorf("limits are required")
	}

	var errorMessages []string

	for i, lim := range l.Limits {
		if lim.Limit <= 0 {
			errorMessages = append(errorMessages, fmt.Sprintf("[limit %d]: limit value <= 0", i))
		}

		if len(lim.Rules) == 0 {
			errorMessages = append(errorMessages, fmt.Sprintf("[limit %d]: no rules specified", i))
			continue
		}

		for j, rule := range lim.Rules {
			rulePrefix := fmt.Sprintf("[limit %d, rule %d]", i, j)

			if rule.HeaderKey != "" && rule.HeaderVal == "" {
				errorMessages = append(errorMessages,
					fmt.Sprintf("%s: header key '%s' provided but header value is empty",
						rulePrefix, rule.HeaderKey))
			}

			if rule.HeaderVal != "" && rule.HeaderKey == "" {
				errorMessages = append(errorMessages,
					fmt.Sprintf("%s: header value provided but header key is empty",
						rulePrefix))
			}

			if rule.HeaderVal == "" && rule.HeaderKey == "" && rule.URLPathPattern == "" {
				errorMessages = append(errorMessages,
					fmt.Sprintf("%s: rule is empty - must specify either header or URL pattern",
						rulePrefix))
			}
		}
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("errors: %s", strings.Join(errorMessages, ", "))
	}

	return nil
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
