package main

import (
    "fmt"
    "strings"
    "regexp"
    "strconv"
    
    linuxproc "github.com/c9s/goprocinfo/linux"
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
    integer, err := strconv.ParseUint(amount, 10, 64)
    if err != nil {
        return nil, err
    }
    return func() (Status, string) {
        if c.memInfo == nil {
            var err error
            c.memInfo, err = linuxproc.ReadMemInfo("/proc/meminfo")
            if err != nil {
                return Unknown, fmt.Sprintf("Could not parse meminfo: %s", err.Error())
            }
        }

        if c.memInfo.MemFree < integer {
            return Warning, fmt.Sprintf("Free memory is less than threshold %d: %d", integer, c.memInfo.MemFree)
        }

        return OK, ""
    }, nil
}

func (c *Context) CheckFreeSwap(amount string) (Check, error) {
    integer, err := strconv.ParseUint(amount, 10, 64)
    if err != nil {
        return nil, err
    }
    return func() (Status, string) {
        if c.memInfo == nil {
            var err error
            c.memInfo, err = linuxproc.ReadMemInfo("/proc/meminfo")
            if err != nil {
                return Unknown, fmt.Sprintf("Could not parse meminfo: %s", err.Error())
            }
        }

        if c.memInfo.SwapFree < integer {
            return Warning, fmt.Sprintf("Free memory is less than threshold %d: %d", integer, c.memInfo.SwapFree)
        }

        return OK, ""
    }, nil
}

func (c *Context) CheckFreeTotalMemory(amount string) (Check, error) {
    integer, err := strconv.ParseUint(amount, 10, 64)
    if err != nil {
        return nil, err
    }
    return func() (Status, string) {
        if c.memInfo == nil {
            var err error
            c.memInfo, err = linuxproc.ReadMemInfo("/proc/meminfo")
            if err != nil {
                return Unknown, fmt.Sprintf("Could not parse meminfo: %s", err.Error())
            }
        }

        if c.memInfo.MemFree + c.memInfo.SwapFree < integer {
            return Warning, fmt.Sprintf("Free memory is less than threshold %d: %d", integer, c.memInfo.MemFree + c.memInfo.SwapFree)
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
            c.cpuInfo, err = linuxproc.ReadCPUInfo("/proc/cpuinfo")
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
            c.cpuInfo, err = linuxproc.ReadCPUInfo("/proc/cpuinfo")
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
