package mowl

import "encoding/json"

type envelope struct {
	Data  json.RawMessage `json:"Data"`
	Error *apiError       `json:"Error"`
	Stack *string         `json:"Stack"`
}

type apiError struct {
	Message string `json:"Message"`
}
