package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	linuxproc "github.com/c9s/goprocinfo/linux"
)

var (
	programName = "go-nhc"
)

type Context struct {
	checks  map[string]Check
	mounts  *linuxproc.Mounts
	memInfo *linuxproc.MemInfo
	cpuInfo *linuxproc.CPUInfo
	psInfo  []*linuxproc.ProcessStatus
	jobInfo []Job
}

func main() {
	_, err := fApp.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	context := Context{
		checks: map[string]Check{},
	}

	context.RegisterEach("interface_%s", context.CheckInterface, *fCheckInterfaces)
	context.RegisterEach("ib_%s", context.CheckInfiniband, *fCheckInfinibands)
	context.RegisterEach("mount_%s", context.CheckMount, *fCheckMounts)
	context.RegisterEach("du_%s", context.CheckDiskUsage, *fCheckDiskUsages)
	context.RegisterEach("file_%s", context.CheckFile, *fCheckFiles)
	context.RegisterEach("user_%s", context.CheckUser, *fCheckUsers)
	context.RegisterEach("ps_%s", context.CheckProcess, *fCheckProcesses)
	context.RegisterEach("port_%s", context.CheckPort, *fCheckPorts)
	context.RegisterEach("cmd_%s", context.CheckCommand, *fCheckCommands)
	context.Register("mem_phys", context.CheckFreeMemory, *fCheckFreeMemory)
	context.Register("mem_swap", context.CheckFreeSwap, *fCheckFreeSwap)
	context.Register("mem_total", context.CheckFreeTotalMemory, *fCheckFreeTotalMemory)
	context.Register("mem_dimms", context.CheckDimms, *fCheckDimms)
	context.Register("cpu_sockets", context.CheckCPUSockets, *fCheckCPUSockets)
	context.Register("cpu_hyperthreading", context.CheckHyperthreading, *fCheckHyperthreading)
	context.Register("ps_unauthorized", context.CheckUnauthorized, *fCheckUnauthorized)

	context.RunChecks(*fVerbose, *fList, *fAll, !*fNoSend)
}

func (c *Context) RegisterEach(id_format string, factory CheckFactory, arguments []string) {
	for _, argument := range arguments {
		if argument == "" {
			continue
		}
		id := fmt.Sprintf(id_format, ArgumentToId(argument))
		c.Register(id, factory, argument)
	}
}

func (c *Context) Register(id string, factory CheckFactory, argument string) {
	if argument == "" {
		return
	}
	check, err := factory(argument)
	if err != nil {
		fmt.Printf("[%s] Parse error: %s\n", id, err.Error())
		os.Exit(127)
	}
	c.checks[id] = check
}

func (c *Context) RunChecks(verbose bool, list bool, all bool, send bool) {
	var global Status
	var failed int

	for id, check := range c.checks {
		status, message := check()

		if status == Ignore {
			if verbose {
				fmt.Printf("%s: [%s] %s\n", status.String(), id, message)
			}
			continue
		}

		if send {
			err := SendSensuResult(id, status, message)
			if err != nil {
				fmt.Printf("Sending result of %s to sensu failed: %s\n", id, err.Error())
			}
		}

		if !all && status.IsFatal() {
			fmt.Printf("ERROR %s: [%s] %s\n", status.String(), id, message)
			os.Exit(status.RC())
		} else if status != OK || list {
			fmt.Printf("%s: [%s] %s\n", status.String(), id, message)
		}

		if status != OK {
			failed++
			if status.Compare(global) > 0 {
				global = status
			}
		}
	}

	if verbose {
		if failed > 0 {
			fmt.Printf("%d checks failed\n", failed)
		} else {
			fmt.Println("All checks returned OK")
		}
	}

	os.Exit(global.RC())
}

func ArgumentToId(argument string) string {
	parts := strings.SplitN(argument, " ", 2)
	re := regexp.MustCompile(`[_/:]+`)
	argument = re.ReplaceAllString(parts[0], "_")
	return strings.TrimLeft(argument, "_")
}
