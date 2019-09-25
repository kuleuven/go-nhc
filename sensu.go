package main

import (
	"encoding/json"
	"fmt"
	"net"
)

const (
	sensuAddr = "127.0.0.1:3030"
)

func SendSensuResult(checkName string, status Status, message string) error {
	if status == OK && message == "" {
		message = "Check returned successfully"
	}
	result := &SensuResult{
		Name:   fmt.Sprintf("nhc_%s", checkName),
		Status: status.RC(),
		Output: fmt.Sprintf("%s: %s\n", status.String(), message),
	}
	return result.Send()
}

type SensuResult struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
	Output string `json:"output"`
}

func (s *SensuResult) Send() error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	conn, err := net.Dial("tcp", sensuAddr)
	defer conn.Close()

	fmt.Fprintln(conn, string(b))
	return nil
}
