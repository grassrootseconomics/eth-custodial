package api

import "errors"

var (
	errInvalidJSON = errors.New("Invalid JSON")
)

type H map[string]any

type okResp struct {
	Ok          bool   `json:"ok"`
	Description string `json:"description"`
	Result      H      `json:"result"`
}

type errResp struct {
	Ok          bool   `json:"ok"`
	Description string `json:"description"`
}
