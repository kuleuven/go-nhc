package main

import (
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	fApp = kingpin.New(programName, "Node Health Check")
	
    fCheckInterfaces      = fApp.Flag("interface", "Check the listed network interfaces").Default("").Strings()
    fCheckInfinibands     = fApp.Flag("infiniband", "Check the listed infiniband ports, format: device:port=speed").Default("").Strings()
    fCheckMounts          = fApp.Flag("mount", "Check whether the listed mounts exist, format: mountpoint[=device[=fstype]]").Default("").Strings()
    fCheckDiskUsages      = fApp.Flag("disk-usage", "Check whether the disk usage is below the threshold, format: mountpoint=threshold. Threshold can be the minimum free space in bytes, or the maximum percentage of disk that may be filled.").Default("").Strings()
    fCheckFiles           = fApp.Flag("file", "Check whether the listed files exist").Default("").Strings()
    fCheckFreeMemory      = fApp.Flag("memory", "Check whether the given amount of physical memory is free").Default("").String()
    fCheckFreeSwap        = fApp.Flag("swap", "Check whether the given amount of swap memory is free").Default("").String()
    fCheckFreeTotalMemory = fApp.Flag("total-memory", "Check whether the given amount of total memory is free").Default("").String()
    fCheckDimms           = fApp.Flag("dimms", "Check that each memory channel has the same number of dimms, and that the dimm size is consistent").Default("").Enum("consistent", "")
    fCheckHyperthreading  = fApp.Flag("hyperthreading", "Check whether hyperthreading is enabled or disabled").Default("").Enum("enabled", "disabled", "")
    fCheckCPUSockets      = fApp.Flag("cpu-sockets", "Check whether the given amount of cpu sockets is present").Default("").String()

	//fBackendPort          = fApp.Flag("port", "Port").Default("2200").Int()
	//fBackendSSHPort       = fApp.Flag("ssh-port", "Port of SSH process").Default("22").Int()
	//fBackendLimitServerIP = fApp.Flag("limit", "Limit backend access to these published balancer addresses").Default("0.0.0.0").IPList()

)
