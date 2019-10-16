// +build linux

package utils

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

const (
	mom_priv_jobs = "/var/spool/torque/mom_priv/jobs"
)

type Job struct {
	Path string
	Name string
	Uid  uint64
	Gid  uint64
}

func ListPBSJobs() (jobs []Job, err error) {
	err = filepath.Walk(mom_priv_jobs, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".JB" {
			name := filepath.Base(path)
			name = name[:len(name)-3]

			uid, gid, err := getJobIds(path)
			if err != nil {
				return err
			}

			job := Job{Path: path, Name: name, Uid: uid, Gid: gid}
			jobs = append(jobs, job)
		}
		return nil
	})
	return
}

type Jobxml struct {
	XMLName xml.Name `xml:"job"`
	Uid     string   `xml:"execution_uid"`
	Gid     string   `xml:"execution_gid"`
}

func getJobIds(path string) (uid uint64, gid uint64, err error) {
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		return
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	job := Jobxml{}
	err = xml.Unmarshal(data, &job)
	if err != nil {
		return
	}

	uid, err = strconv.ParseUint(job.Uid, 10, 64)
	if err != nil {
		return
	}

	gid, err = strconv.ParseUint(job.Gid, 10, 64)
	if err != nil {
		return
	}

	return
}
