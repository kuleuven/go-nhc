package main

import (
	"log"
	"os"
    "fmt"
    "strings"
    "regexp"
    
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
    context.RegisterEach("file_%s", context.CheckFile, *fCheckFiles)
    context.Register("mem_phys", context.CheckFreeMemory, *fCheckFreeMemory)
    context.Register("mem_swap", context.CheckFreeSwap, *fCheckFreeSwap)
    context.Register("mem_total", context.CheckFreeTotalMemory, *fCheckFreeTotalMemory)
    context.Register("cpu_hyperthreading", context.CheckHyperthreading, *fCheckHyperthreading)
    context.Register("cpu_sockets", context.CheckCPUSockets, *fCheckCPUSockets)

    context.RunChecks()
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

func (c *Context) RunChecks() {
    for id, check := range c.checks {
        status, message := check()

        if status != OK {
            fmt.Printf("%s: [%s] %s\n", status.String(), id, message)
            os.Exit(status.RC())
        }
    }

    fmt.Println("All checks returned OK")
}

func ArgumentToId(argument string) string {
    parts := strings.SplitN(argument, "=", 2)
    re := regexp.MustCompile(`[_/:]+`)
    argument = re.ReplaceAllString(parts[0], "_")
    return strings.TrimLeft(argument, "_")
}
