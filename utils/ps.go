// +build linux

package utils

import (
	"fmt"
	"io"
	"os"

	linuxproc "github.com/c9s/goprocinfo/linux"
)

const (
	proc_mount = "/proc"
)

func ListProcesses() ([]*linuxproc.ProcessStatus, error) {
	d, err := os.Open(proc_mount)
	if err != nil {
		return nil, err
	}
	defer d.Close()

	result := []*linuxproc.ProcessStatus{}

	for {
		fis, err := d.Readdir(10)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		for _, fi := range fis {
			// Only directories - pids are dirs
			if !fi.IsDir() {
				continue
			}

			// Only numeric
			name := fi.Name()
			if name[0] < '0' || name[0] > '9' {
				continue
			}

			// Ignore errors from this point on - process can be 'gone'
			stat, err := linuxproc.ReadProcessStatus(fmt.Sprintf("%s/%s/status", proc_mount, name))
			if err == nil {
				result = append(result, stat)
			}
		}
	}

	return result, nil
}
