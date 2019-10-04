// +build linux,!cgo

package main

import (
	"fmt"
	"os/exec"
)

func (c *Context) CheckUser(username string) (Check, error) {
	return func() (Status, string) {
		cmd := exec.Command("/bin/id", username)
		err := cmd.Run()
		if err != nil {
			return Critical, fmt.Sprintf("User information for %s could not be retrieved: %s", username, err.Error())
		}

		return OK, ""
	}, nil

}
