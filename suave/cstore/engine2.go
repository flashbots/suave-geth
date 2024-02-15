package cstore

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/google/uuid"

	"github.com/gorilla/mux"
)

// NonAuthenticatedEngine is a confidential store engine that does not do any validation
// of the caller. The validation must be done on the application layer.
// It is accessed via and HTTP API.
type NonAuthenticatedEngine struct {
	storage ConfidentialStorageBackend
	server  *http.Server

	sessions     map[string]*TransactionalStore2
	sessionsLock sync.Mutex
}

func NewNonAuthenticatedEngine(storage ConfidentialStorageBackend) *NonAuthenticatedEngine {
	a := &NonAuthenticatedEngine{
		storage:  storage,
		sessions: make(map[string]*TransactionalStore2),
	}

	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not found", http.StatusNotFound)
	})

	r.HandleFunc("/cstore/new", a.handleNewSession).Methods("POST")
	r.HandleFunc("/cstore/{id}/record", a.handleNewRecord).Methods("POST")
	r.HandleFunc("/cstore/{id}/record/{dataid}/{key}", a.handleRecordStorePost).Methods("POST")
	r.HandleFunc("/cstore/{id}/record/{dataid}/{key}", a.handleRecordStoreGet).Methods("GET")
	r.HandleFunc("/cstore/{id}/record/{dataid}", a.handleRecordGet).Methods("GET")
	r.HandleFunc("/cstore/{id}/fetchByBlock", a.handleFetchByBlock).Methods("GET")
	r.HandleFunc("/cstore/{id}/finalize", a.handleFinalizeSession).Methods("POST")

	a.server = &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	return a
}

func (a *NonAuthenticatedEngine) Start() error {
	log.Info("Server started on :8080")

	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Error starting server: %s\n", err)
		}
	}()

	return nil
}

func (a *NonAuthenticatedEngine) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.server.Shutdown(ctx); err != nil {
		return err
	}
	log.Info("Server shut down gracefully")
	return nil
}

func (a *NonAuthenticatedEngine) finalize(store *TransactionalStore2) error {
	for _, record := range store.pendingRecords {
		err := a.storage.InitRecord(record)
		if err != nil {
			// TODO: deinitialize!
			return fmt.Errorf("confidential engine: store backend failed to initialize record: %w", err)
		}
	}

	for _, sw := range store.pendingWrites {
		if _, err := a.storage.Store(sw.DataRecord, sw.Caller, sw.Key, sw.Value); err != nil {
			// TODO: deinitialize and deStore!
			return fmt.Errorf("failed to store data: %w", err)
		}
	}

	return nil
}

func (a *NonAuthenticatedEngine) getSession(id string) *TransactionalStore2 {
	a.sessionsLock.Lock()
	defer a.sessionsLock.Unlock()
	return a.sessions[id]
}

func (a *NonAuthenticatedEngine) handleNewSession(w http.ResponseWriter, r *http.Request) {
	id := uuid.New().String()[:7]
	store := NewTransactionalStore2(a.storage)

	a.sessionsLock.Lock()
	a.sessions[id] = store
	a.sessionsLock.Unlock()

	fmt.Fprint(w, id)
}

func (a *NonAuthenticatedEngine) handleNewRecord(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	session := a.getSession(id)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	// decode namespace from query. If not found, return 404
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		http.Error(w, "namespace not found", http.StatusNotFound)
		return
	}

	record := types.DataRecord{
		Salt:                suave.RandomDataRecordId(),
		DecryptionCondition: 0,
		Version:             namespace,
	}

	// decode decrypt condition from query. If not found, default to 0
	decryptConditionStr := r.URL.Query().Get("decryptCondition")
	if decryptConditionStr != "" {
		decryptCondition, err := strconv.ParseUint(decryptConditionStr, 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to parse decrypt condition: %v", err), http.StatusBadRequest)
			return
		}
		record.DecryptionCondition = decryptCondition
	}

	record, err := session.InitRecord(record)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to init record: %v", err), http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "%s", hex.EncodeToString(record.Id[:]))
}

func (a *NonAuthenticatedEngine) handleRecordStorePost(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	key := mux.Vars(r)["key"]
	dataId := decodeDataId(r)

	session := a.getSession(id)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	value := []byte(r.URL.Query().Get("value"))
	session.Store(dataId, key, value)
}

func (a *NonAuthenticatedEngine) handleRecordStoreGet(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	key := mux.Vars(r)["key"]
	dataId := decodeDataId(r)

	session := a.getSession(id)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	val, err := session.Retrieve(dataId, key)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to retrieve: %v", err), http.StatusBadRequest)
		return
	}

	w.Write(val)
}

func (a *NonAuthenticatedEngine) handleRecordGet(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	dataId := decodeDataId(r)

	session := a.getSession(id)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	record, err := session.FetchRecordByID(dataId)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch record: %v", err), http.StatusBadRequest)
		return
	}

	recordJson, err := json.Marshal(record)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal record: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(recordJson)
}

func (a *NonAuthenticatedEngine) handleFetchByBlock(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	session := a.getSession(id)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	// decode namespace from query. If not found, return 404
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		http.Error(w, "namespace not found", http.StatusNotFound)
		return
	}

	var decryptCondition uint64

	// decode decrypt condition from query. If not found, default to 0
	decryptConditionStr := r.URL.Query().Get("decryptCondition")
	if decryptConditionStr != "" {
		num, err := strconv.ParseUint(decryptConditionStr, 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to parse decrypt condition: %v", err), http.StatusBadRequest)
			return
		}
		decryptCondition = num
	}

	records := session.FetchRecordsByProtocolAndBlock(decryptCondition, namespace)

	recordsJson, err := json.Marshal(records)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal records: %v", err), http.StatusInternalServerError)
		return
	}
	w.Write(recordsJson)
}

func (a *NonAuthenticatedEngine) handleFinalizeSession(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	session := a.getSession(id)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	if err := a.finalize(session); err != nil {
		http.Error(w, fmt.Sprintf("failed to finalize: %v", err), http.StatusInternalServerError)
		return
	}
}

func decodeDataId(r *http.Request) types.DataId {
	dataIdStr := mux.Vars(r)["dataid"]

	buf, err := hex.DecodeString(dataIdStr)
	if err != nil {
		panic(err)
	}

	var dataId types.DataId
	copy(dataId[:], buf)
	return dataId
}
