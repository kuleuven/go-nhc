// +build linux

package utils

import (
	"syscall"
)

type DiskStatus struct {
	Blocks     uint64 `json:"blocks"`
	BlocksUsed uint64 `json:"blocks_used"`
	BlocksFree uint64 `json:"blocks_free"`
	Inodes     uint64 `json:"inodes"`
	InodesUsed uint64 `json:"inodes_used"`
	InodesFree uint64 `json:"inodes_free"`
}

func DiskUsage(path string) (disk DiskStatus, err error) {
	fs := syscall.Statfs_t{}
	err = syscall.Statfs(path, &fs)
	if err != nil {
		return
	}
	disk.Blocks = fs.Blocks * uint64(fs.Bsize)
	disk.BlocksFree = fs.Bfree * uint64(fs.Bsize)
	disk.BlocksUsed = disk.Blocks - disk.BlocksFree

	disk.Inodes = fs.Files
	disk.InodesFree = fs.Ffree
	disk.InodesUsed = disk.Inodes - disk.InodesFree
	return
}
