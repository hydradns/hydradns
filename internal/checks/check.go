package checks

import (
	"context"
	"net"
)

type Status string

const (
	StatusPass  Status = "pass"
	StatusFail  Status = "fail"
	StatusSkip  Status = "skip"
	StatusError Status = "error"
)

type Resolver struct {
	IP   net.IP
	Port int
}

type Check interface {
	Name() string
	Run(ctx context.Context, resolver Resolver) (Result, error)
}

type Result struct {
	Status   Status                 `json:"status"`
	Reason   string                 `json:"reason,omitempty"`
	Evidence map[string]interface{} `json:"evidence,omitempty"`
}
