package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/inhies/go-bytesize"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	fApp = kingpin.New(programName, "Node Health Check")

	fCheckInterfaces      = fApp.Flag("interface", "Check the listed network interfaces").Default("").Strings()
	fCheckInfinibands     = fApp.Flag("infiniband", "Check the listed infiniband ports, format: '<DEV> port=<PORT> speed=<NUM>'").Default("").Strings()
	fCheckMounts          = fApp.Flag("mount", "Check whether the listed mounts exist, format: '<MOUNTPOINT> [device=<DEV>] [fs_type=<TYPE>]'").Default("").Strings()
	fCheckDiskUsages      = fApp.Flag("disk-usage", "Check whether the disk usage is below the threshold, format: 'mountpoint [max_used_percent=<INT>] [min_free=<SIZE>]'").Default("").Strings()
	fCheckFiles           = fApp.Flag("file", "Check whether the listed files exist").Default("").Strings()
	fCheckUsers           = fApp.Flag("user", "Check whether the listed users exist").Default("").Strings()
	fCheckProcesses       = fApp.Flag("process", "Check whether the given service is running, format: '<SERVICE> [daemon=<PROCESS_NAME>] [user=<USER>]'").Default("").Strings()
	fCheckPorts           = fApp.Flag("port", "Check whether a service is listening at the given port").Default("").Strings()
	fCheckCommands        = fApp.Flag("command", "Check whether the given command returns successfully").Default("").Strings()
	fCheckFreeMemory      = fApp.Flag("memory", "Check whether the given amount of physical memory is free").Default("").String()
	fCheckFreeSwap        = fApp.Flag("swap", "Check whether the given amount of swap memory is free").Default("").String()
	fCheckFreeTotalMemory = fApp.Flag("total-memory", "Check whether the given amount of total memory is free").Default("").String()
	fCheckDimms           = fApp.Flag("dimms", "Check that each memory channel has the same number of dimms, and that the dimm size is consistent").Default("").Enum("consistent", "")
	fCheckHyperthreading  = fApp.Flag("hyperthreading", "Check whether hyperthreading is enabled or disabled").Default("").Enum("enabled", "disabled", "")
	fCheckCPUSockets      = fApp.Flag("cpu-sockets", "Check whether the given amount of cpu sockets is present").Default("").String()
	fCheckUnauthorized    = fApp.Flag("unauthorized", "Check whether unauthorized jobs are running, not governed by the specified job scheduler").Default("").String()

	fVerbose   = fApp.Flag("verbose", "Verbose mode - show ignored checks and print summarizing message").Short('v').Bool()
	fList      = fApp.Flag("list", "List all checks that passed").Short('l').Bool()
	fOnlyFatal = fApp.Flag("only-fatal", "Ignore Warnings, only fail on a fatal check status").Short('o').Bool()
	fAll       = fApp.Flag("all", "Run all checks, do not stop on first failed check").Short('a').Bool()
	fSend      = fApp.Flag("send", "Send check results to sensu agent").Short('s').Bool()
)

var (
	metadataMapRegex = regexp.MustCompile("[=]")
	stringType       = reflect.TypeOf("")
	intType          = reflect.TypeOf(0)
	uint64Type       = reflect.TypeOf(uint64(0))
	boolType         = reflect.TypeOf(false)
	byteSizeType     = reflect.TypeOf(bytesize.ByteSize(0))
)

func ParseMetadata(meta interface{}, argument string, default_key string) error {
	arguments := strings.Split(argument, " ")
	target := reflect.ValueOf(meta).Elem()

	for index, value := range arguments {
		parts := metadataMapRegex.Split(value, 2)

		var key string
		var str string

		if index == 0 && len(parts) == 1 && default_key != "" {
			key = default_key
			str = parts[0]
		} else if len(parts) != 2 {
			return fmt.Errorf("expected KEY=VALUE got '%s'", value)
		} else {
			key = parts[0]
			str = parts[1]
		}

		field := target.FieldByName(strcase.ToCamel(key))
		if !field.IsValid() {
			return fmt.Errorf("got unallowed key '%s'", key)
		}

		var val reflect.Value

		switch field.Type() {
		case stringType:
			val = reflect.ValueOf(str)

		case intType:
			i, err := strconv.Atoi(str)
			if err != nil {
				return err
			}
			val = reflect.ValueOf(i)

		case uint64Type:
			i, err := strconv.ParseUint(str, 10, 64)
			if err != nil {
				return err
			}
			val = reflect.ValueOf(i)

		case boolType:
			switch str {
			case "yes", "y", "true", "1":
				val = reflect.ValueOf(true)
			case "no", "n", "false", "0":
				val = reflect.ValueOf(false)
			default:
				return fmt.Errorf("Unknown value for boolean: '%s'", str)
			}

		case byteSizeType:
			v, err := bytesize.Parse(str)
			if err != nil {
				return err
			}
			val = reflect.ValueOf(v)

		default:
			return fmt.Errorf("Cannot handle type '%s'", field.Type().String())
		}

		field.Set(val)
	}

	return nil
}
