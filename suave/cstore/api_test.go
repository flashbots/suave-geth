package cstore

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAPI_NewSession(t *testing.T) {
	fakeDaSigner := FakeDASigner{localAddresses: []common.Address{{0x42}}}

	engine := NewEngine(NewLocalConfidentialStore(), MockTransport{}, fakeDaSigner, MockChainSigner{})
	require.NoError(t, engine.Start())

	clt := apiClient{url: "http://localhost:8080"}
	id := clt.newSession(t)

	dataid := clt.newRecord(t, id, "hello")
	fmt.Println(dataid)
}

type apiClient struct {
	url string
}

func (a *apiClient) put(t *testing.T, id string, key, value []byte) {
	a.query(t, http.MethodPut, fmt.Sprintf("/cstore/%s/put", id))
}

func (a *apiClient) newSession(t *testing.T) string {
	return string(a.query(t, http.MethodPost, "/cstore/new"))
}

func (a *apiClient) newRecord(t *testing.T, id string, namespace string) string {
	return string(a.query(t, http.MethodPost, fmt.Sprintf("/cstore/%s/record?namespace=%s", id, namespace)))
}

func (a *apiClient) getRecord(t *testing.T, id, dataId string) []byte {
	return a.query(t, http.MethodGet, fmt.Sprintf("/cstore/%s/record/%s", id, dataId))
}

func (a *apiClient) fetchByBlock(t *testing.T, id string, block uint64, namespace string) []byte {
	return a.query(t, http.MethodGet, fmt.Sprintf("/cstore/%s/fetchByBlock?block=%d&namespace=%s", id, block, namespace))
}

func (a *apiClient) getRecordKey(t *testing.T, id, dataId, key string) []byte {
	return a.query(t, http.MethodGet, fmt.Sprintf("/cstore/%s/record/%s/%s", id, dataId, key))
}

func (a *apiClient) putRecordKey(t *testing.T, id, dataId string) []byte {
	return a.query(t, http.MethodPost, fmt.Sprintf("/cstore/%s/record/%s", id, dataId))
}

func (a *apiClient) query(t *testing.T, method string, path string) []byte {
	req, err := http.NewRequest(method, a.url+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
