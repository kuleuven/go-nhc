package main

import (
    "fmt"
    "os"
    "strconv"
    "strings"
    "io/ioutil"
    "path/filepath"
)

const (
    memory_channels_folder = "/sys/devices/system/edac/mc"
)

type MemoryChannel struct {
    Name  string
    Dimms []Dimm
}
type Dimm struct {
    Name string
    Size uint64
}

func ListMemoryChannels() ([]MemoryChannel, error) {
    files, err := filepath.Glob(fmt.Sprintf("%s/mc*", memory_channels_folder))
    if err != nil {
        return nil, err
    }

    result := make([]MemoryChannel, 0, len(files))
    for _, file := range files {
        name := filepath.Base(file)
        dimms, err := ListDimms(name)
        if err != nil {
            return nil, err
        }
        result = append(result, MemoryChannel{
            Name: name,
            Dimms: dimms,
        })
    }

    return result, nil
}

func ListDimms(mc string) ([]Dimm, error) {
    files, err := filepath.Glob(fmt.Sprintf("%s/%s/dimm*", memory_channels_folder, mc))
    if err != nil {
        return nil, err
    }

    result := make([]Dimm, 0, len(files))
    for _, file := range files {
        name := filepath.Base(file)

        handle, err := os.Open(fmt.Sprintf("%s/size", file))
        defer handle.Close()
        if err != nil {
            return nil, err
        }

        b, err := ioutil.ReadAll(handle)
        if err != nil {
            return nil, err
        }

        i, err := strconv.ParseUint(strings.TrimSuffix(string(b), "\n"), 10, 64)
        if err != nil {
            return nil, err
        }

        result = append(result, Dimm{
            Name: name,
            Size: i,
        })
    }

    return result, nil
}
