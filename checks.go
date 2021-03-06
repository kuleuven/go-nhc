// +build linux

package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gitea.icts.kuleuven.be/ceif-lnx/go-nhc/utils"
	linuxproc "github.com/c9s/goprocinfo/linux"
	"github.com/inhies/go-bytesize"
)

const (
	cpuinfo_file = "/proc/cpuinfo"
	meminfo_file = "/proc/meminfo"
)

func (c *Context) CheckInterface(iface string) (Check, error) {
	return func() (Status, string) {
		if iface == "lo" {
			return AssureExists(fmt.Sprintf("/sys/class/net/%s", iface))
		}

		reUP := regexp.MustCompile(`up`)

		return AssureContent(fmt.Sprintf("/sys/class/net/%s/operstate", iface), reUP)
	}, nil
}

type InfinibandMetadata struct {
	Device   string
	Port     int
	Speed    int
	Warning  uint64
	Critical uint64
}

var (
	infinibandErrorCounters = []string{
		"symbol_error",
		"link_downed",
		"port_rcv_errors",
		"local_link_integrity_errors",
		"excessive_buffer_overrun_errors",
		"VL15_dropped",
	}
)

func (c *Context) CheckInfiniband(argument string) (Check, error) {
	m := &InfinibandMetadata{
		Port:     1,
		Warning:  10,
		Critical: 100,
	}
	err := ParseMetadata(m, argument, "Device")
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(m.Device, ":", 2)
	if len(parts) == 2 {
		m.Device = parts[0]
		m.Port, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
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
		status, message = AssureContent(file, speed_re)
		if status != OK {
			return status, message
		}

		message = ""

		for _, counter := range infinibandErrorCounters {
			file = fmt.Sprintf("/sys/class/infiniband/%s/ports/%d/counters/%s", m.Device, m.Port, counter)

			handle, err := os.Open(file)
			defer handle.Close()
			if err != nil {
				return Unknown, fmt.Sprintf("Could not open file %s: %s", file, err.Error())
			}

			b, err := ioutil.ReadAll(handle)
			if err != nil {
				return Unknown, fmt.Sprintf("Could not read file %s: %s", file, err.Error())
			}

			value := strings.TrimSuffix(string(b), "\n")
			i, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return Unknown, fmt.Sprintf("Could not read parse %s as integer while reading %s: %s", value, counter, err.Error())
			}
			if i >= m.Critical && status != Critical {
				status = Critical
				message = fmt.Sprintf("Port counter %s is higher than threshold %d: %d", counter, m.Critical, i)
			} else if i >= m.Warning && status == OK {
				status = Warning
				message = fmt.Sprintf("Port counter %s is higher than threshold %d: %d", counter, m.Warning, i)
			}
		}

		return status, message
	}, nil
}

type MountMetadata struct {
	MountPoint string
	Device     string
	FsType     string
	ReadOnly   bool
	Remount    bool
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
			if mount.MountPoint == m.MountPoint && mount.FSType != "autofs" {
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

		if m.Remount {
			cmd := exec.Command("/usr/bin/mount", m.MountPoint)
			err = cmd.Run()
			if err != nil {
				return Critical, fmt.Sprintf("Mount point %s is not mounted, remount failed: %s", m.MountPoint, err)
			}
			return Warning, fmt.Sprintf("Mount point %s was not mounted, did remount it", m.MountPoint)
		}

		return Critical, fmt.Sprintf("Mount point %s is not mounted", m.MountPoint)
	}, nil
}

func (c *Context) CheckFile(file string) (Check, error) {
	return func() (Status, string) {
		return AssureExists(file)
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

		if c.memInfo.MemAvailable < uint64(th) {
			bs := bytesize.ByteSize(c.memInfo.MemAvailable)
			return Critical, fmt.Sprintf("Available memory is less than threshold %s: %s", th.String(), bs.String())
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

		total := c.memInfo.MemAvailable + c.memInfo.SwapFree
		if total < uint64(th) {
			bs := bytesize.ByteSize(total)
			return Critical, fmt.Sprintf("Available memory is less than threshold %s: %s", th.String(), bs.String())
		}

		return OK, ""
	}, nil
}

func (c *Context) CheckDimms(argument string) (Check, error) {
	return func() (Status, string) {
		channels, err := utils.ListMemoryChannels()
		if err != nil {
			return Unknown, fmt.Sprintf("Could not parse dimm info: %s", err.Error())
		}

		var dimmsPerChannel int
		var dimmSize uint64

		for _, channel := range channels {
			if dimmsPerChannel == 0 {
				dimmsPerChannel = len(channel.Dimms)

				if dimmsPerChannel == 0 {
					return Critical, "First memory channel has no dimms"
				}
			} else if dimmsPerChannel != len(channel.Dimms) {
				return Critical, fmt.Sprintf("Number of dimms differ per memory channel: first has %d channels, %s has %d channels", dimmsPerChannel, channel.Name, len(channel.Dimms))
			}

			for _, dimm := range channel.Dimms {
				if dimmSize == 0 {
					dimmSize = dimm.Size

					if dimmSize == 0 {
						return Critical, "First dimm has no size"
					}
				} else if dimmSize != dimm.Size {
					return Critical, fmt.Sprintf("Dimm sizes differ: first dimm has size %d, %s/%s has size %d", dimmSize, channel.Name, dimm.Name, dimm.Size)
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
			return Critical, fmt.Sprintf("Hyperthreading must be %v, but found %v physical cores and %v cores", check, core, cpu)
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
		stat, err := utils.DiskUsage(m.MountPoint)
		if err != nil {
			return Unknown, fmt.Sprintf("Could not retrieve disk usage: %s", err.Error())
		}

		if m.MaxUsedPercent > 0 && stat.BlocksUsed*100 > stat.Blocks*uint64(m.MaxUsedPercent) {
			bs := bytesize.ByteSize(stat.BlocksUsed)
			return Critical, fmt.Sprintf("Disk usage is above %d%%: %s", m.MaxUsedPercent, bs.String())
		}

		if uint64(m.MinFree) > 0 && stat.BlocksFree < uint64(m.MinFree) {
			bs := bytesize.ByteSize(stat.BlocksFree)
			return Critical, fmt.Sprintf("Free disk space is lower than threshold %s: %s", m.MinFree.String(), bs.String())
		}

		return OK, ""
	}, nil
}

func (c *Context) CheckPort(argument string) (Check, error) {
	port, err := strconv.Atoi(argument)
	if err != nil {
		return nil, err
	}

	return func() (Status, string) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf(":%d", port), time.Second)
		if err != nil {
			return Critical, fmt.Sprintf("Could not connect to port %d: %s", port, err.Error())
		}
		defer conn.Close()
		return OK, ""
	}, nil
}

type ProcessMetadata struct {
	Service string
	Daemon  string
	Cmdline string
	User    string
	Count   uint
	Start   bool
	Restart bool
	Fatal   bool
}

func (c *Context) CheckProcess(argument string) (Check, error) {
	m := &ProcessMetadata{
		Fatal: true,
		Count: 1,
	}
	err := ParseMetadata(m, argument, "Service")
	if err != nil {
		return nil, err
	}

	if m.Daemon == "" {
		m.Daemon = m.Service
	}

	m.Cmdline = strings.ReplaceAll(m.Cmdline, "+", " ")

	return func() (Status, string) {
		if c.psInfo == nil {
			var err error
			c.psInfo, err = utils.ListProcesses()
			if err != nil {
				return Unknown, fmt.Sprintf("Could not parse process info: %s", err.Error())
			}
		}

		var seen uint

		for _, pStatus := range c.psInfo {
			if pStatus.Name == m.Daemon {
				if m.Cmdline != "" {
					pCmdline, err := linuxproc.ReadProcessCmdline(fmt.Sprintf("/proc/%d/cmdline", pStatus.Pid))
					if err != nil {
						// Ignore - process is probably gone
						continue
					}

					if strings.Contains(m.Cmdline, " ") {
						// Compare substring
						if m.Cmdline != pCmdline && !strings.HasPrefix(pCmdline, m.Cmdline+" ") {
							continue
						}
					} else {
						// Compare second part
						parts := strings.SplitN(pCmdline, " ", 3)
						if len(parts) < 2 || m.Cmdline != parts[1] {
							continue
						}
					}
				}

				if m.User != "" {
					user, err := user.Lookup(m.User)
					if err != nil {
						return Unknown.NonFatalUnless(m.Fatal), fmt.Sprintf("Could not lookup %s: %s", m.User, err.Error())
					}
					uid, err := strconv.ParseUint(user.Uid, 10, 64)
					if err != nil {
						return Unknown.NonFatalUnless(m.Fatal), fmt.Sprintf("Could not parse uid %s: %s", user.Uid, err.Error())
					}
					if pStatus.EffectiveUid != uid {
						return Critical, fmt.Sprintf("Process %s is not running under user %d (%s), but %d", m.Service, uid, m.User, pStatus.EffectiveUid)
					}
				}

				seen++
			}
		}

		if m.Count > 0 {
			if seen < m.Count && seen > 0 {
				return Warning, fmt.Sprintf("Process count is not reached: expected %d, got %d", m.Count, seen)
			}

			if seen > m.Count {
				return Warning, fmt.Sprintf("Process count is exceeded: expected %d, got %d", m.Count, seen)
			}
		}

		if seen > 0 {
			return OK, ""
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
				return Critical.NonFatalUnless(m.Fatal), fmt.Sprintf("Process %s not found, and could not start: %s", m.Service, err.Error())
			}
			return Critical, fmt.Sprintf("Process %s not found, %sed it successfully", m.Service, action)
		}

		return Critical.NonFatalUnless(m.Fatal), fmt.Sprintf("Process %s not found", m.Service)
	}, nil
}

type UnauthorizedMetadata struct {
	Scheduler    string
	MaxSystemUid uint64
}

func (c *Context) CheckCommand(argument string) (Check, error) {
	return func() (Status, string) {
		cmd := exec.Command("/bin/sh", "-c", argument)
		err := cmd.Run()
		if err != nil {
			return Critical, fmt.Sprintf("Command %s could not run successfully: %s", argument, err.Error())
		}

		return OK, ""
	}, nil
}
