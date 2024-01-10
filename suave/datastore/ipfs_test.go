package datastore_test

import (
	"testing"
)

func TestIPFS(t *testing.T) {
	// t.Parallel()

	// ipfs := datastore.IPFS{}
	// data := []byte("hello world")

	// require.Equal(t, "http://localhost:5001/api/v0", ipfs.String())

	// t.Run("Put", func(t *testing.T) {
	// 	req, err := ipfs.Put(context.TODO(), bytes.NewReader(data))
	// 	require.NoError(t, err)
	// 	require.NotNil(t, req)

	// 	t.Log(req.URL.String())

	// 	res, err := http.DefaultClient.Do(req)
	// 	bb, _ := io.ReadAll(res.Body)
	// 	t.Log(string(bb))

	// 	require.NoError(t, err)
	// 	require.Equal(t, http.StatusOK, res.StatusCode)

	// 	b, err := io.ReadAll(res.Body)
	// 	require.NoError(t, err)
	// 	_, cid, err := cid.CidFromBytes(b)
	// 	require.NoError(t, err)
	// 	require.Equal(t, "QmT78zSuBmuS4z925WZfrqQ1qHaJ56DQaTfyMUF7F8ff5o", cid.String())
	// })

	// t.Run("Get", func(t *testing.T) {
	// 	cid := cid.MustParse("QmT78zSuBmuS4z925WZfrqQ1qHaJ56DQaTfyMUF7F8ff5o")

	// 	req, err := ipfs.Get(context.TODO(), cid)
	// 	require.NoError(t, err)
	// 	require.NotNil(t, req)

	// 	res, err := http.DefaultClient.Do(req)
	// 	require.NoError(t, err)

	// 	b, err := io.ReadAll(res.Body)
	// 	require.NoError(t, err)
	// 	t.Log(string(b))
	// 	t.Fail()
	// })
}
