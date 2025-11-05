package server

type assemblyDryRunRequest struct {
	Plan map[string]any `json:"plan" binding:"required"`
}

type assemblySavePlanRequest struct {
	Name    string          `json:"name" binding:"required"`
	Include map[string]bool `json:"include"`
}

type assemblyCooldownClearRequest struct {
	Credentials []string `json:"credentials"`
	All         bool     `json:"all"`
}
