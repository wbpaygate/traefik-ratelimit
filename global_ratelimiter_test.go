package traefik_ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wbpaygate/traefik-ratelimit/internal/keeper"
)

type testCase struct {
	name            string
	config          *Config
	keeperResponse  string
	setup           func(t *testing.T, config *Config, keeperSrv *httptest.Server) *httptest.Server
	requestPath     string
	expectedAllowed int
	waitBeforeTest  time.Duration
}

func runRateLimiterTests(t *testing.T, tests []testCase) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			rl, err := New(t.Context(), h, tt.config, "test")
			if err != nil {
				t.Fatalf("cannot create new TraefikRateLimiter: %v", err)
			}

			testSrv := httptest.NewServer(rl)
			defer testSrv.Close()

			var keeperSrv *httptest.Server
			if tt.setup != nil {
				keeperSrv = tt.setup(t, tt.config, keeperSrv)
				if keeperSrv != nil {
					defer keeperSrv.Close()
				}
			}

			if keeperSrv != nil {
				keeperClient := keeper.NewTestClient(keeperSrv.Client(), keeperSrv.URL)
				globalRateLimiter.Configure(t.Context(), tt.config, keeperClient)
			} else {
				globalRateLimiter.Configure(t.Context(), tt.config, nil)
			}

			if tt.waitBeforeTest > 0 {
				time.Sleep(tt.waitBeforeTest)
			}

			testClient := testSrv.Client()
			var allowedReqs int

			const requestCount = 100
			for i := 0; i < requestCount; i++ {
				req, err := http.NewRequest(http.MethodGet, testSrv.URL+tt.requestPath, http.NoBody)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}

				resp, err := testClient.Do(req)
				if err != nil {
					t.Fatalf("request %d failed: %v", i, err)
				}
				resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					allowedReqs++
				}
			}

			if tt.expectedAllowed >= 0 && allowedReqs > tt.expectedAllowed {
				t.Errorf("%d requests allowed, expected no more than %d", allowedReqs, tt.expectedAllowed)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	tests := []testCase{
		{
			name: "Default config should limit to 3 requests",
			config: &Config{
				RatelimitData:  `{"limits":[{"limit":3,"rules":[{"urlpathpattern":"/whoami"}]}]}`,
				RatelimitDebug: "true",
			},
			requestPath:     "/whoami",
			expectedAllowed: 3,
		},
		{
			name: "Empty config should not limit",
			config: &Config{
				RatelimitDebug: "true",
			},
			requestPath:     "",
			expectedAllowed: -1, // -1 имеется ввиду нет проверки
		},
		{
			name: "Should load limits from keeper",
			config: &Config{
				KeeperURL:            "", // будет установлено в setup
				KeeperReloadInterval: "2s",
				RatelimitDebug:       "true",
			},
			keeperResponse: `{"limits":[{"limit":7,"rules":[{"urlpathpattern":"/whoami"}]}]}`,
			setup: func(t *testing.T, config *Config, _ *httptest.Server) *httptest.Server {
				srv := keeper.NewTestServer(`{"limits":[{"limit":7,"rules":[{"urlpathpattern":"/whoami"}]}]}`)
				config.KeeperURL = srv.URL
				return srv
			},
			requestPath:     "/whoami",
			expectedAllowed: 7,
			waitBeforeTest:  3 * time.Second,
		},
	}

	runRateLimiterTests(t, tests)
}
