package sys

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

func GetCpuUsage() (float64, error) {
	v, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, err
	}
	return v[0], nil
}

func GetMemUsage() (total, used uint64, err error) {
	var v *mem.VirtualMemoryStat

	v, err = mem.VirtualMemory()
	if err != nil {
		return
	}

	total = v.Total
	used = v.Used
	return
}

func GetDiskUsage(path string) (total, used uint64, err error) {
	var v *disk.UsageStat

	v, err = disk.Usage(path)
	if err != nil {
		return
	}

	total = v.Total
	used = v.Used
	return
}
