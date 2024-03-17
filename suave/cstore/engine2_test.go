package cstore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/stretchr/testify/require"
)

func TestAPI_NewSession(t *testing.T) {
	engine := NewNonAuthenticatedEngine(NewLocalConfidentialStore())
	require.NoError(t, engine.Start())

	clt := apiClient{url: "http://localhost:8080"}
	id := clt.newSession(t)

	dataid := clt.newRecord(t, id, "hello")

	val := "value"
	clt.postRecordKey(t, id, dataid, "key", "value")
	require.Equal(t, val, clt.getRecordKey(t, id, dataid, "key"))

	record := clt.getRecord(t, id, dataid)
	require.Equal(t, hex.EncodeToString(record.Id[:]), dataid)

	records := clt.fetchByBlock(t, id, 0, "hello")
	require.Len(t, records, 1)
	require.Equal(t, hex.EncodeToString(record.Id[:]), hex.EncodeToString(records[0].Id[:]))

	// after finalize the data should be available
	clt.finalize(t, id)

	{
		id2 := clt.newSession(t)
		require.Equal(t, val, clt.getRecordKey(t, id2, dataid, "key"))
		record = clt.getRecord(t, id2, dataid)
		require.Equal(t, hex.EncodeToString(record.Id[:]), dataid)
		records = clt.fetchByBlock(t, id2, 0, "hello")
		require.Len(t, records, 1)
	}
}

type apiClient struct {
	url string
}

func (a *apiClient) newSession(t *testing.T) string {
	return string(a.query(t, http.MethodPost, "/cstore/new"))
}

func (a *apiClient) newRecord(t *testing.T, id string, namespace string) string {
	return string(a.query(t, http.MethodPost, fmt.Sprintf("/cstore/%s/record?namespace=%s", id, namespace)))
}

func (a *apiClient) getRecord(t *testing.T, id, dataId string) (record suave.DataRecord) {
	data := a.query(t, http.MethodGet, fmt.Sprintf("/cstore/%s/record/%s", id, dataId))
	require.NoError(t, json.Unmarshal(data, &record))
	return
}

func (a *apiClient) fetchByBlock(t *testing.T, id string, block uint64, namespace string) (records []suave.DataRecord) {
	data := a.query(t, http.MethodGet, fmt.Sprintf("/cstore/%s/fetchByBlock?block=%d&namespace=%s", id, block, namespace))
	require.NoError(t, json.Unmarshal(data, &records))
	return
}

func (a *apiClient) getRecordKey(t *testing.T, id, dataId, key string) string {
	return string(a.query(t, http.MethodGet, fmt.Sprintf("/cstore/%s/record/%s/%s", id, dataId, key)))
}

func (a *apiClient) postRecordKey(t *testing.T, id, dataId, key, value string) string {
	return string(a.query(t, http.MethodPost, fmt.Sprintf("/cstore/%s/record/%s/%s?value=%s", id, dataId, key, value)))
}

func (a *apiClient) finalize(t *testing.T, id string) {
	a.query(t, http.MethodPost, fmt.Sprintf("/cstore/%s/finalize", id))
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
