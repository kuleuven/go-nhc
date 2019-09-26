package main

import (
	"encoding/json"
	"fmt"
	"net"
)

type SensuClient struct {
	Address string
}

type SensuResult struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
	Output string `json:"output"`
}

func NewSensuClient(address string) *SensuClient {
	return &SensuClient{
		Address: address,
	}
}

func (s *SensuClient) SendResult(checkName string, status Status, message string) error {
	if status == OK && message == "" {
		message = "Check returned successfully"
	}
	result := &SensuResult{
		Name:   fmt.Sprintf("nhc_%s", checkName),
		Status: status.RC(),
		Output: fmt.Sprintf("%s: %s\n", status.String(), message),
	}
	return s.Send(result)
}

func (s *SensuClient) Send(r *SensuResult) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}

	conn, err := net.Dial("tcp", s.Address)
	defer conn.Close()

	fmt.Fprintln(conn, string(b))
	return nil
}
