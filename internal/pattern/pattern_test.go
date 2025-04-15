package pattern

import (
	"regexp"
	"testing"
)

func TestPattern_Match(t *testing.T) {
	tests := []struct {
		name       string
		patternStr string
		urlPath    []byte
		want       bool
	}{
		{
			name:       "regular:positive",
			patternStr: "/app/health",
			urlPath:    []byte("/app/health"),
			want:       true,
		},
		{
			name:       "regular:negative_1",
			patternStr: "/app/health",
			urlPath:    []byte("/app/health/"),
			want:       false,
		},
		{
			name:       "regular:negative_2",
			patternStr: "/user/do",
			urlPath:    []byte("/user/do/something"),
			want:       false,
		},
		{
			name:       "regular:negative_3",
			patternStr: "/app/health",
			urlPath:    []byte("/user/do/something"),
			want:       false,
		},
		{
			name:       "contains any:positive",
			patternStr: "/user/*/pay",
			urlPath:    []byte("/user/123456789/pay"),
			want:       true,
		},
		{
			name:       "contains any:negative_1",
			patternStr: "/user/*/pay",
			urlPath:    []byte("/user/pay"),
			want:       false,
		},
		{
			name:       "contains any:negative_2",
			patternStr: "/user/*/pay",
			urlPath:    []byte("/user/123456789/payment"),
			want:       false,
		},
		{
			name:       "contains any:negative_3",
			patternStr: "/user/*/pay",
			urlPath:    []byte("/user/123456789/pay/do"),
			want:       false,
		},
		{
			name:       "contains 3 any:positive",
			patternStr: "/merchant/*/user/*/transaction/*",
			urlPath:    []byte("/merchant/01/user/123456789/transaction/abcd-1234-efjk-5678"),
			want:       true,
		},
		{
			name:       "contains 3 any:negative",
			patternStr: "/merchant/*/user/*/transaction/*",
			urlPath:    []byte("/merchant/01/user/123456789/transaction/abcd-1234-efjk-5678/do"),
			want:       false,
		},
		{
			name:       "contains end:positive",
			patternStr: "/user/pay$",
			urlPath:    []byte("/user/pay"),
			want:       true,
		},
		{
			name:       "contains end:negative_1",
			patternStr: "/user/pay$",
			urlPath:    []byte("/user/pay/"),
			want:       false,
		},
		{
			name:       "contains end:negative_2",
			patternStr: "/user/pay$",
			urlPath:    []byte("/user/payment"),
			want:       false,
		},
		{
			name:       "contains 3 any and end:positive",
			patternStr: "/merchant/*/user/*/transaction/*/pay$",
			urlPath:    []byte("/merchant/01/user/123456789/transaction/abcd-1234-efjk-5678/pay"),
			want:       true,
		},
		{
			name:       "contains 3 any and end:negative",
			patternStr: "/merchant/*/user/*/transaction/*/pay$",
			urlPath:    []byte("/merchant/01/user/123456789/transaction/abcd-1234-efjk-5678/payment"),
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPattern(tt.patternStr)

			if got := p.Match(tt.urlPath); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

//go test -bench=. -benchmem ./internal/pattern
//goos: darwin
//goarch: arm64
//pkg: github.com/wbpaygate/traefik-ratelimit/internal/pattern
//cpu: Apple M3 Pro
//BenchmarkPatternMatch/StaticPath_Custom-12              57322784                20.21 ns/op            0 B/op          0 allocs/op
//BenchmarkPatternMatch/StaticPath_Regexp-12              32098612                36.91 ns/op            0 B/op          0 allocs/op
//BenchmarkPatternMatch/WildcardPath_Custom-12            49257882                24.12 ns/op            0 B/op          0 allocs/op
//BenchmarkPatternMatch/WildcardPath_Regexp-12            11216604               106.0 ns/op             0 B/op          0 allocs/op
//BenchmarkPatternMatch/ComplexPath_Custom-12             42511861                27.09 ns/op            0 B/op          0 allocs/op
//BenchmarkPatternMatch/ComplexPath_Regexp-12              8754301               135.2 ns/op             0 B/op          0 allocs/op
//BenchmarkPatternMatch/EndAnchor_Custom-12               58530517                20.47 ns/op            0 B/op          0 allocs/op
//BenchmarkPatternMatch/EndAnchor_Regexp-12               10437505               112.8 ns/op             0 B/op          0 allocs/op
//PASS
//ok      github.com/wbpaygate/traefik-ratelimit/internal/pattern 10.130s

func BenchmarkPatternMatch(b *testing.B) {
	benchmarks := []struct {
		name      string
		pattern   string
		urlPath   string
		useRegexp bool
	}{
		{"StaticPath_Custom", "/user/profile", "/user/profile", false},
		{"StaticPath_Regexp", "^/user/profile$", "/user/profile", true},
		{"WildcardPath_Custom", "/user/*/profile", "/user/123/profile", false},
		{"WildcardPath_Regexp", "^/user/[^/]+/profile$", "/user/123/profile", true},
		{"ComplexPath_Custom", "/api/*/resource/*", "/api/v1/resource/123", false},
		{"ComplexPath_Regexp", "^/api/[^/]+/resource/[^/]+$", "/api/v1/resource/123", true},
		{"EndAnchor_Custom", "/images/*$", "/images/photo.jpg", false},
		{"EndAnchor_Regexp", "^/images/.+$", "/images/photo.jpg", true},
	}

	customPatterns := make(map[string]*Pattern)
	regexPatterns := make(map[string]*regexp.Regexp)

	for _, bb := range benchmarks {
		if !bb.useRegexp {
			customPatterns[bb.name] = NewPattern(bb.pattern)
		} else {
			regexPatterns[bb.name] = regexp.MustCompile(bb.pattern)
		}
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			urlBytes := []byte(bb.urlPath)

			if bb.useRegexp {
				re := regexPatterns[bb.name]
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					re.Match(urlBytes)
				}

			} else {
				p := customPatterns[bb.name]
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					p.Match(urlBytes)
				}
			}
		})
	}
}
