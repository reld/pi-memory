package api

import "encoding/json"

type Request struct {
	Version int             `json:"version"`
	Command string          `json:"command"`
	Payload json.RawMessage `json:"payload"`
}

type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type Response struct {
	OK     bool       `json:"ok"`
	Result any        `json:"result,omitempty"`
	Error  *ErrorBody `json:"error,omitempty"`
}

func Success(result any) Response {
	return Response{OK: true, Result: result}
}

func Failure(code, message string, details map[string]any) Response {
	return Response{OK: false, Error: &ErrorBody{Code: code, Message: message, Details: details}}
}
