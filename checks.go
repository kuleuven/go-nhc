package main

import (
    "fmt"
    "strings"
    "regexp"
    "strconv"
    
    "github.com/inhies/go-bytesize"
    linuxproc "github.com/c9s/goprocinfo/linux"
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

func (c *Context) CheckInfiniband(device_info string) (Check, error) {
    parts := strings.SplitN(device_info, "=", 2)
    if len(parts) < 2 {
        return nil, fmt.Errorf("Could not parse device %s", device_info)
    }

    speed, err := strconv.Atoi(parts[1])
    if err != nil {
        return nil, err
    }

    parts = strings.SplitN(parts[0], ":", 2)
    if len(parts) < 2 {
        return nil, fmt.Errorf("Could not parse device %s", device_info)
    }

    device := parts[0]

    port, err := strconv.Atoi(parts[1])
    if err != nil {
        return nil, err
    }

    state_re := regexp.MustCompile("4: ACTIVE")
    speed_re, err := regexp.Compile(fmt.Sprintf("^%d\\s", speed))
    if err != nil {
        return nil, err
    }

    return func() (Status, string) {
        file := fmt.Sprintf("/sys/class/infiniband/%s/ports/%d/state", device, port)
        status, message := AssureContent(file, state_re)
        if status != OK {
            return status, message
        }

        file = fmt.Sprintf("/sys/class/infiniband/%s/ports/%d/rate", device, port)
        return AssureContent(file, speed_re)
    }, nil
}

func (c *Context) CheckMount(mount string) (Check, error) {
    parts := strings.SplitN(mount, "=", 3)
    mountpoint := parts[0]

    var device string
    if len(parts) > 1 {
        device = parts[1]
    }

    var fstype string
    if len(parts) > 2 {
        fstype = parts[2]
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
            if mount.MountPoint == mountpoint {
                if device != "" && mount.Device != device {
                    return Critical, fmt.Sprintf("Mountpoint %s does not match required device %s: %s", mountpoint, device, mount.Device)
                }
                if fstype != "" && mount.FSType != fstype {
                    return Warning, fmt.Sprintf("Mountpoint %s does not match required fstype %s: %s", mountpoint, fstype, mount.FSType)
                }
                if mount.Options != "rw" && ! strings.HasPrefix(mount.Options, "rw,") {
                    return Critical, fmt.Sprintf("Mountpoint %s is not mounted read-write: %s", mountpoint, mount.Options)
                }

                return OK, ""
            }
        }

        return Critical, fmt.Sprintf("Mountpoint %s is not mounted", mountpoint)
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

func (c *Context) CheckDiskUsage(mount string) (Check, error) {
    parts := strings.SplitN(mount, "=", 2)
    if len(parts) < 2 {
        return nil, fmt.Errorf("Could not parse mountpoint and usage level %s", mount)
    }

    mountpoint := parts[0]
    threshold := parts[1]

    if strings.HasSuffix(threshold, "%") {
        percent, err := strconv.ParseFloat(threshold[:len(threshold)-1], 32)
        if err != nil {
            return nil, err
        }

        return func() (Status, string) {
            stat, err := DiskUsage(mountpoint)
            if err != nil {
                return Unknown, fmt.Sprintf("Could not retrieve disk usage: %s", err.Error())
            }

            if stat.Used > uint64(float64(stat.All) * percent) {
                bs := bytesize.ByteSize(stat.Used)
                return Critical, fmt.Sprintf("Disk usage is above %d%: %s", percent, bs.String())
            }
            
            return OK, ""
        }, nil
    } else {
        th, err := bytesize.Parse(threshold)
        if err != nil {
            return nil, err
        }

        return func() (Status, string) {
            stat, err := DiskUsage(mountpoint)
            if err != nil {
                return Unknown, fmt.Sprintf("Could not retrieve disk usage: %s", err.Error())
            }

            if stat.Free < uint64(th) {
                bs := bytesize.ByteSize(stat.Free)
                return Critical, fmt.Sprintf("Free disk space is lower than threshold %s: %s", th.String(), bs.String())
            }
            
            return OK, ""
        }, nil
    }
}
