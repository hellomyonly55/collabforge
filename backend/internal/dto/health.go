package dto

type Health struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}
