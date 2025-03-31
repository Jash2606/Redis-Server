package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"Key_Value_Cache_Ass/models"
)

var CacheInstance = models.NewCache()

var (
	putRequestPool = sync.Pool{
		New: func() interface{} {
			return &PutRequest{}
		},
	}
	responsePool = sync.Pool{
		New: func() interface{} {
			return &Response{}
		},
	}
	bufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
)

var (
	successPutResponseBytes = []byte(`{"status":"OK","message":"Key inserted/updated successfully."}`)
	keyNotFoundResponseBytes = []byte(`{"status":"ERROR","message":"Key not found."}`)
	methodNotAllowedResponseBytes = []byte(`{"status":"ERROR","message":"Method Not Allowed"}`)
	invalidJSONResponseBytes = []byte(`{"status":"ERROR","message":"Invalid JSON"}`)
	invalidKeyValueLengthResponseBytes = []byte(`{"status":"ERROR","message":"Key and Value must be at most 256 characters"}`)
	missingKeyResponseBytes = []byte(`{"status":"ERROR","message":"Missing key parameter"}`)
	tooManyRequestsResponseBytes = []byte(`{"status":"ERROR","message":"Too many requests"}`)
	timeoutResponseBytes = []byte(`{"status":"ERROR","message":"Request timeout"}`)
)

type PutRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
}

var (
	maxConcurrent = 1000
	semaphore     = make(chan struct{}, maxConcurrent)
)

func writeJSONResponse(w http.ResponseWriter, statusCode int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(statusCode)
	w.Write(body)
}

func writeResponse(w http.ResponseWriter, statusCode int, resp *Response) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(statusCode)
	
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)
	
	if err := json.NewEncoder(buf).Encode(resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	
	w.Write(buf.Bytes())
}

func PutCache(w http.ResponseWriter, r *http.Request) {
	select {
	case semaphore <- struct{}{}:
		defer func() { <-semaphore }()
	default:
		writeJSONResponse(w, http.StatusTooManyRequests, tooManyRequestsResponseBytes)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, methodNotAllowedResponseBytes)
		return
	}

	req := putRequestPool.Get().(*PutRequest)
	defer putRequestPool.Put(req)
	*req = PutRequest{}

	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	_, err := io.Copy(buf, r.Body)
	if err != nil {
		writeJSONResponse(w, http.StatusBadRequest, invalidJSONResponseBytes)
		return
	}

	if err := json.Unmarshal(buf.Bytes(), req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, invalidJSONResponseBytes)
		return
	}

	if len(req.Key) > 256 || len(req.Value) > 256 {
		writeJSONResponse(w, http.StatusBadRequest, invalidKeyValueLengthResponseBytes)
		return
	}

	select {
	case <-ctx.Done():
		writeJSONResponse(w, http.StatusRequestTimeout, timeoutResponseBytes)
		return
	default:
		CacheInstance.Put(req.Key, req.Value)
		writeJSONResponse(w, http.StatusOK, successPutResponseBytes)
	}
}

func GetCache(w http.ResponseWriter, r *http.Request) {
	select {
	case semaphore <- struct{}{}:
		defer func() { <-semaphore }()
	default:
		writeJSONResponse(w, http.StatusTooManyRequests, tooManyRequestsResponseBytes)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	key := r.URL.Query().Get("key")
	if key == "" {
		writeJSONResponse(w, http.StatusBadRequest, missingKeyResponseBytes)
		return
	}

	select {
	case <-ctx.Done():
		writeJSONResponse(w, http.StatusRequestTimeout, timeoutResponseBytes)
		return
	default:
		val, found := CacheInstance.Get(key)
		if !found {
			writeJSONResponse(w, http.StatusNotFound, keyNotFoundResponseBytes)
			return
		}

		resp := responsePool.Get().(*Response)
		defer responsePool.Put(resp)
		
		*resp = Response{
			Status: "OK",
			Key:    key,
			Value:  val,
		}

		writeResponse(w, http.StatusOK, resp)
	}
}
