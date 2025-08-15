package control

type Message struct {
	TenantID string `json:"tenant_id"`
	Workers  int32  `json:"workers,omitempty"`
}
