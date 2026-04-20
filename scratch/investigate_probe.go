package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"github.com/SurgeDM/Surge/internal/processing"
	"github.com/SurgeDM/Surge/internal/config"
)

func main() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Probe request: %s %s Range: %s\n", r.Method, r.URL, r.Header.Get("Range"))
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	ctx := context.Background()
	runCfg := config.DefaultSettings().ToRuntimeConfig()
	
	fmt.Println("Probing server that returns 403...")
	result, err := processing.ProbeServerWithProxy(ctx, server.URL, "test.bin", nil, runCfg)
	
	if err != nil {
		fmt.Printf("Probe failed as expected: %v\n", err)
	} else {
		fmt.Printf("Probe succeeded surprisingly! Result: %+v\n", result)
	}
    
    if result == nil {
        fmt.Println("Result is nil on error.")
    }
}
