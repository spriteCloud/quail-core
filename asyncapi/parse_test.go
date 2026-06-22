package asyncapi

import "testing"

func TestParse_FlattensChannelsAndDirections(t *testing.T) {
	body := []byte(`{
		"asyncapi": "2.6.0",
		"info": {"title": "X", "version": "1.0"},
		"channels": {
			"orders/created": {
				"publish": {"operationId": "publishOrderCreated", "message": {"name": "OrderCreated"}}
			},
			"orders/cancelled": {
				"subscribe": {"operationId": "onOrderCancelled", "message": {"name": "OrderCancelled"}}
			}
		}
	}`)
	_, channels, err := Parse(body)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(channels) != 2 {
		t.Fatalf("expected 2 channels; got %d (%+v)", len(channels), channels)
	}
}

func TestParse_RejectsNonAsyncAPI(t *testing.T) {
	if _, _, err := Parse([]byte(`{"foo":"bar"}`)); err == nil {
		t.Error("expected error on missing asyncapi: field")
	}
}
