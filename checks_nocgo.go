// +build linux,!cgo

package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func (c *Context) CheckUnauthorized(argument string) (Check, error) {
	m := &UnauthorizedMetadata{}
	err := ParseMetadata(m, argument, "Scheduler")
	if err != nil {
		return nil, err
	}
	switch m.Scheduler {
	case "pbs":
	default:
		return nil, fmt.Errorf("Unknown job scheduler %s", m.Scheduler)
	}
	if m.MaxSystemUid == 0 {
		m.MaxSystemUid = 1000
	}

	return func() (Status, string) {
		if c.psInfo == nil {
			var err error
			c.psInfo, err = ListProcesses()
			if err != nil {
				return Unknown, fmt.Sprintf("Could not parse process info: %s", err.Error())
			}
		}
		if c.jobInfo == nil {
			var err error
			c.jobInfo, err = ListPBSJobs()
			if err != nil {
				return Unknown, fmt.Sprintf("Could not parse job info for %s: %s", argument, err.Error())
			}
		}

	OUTER:
		for _, pStatus := range c.psInfo {
			if pStatus.RealUid < m.MaxSystemUid {
				continue OUTER
			}

			// Check whether the uid corresponds with a job uid - usually the case
			for _, job := range c.jobInfo {
				if job.Uid == pStatus.RealUid {
					continue OUTER
				}
			}

			// Check whether the uid is allowed because the corresponding user is in the group corresponding with the job gid (cfr. pbs_inode)
			cmd := exec.Command("/bin/id", "-G", fmt.Sprintf("%d", pStatus.RealUid))
			groups, err := cmd.Output()
			if err != nil {
				return Critical, fmt.Sprintf("Process %d is runned by unknown uid %d: %s", pStatus.Pid, pStatus.RealUid, err.Error())
			}
			groupsString := strings.TrimRight(string(groups), "\n")
			groupInts := []uint64{}
			for _, group := range strings.Split(groupsString, " ") {
				groupInt, err := strconv.ParseUint(group, 10, 64)
				if err != nil {
					return Critical, fmt.Sprintf("Could not parse group ids for user %d: %s", pStatus.RealUid, err.Error())
				}
				groupInts = append(groupInts, groupInt)
			}
			for _, job := range c.jobInfo {
				for _, gid := range groupInts {
					if job.Gid == gid {
						continue OUTER
					}
				}
			}
			return Critical, fmt.Sprintf("Process %d of user %d is unauthorized", pStatus.Pid, pStatus.RealUid)
		}

		return OK, ""
	}, nil
}

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
