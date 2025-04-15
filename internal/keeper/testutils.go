package keeper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

func NewTestServer(limitsConfig string) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp, err := json.Marshal(&Value{
			Value:       limitsConfig,
			Version:     100,
			ModRevision: 200,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	}))
	return srv
}

func NewTestClient(cl *http.Client, url string) *KeeperClient {
	return NewKeeperClient(cl, url, "", "")
}
