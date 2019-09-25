// +build linux

package main

import (
	"fmt"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"

	linuxproc "github.com/c9s/goprocinfo/linux"
	"github.com/inhies/go-bytesize"
)

const (
	cpuinfo_file = "/proc/cpuinfo"
	meminfo_file = "/proc/meminfo"
)

func (c *Context) CheckInterface(iface string) (Check, error) {
	return func() (Status, string) {
		return AssureExists(fmt.Sprintf("/sys/class/net/%s", iface))
	}, nil
}

type InfinibandMetadata struct {
	Device string
	Port   int
	Speed  int
}

func (c *Context) CheckInfiniband(argument string) (Check, error) {
	m := &InfinibandMetadata{}
	err := ParseMetadata(m, argument, "Device")
	if err != nil {
		return nil, err
	}

	state_re := regexp.MustCompile("4: ACTIVE")
	speed_re, err := regexp.Compile(fmt.Sprintf("^%d\\s", m.Speed))
	if err != nil {
		return nil, err
	}

	return func() (Status, string) {
		file := fmt.Sprintf("/sys/class/infiniband/%s/ports/%d/state", m.Device, m.Port)
		status, message := AssureContent(file, state_re)
		if status != OK {
			return status, message
		}

		file = fmt.Sprintf("/sys/class/infiniband/%s/ports/%d/rate", m.Device, m.Port)
		return AssureContent(file, speed_re)
	}, nil
}

type MountMetadata struct {
	MountPoint string
	Device     string
	FsType     string
	ReadOnly   bool
}

func (c *Context) CheckMount(argument string) (Check, error) {
	m := &MountMetadata{}
	err := ParseMetadata(m, argument, "MountPoint")
	if err != nil {
		return nil, err
	}

	return func() (Status, string) {
		if c.mounts == nil {
			var err error
			c.mounts, err = linuxproc.ReadMounts("/proc/mounts")
			if err != nil {
				return Unknown, fmt.Sprintf("Could not parse mounts: %s", err.Error())
			}
		}

		for _, mount := range c.mounts.Mounts {
			if mount.MountPoint == m.MountPoint {
				if m.Device != "" && mount.Device != m.Device {
					return Critical, fmt.Sprintf("Mount point %s does not match required device %s: %s", m.MountPoint, m.Device, mount.Device)
				}
				if m.FsType != "" && mount.FSType != m.FsType {
					return Warning, fmt.Sprintf("Mount point %s does not match required fstype %s: %s", m.MountPoint, m.FsType, mount.FSType)
				}
				if !m.ReadOnly && mount.Options != "rw" && !strings.HasPrefix(mount.Options, "rw,") {
					return Critical, fmt.Sprintf("Mount point %s is not mounted read-write: %s", m.MountPoint, mount.Options)
				}

				return OK, ""
			}
		}

		return Critical, fmt.Sprintf("Mount point %s is not mounted", m.MountPoint)
	}, nil
}

func (c *Context) CheckFile(file string) (Check, error) {
	return func() (Status, string) {
		return AssureExists(file)
	}, nil
}

func (c *Context) CheckUser(username string) (Check, error) {
	return func() (Status, string) {
		_, err := user.Lookup(username)
		if err != nil {
			return Critical, fmt.Sprintf("User information for %s could not be retrieved: %s", username, err.Error())
		}
		return OK, ""
	}, nil
}

func (c *Context) CheckFreeMemory(amount string) (Check, error) {
	th, err := bytesize.Parse(amount)
	if err != nil {
		return nil, err
	}
	return func() (Status, string) {
		if c.memInfo == nil {
			var err error
			c.memInfo, err = linuxproc.ReadMemInfo(meminfo_file)
			if err != nil {
				return Unknown, fmt.Sprintf("Could not parse meminfo: %s", err.Error())
			}
		}

		if c.memInfo.MemFree < uint64(th) {
			bs := bytesize.ByteSize(c.memInfo.MemFree)
			return Warning, fmt.Sprintf("Free memory is less than threshold %s: %s", th.String(), bs.String())
		}

		return OK, ""
	}, nil
}

func (c *Context) CheckFreeSwap(amount string) (Check, error) {
	th, err := bytesize.Parse(amount)
	if err != nil {
		return nil, err
	}
	return func() (Status, string) {
		if c.memInfo == nil {
			var err error
			c.memInfo, err = linuxproc.ReadMemInfo(meminfo_file)
			if err != nil {
				return Unknown, fmt.Sprintf("Could not parse meminfo: %s", err.Error())
			}
		}

		if c.memInfo.SwapFree < uint64(th) {
			bs := bytesize.ByteSize(c.memInfo.SwapFree)
			return Warning, fmt.Sprintf("Free memory is less than threshold %s: %s", th.String(), bs.String())
		}

		return OK, ""
	}, nil
}

func (c *Context) CheckFreeTotalMemory(amount string) (Check, error) {
	th, err := bytesize.Parse(amount)
	if err != nil {
		return nil, err
	}
	return func() (Status, string) {
		if c.memInfo == nil {
			var err error
			c.memInfo, err = linuxproc.ReadMemInfo(meminfo_file)
			if err != nil {
				return Unknown, fmt.Sprintf("Could not parse meminfo: %s", err.Error())
			}
		}

		total := c.memInfo.MemFree + c.memInfo.SwapFree
		if total < uint64(th) {
			bs := bytesize.ByteSize(total)
			return Warning, fmt.Sprintf("Free memory is less than threshold %s: %s", th.String(), bs.String())
		}

		return OK, ""
	}, nil
}

func (c *Context) CheckDimms(argument string) (Check, error) {
	return func() (Status, string) {
		channels, err := ListMemoryChannels()
		if err != nil {
			return Unknown, fmt.Sprintf("Could not parse dimm info: %s", err.Error())
		}

		var dimmsPerChannel int
		var dimmSize uint64

		for _, channel := range channels {
			if dimmsPerChannel == 0 {
				dimmsPerChannel = len(channel.Dimms)

				if dimmsPerChannel == 0 {
					return Warning, "First memory channel has no dimms"
				}
			} else if dimmsPerChannel != len(channel.Dimms) {
				return Warning, fmt.Sprintf("Number of dimms differ per memory channel: first has %d channels, %s has %d channels", dimmsPerChannel, channel.Name, len(channel.Dimms))
			}

			for _, dimm := range channel.Dimms {
				if dimmSize == 0 {
					dimmSize = dimm.Size

					if dimmSize == 0 {
						return Warning, "First dimm has no size"
					}
				} else if dimmSize != dimm.Size {
					return Warning, fmt.Sprintf("Dimm sizes differ: first dimm has size %d, %s/%s has size %d", dimmSize, channel.Name, dimm.Name, dimm.Size)
				}
			}
		}

		return OK, ""
	}, nil
}

func (c *Context) CheckHyperthreading(state string) (Check, error) {
	var check bool
	switch state {
	case "enabled":
		check = true
	case "disabled":
		check = false
	default:
		return nil, fmt.Errorf("Unknown target state %s", state)
	}

	return func() (Status, string) {
		if c.cpuInfo == nil {
			var err error
			c.cpuInfo, err = linuxproc.ReadCPUInfo(cpuinfo_file)
			if err != nil {
				return Unknown, fmt.Sprintf("Could not parse cpuinfo: %s", err.Error())
			}
		}

		core := c.cpuInfo.NumCore()
		cpu := c.cpuInfo.NumCPU()

		if (core == cpu) == check {
			return Warning, fmt.Sprintf("Hyperthreading must be %v, but found %v physical cores and %v cores", check, core, cpu)
		}

		return OK, ""
	}, nil
}

func (c *Context) CheckCPUSockets(amount string) (Check, error) {
	integer, err := strconv.Atoi(amount)
	if err != nil {
		return nil, err
	}
	return func() (Status, string) {
		if c.cpuInfo == nil {
			var err error
			c.cpuInfo, err = linuxproc.ReadCPUInfo(cpuinfo_file)
			if err != nil {
				return Unknown, fmt.Sprintf("Could not parse cpuinfo: %s", err.Error())
			}
		}

		phys := c.cpuInfo.NumPhysicalCPU()

		if phys != integer {
			return Critical, fmt.Sprintf("Expected %d CPU sockets, found %d", integer, phys)
		}

		return OK, ""
	}, nil
}

type DiskUsageMetadata struct {
	MountPoint     string
	MaxUsedPercent int
	MinFree        bytesize.ByteSize
}

func (c *Context) CheckDiskUsage(argument string) (Check, error) {
	m := &DiskUsageMetadata{}
	err := ParseMetadata(m, argument, "MountPoint")
	if err != nil {
		return nil, err
	}

	return func() (Status, string) {
		stat, err := DiskUsage(m.MountPoint)
		if err != nil {
			return Unknown, fmt.Sprintf("Could not retrieve disk usage: %s", err.Error())
		}

		if m.MaxUsedPercent > 0 && stat.Used*100 > stat.All*uint64(m.MaxUsedPercent) {
			bs := bytesize.ByteSize(stat.Used)
			return Critical, fmt.Sprintf("Disk usage is above %d%: %s", m.MaxUsedPercent, bs.String())
		}

		if uint64(m.MinFree) > 0 && stat.Free < uint64(m.MinFree) {
			bs := bytesize.ByteSize(stat.Free)
			return Critical, fmt.Sprintf("Free disk space is lower than threshold %s: %s", m.MinFree.String(), bs.String())
		}

		return OK, ""
	}, nil
}

type ProcessMetadata struct {
	Service string
	Daemon  string
	User    string
	Start   bool
	Restart bool
}

func (c *Context) CheckProcess(argument string) (Check, error) {
	m := &ProcessMetadata{}
	err := ParseMetadata(m, argument, "Service")
	if err != nil {
		return nil, err
	}

	if m.Daemon == "" {
		m.Daemon = m.Service
	}

	return func() (Status, string) {
		if c.psInfo == nil {
			var err error
			c.psInfo, err = ListProcesses()
			if err != nil {
				return Unknown, fmt.Sprintf("Could not parse process info: %s", err.Error())
			}
		}

		for _, pStatus := range c.psInfo {
			if pStatus.Name == m.Daemon {
				if m.User != "" {
					user, err := user.Lookup(m.User)
					if err != nil {
						return Unknown, fmt.Sprintf("Could not lookup %s: %s", m.User, err.Error())
					}
					uid, err := strconv.ParseUint(user.Uid, 10, 64)
					if err != nil {
						return Unknown, fmt.Sprintf("Could not parse uid %s: %s", user.Uid, err.Error())
					}
					if pStatus.EffectiveUid != uid {
						return Warning, fmt.Sprintf("Process %s is not running under user %d (%s), but %d", uid, m.User, pStatus.EffectiveUid)
					}
				}

				return OK, ""
			}
		}

		if m.Start || m.Restart {
			var action string
			switch {
			case m.Start:
				action = "start"
			case m.Restart:
				action = "restart"
			}

			cmd := exec.Command("/usr/bin/systemctl", action, m.Service)
			err := cmd.Run()
			if err != nil {
				return Critical, fmt.Sprintf("Process %s not found, and could not start: %s", m.Service, err.Error())
			}
			return Warning, fmt.Sprintf("Process %s not found, %sed it successfully", m.Service, action)
		}

		return Critical, fmt.Sprintf("Process %s not found", m.Service)
	}, nil
}

type UnauthorizedMetadata struct {
	Scheduler    string
	MaxSystemUid uint64
}

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
			user, err := user.LookupId(fmt.Sprintf("%d", pStatus.RealUid))
			if err != nil {
				return Critical, fmt.Sprintf("Process %d is runned by unknown uid %d: %s", pStatus.Pid, pStatus.RealUid, err.Error())
			}
			groups, err := user.GroupIds()
			if err != nil {
				return Critical, fmt.Sprintf("Could not lookup groups of user %d (%s): %s", pStatus.RealUid, user.Username, err.Error())
			}
			groupInts := []uint64{}
			for _, group := range groups {
				groupInt, err := strconv.ParseUint(group, 10, 64)
				if err != nil {
					return Critical, fmt.Sprintf("Could not parse group ids for user %d (%s): %s", pStatus.RealUid, user.Username, err.Error())
				}
				groupInts = append(groupInts, groupInt)
			}
			for _, job := range c.jobInfo {
				if job.Uid == pStatus.RealUid {
					continue OUTER
				}
				for _, gid := range groupInts {
					if job.Gid == gid {
						continue OUTER
					}
				}
			}
			return Critical, fmt.Sprintf("Process %d of user %d (%s) is unauthorized", pStatus.Pid, pStatus.RealUid, user.Username)
		}

		return OK, ""
	}, nil
}
