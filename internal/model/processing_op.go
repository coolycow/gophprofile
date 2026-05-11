package model

// ProcessingOp операция обработки аватара
type ProcessingOp struct {
	Operation string         `json:"operation"`
	Params    map[string]any `json:"params"`
}
