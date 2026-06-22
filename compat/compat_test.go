package compat

import (
	"strings"
	"testing"
)

func TestOpenAPI_RemovedEndpointSurfaces(t *testing.T) {
	old := []byte(`{"openapi":"3.0.0","info":{},"paths":{
		"/pets":{"get":{"responses":{"200":{"description":"ok"}}}},
		"/users":{"get":{"responses":{"200":{"description":"ok"}}}}
	}}`)
	new_ := []byte(`{"openapi":"3.0.0","info":{},"paths":{
		"/pets":{"get":{"responses":{"200":{"description":"ok"}}}}
	}}`)
	regs, err := OpenAPI(old, new_)
	if err != nil {
		t.Fatal(err)
	}
	if len(regs) != 1 {
		t.Fatalf("expected 1 regression; got %d (%+v)", len(regs), regs)
	}
	if regs[0].Kind != "endpoint-removed" || !strings.Contains(regs[0].Detail, "/users") {
		t.Errorf("expected endpoint-removed for /users; got %+v", regs[0])
	}
}

func TestOpenAPI_Removed2xxStatusSurfaces(t *testing.T) {
	old := []byte(`{"openapi":"3.0.0","info":{},"paths":{
		"/pets":{"get":{"responses":{"200":{"description":"ok"},"201":{"description":"created"}}}}
	}}`)
	new_ := []byte(`{"openapi":"3.0.0","info":{},"paths":{
		"/pets":{"get":{"responses":{"200":{"description":"ok"}}}}
	}}`)
	regs, err := OpenAPI(old, new_)
	if err != nil {
		t.Fatal(err)
	}
	if len(regs) != 1 || regs[0].Kind != "status-removed" {
		t.Fatalf("expected status-removed; got %+v", regs)
	}
}

func TestProto_RemovedRPCAndStreamingChange(t *testing.T) {
	old := []byte(`service Pets {
  rpc GetPet(Req) returns (Resp);
  rpc ListPets(Req) returns (stream Resp);
}`)
	new_ := []byte(`service Pets {
  rpc ListPets(Req) returns (Resp);
}`)
	regs, err := Proto(old, new_)
	if err != nil {
		t.Fatal(err)
	}
	if len(regs) != 2 {
		t.Fatalf("expected 2 regressions; got %d (%+v)", len(regs), regs)
	}
	kinds := map[string]bool{}
	for _, r := range regs {
		kinds[r.Kind] = true
	}
	if !kinds["rpc-removed"] {
		t.Errorf("missing rpc-removed regression: %+v", regs)
	}
	if !kinds["streaming-shape-changed"] {
		t.Errorf("missing streaming-shape-changed regression: %+v", regs)
	}
}

func TestAsyncAPI_RemovedChannel(t *testing.T) {
	old := []byte(`{"asyncapi":"2.6.0","info":{},"channels":{
		"orders/created":{"publish":{"operationId":"a","message":{"name":"O"}}},
		"orders/cancelled":{"publish":{"operationId":"b","message":{"name":"O"}}}
	}}`)
	new_ := []byte(`{"asyncapi":"2.6.0","info":{},"channels":{
		"orders/created":{"publish":{"operationId":"a","message":{"name":"O"}}}
	}}`)
	regs, err := AsyncAPI(old, new_)
	if err != nil {
		t.Fatal(err)
	}
	if len(regs) != 1 || regs[0].Kind != "channel-removed" {
		t.Fatalf("expected channel-removed; got %+v", regs)
	}
}

func TestOpenAPI_NoRegressionsWhenSchemasEqual(t *testing.T) {
	body := []byte(`{"openapi":"3.0.0","info":{},"paths":{
		"/pets":{"get":{"responses":{"200":{"description":"ok"}}}}
	}}`)
	regs, err := OpenAPI(body, body)
	if err != nil {
		t.Fatal(err)
	}
	if len(regs) != 0 {
		t.Errorf("expected no regressions on identical schemas; got %+v", regs)
	}
}
