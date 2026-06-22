// Package asyncapi parses an AsyncAPI 2.x document enough to surface
// the channels × operations × message payloads a contract test cares
// about. Intentionally narrow — schema validation lives elsewhere.
package asyncapi

import (
	"encoding/json"
	"fmt"
)

type Doc struct {
	AsyncAPI string                 `json:"asyncapi"`
	Info     Info                   `json:"info"`
	Channels map[string]ChannelItem `json:"channels"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type ChannelItem struct {
	Publish   *Op `json:"publish,omitempty"`
	Subscribe *Op `json:"subscribe,omitempty"`
}

type Op struct {
	OperationID string     `json:"operationId"`
	Message     RawMessage `json:"message"`
}

// RawMessage is the message reference / inlined shape.
type RawMessage struct {
	Name    string            `json:"name"`
	Payload map[string]any    `json:"payload"`
}

// Channel is the flattened (channel, direction, operationId) shape
// the compat layer + tests render against.
type Channel struct {
	Path        string
	Direction   string // publish | subscribe
	OperationID string
	MessageName string
}

func Parse(body []byte) (*Doc, []Channel, error) {
	var d Doc
	if err := json.Unmarshal(body, &d); err != nil {
		return nil, nil, fmt.Errorf("asyncapi: parse: %w", err)
	}
	if d.AsyncAPI == "" {
		return &d, nil, fmt.Errorf("asyncapi: not an AsyncAPI document (missing `asyncapi:` field)")
	}
	var channels []Channel
	for p, c := range d.Channels {
		if c.Publish != nil {
			channels = append(channels, Channel{Path: p, Direction: "publish", OperationID: c.Publish.OperationID, MessageName: c.Publish.Message.Name})
		}
		if c.Subscribe != nil {
			channels = append(channels, Channel{Path: p, Direction: "subscribe", OperationID: c.Subscribe.OperationID, MessageName: c.Subscribe.Message.Name})
		}
	}
	return &d, channels, nil
}
