// +build linux,cgo

package main

import (
	"fmt"
	"os/user"
)

func (c *Context) CheckUser(username string) (Check, error) {
	return func() (Status, string) {
		_, err := user.Lookup(username)
		if err != nil {
			return Critical, fmt.Sprintf("User information for %s could not be retrieved: %s", username, err.Error())
		}
		return OK, ""
	}, nil
}
