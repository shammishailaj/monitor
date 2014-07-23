package cpu

import (
	"sync"
	"time"
	"unsafe"
)

/*
#include <mach/mach_init.h>
#include <mach/mach_error.h>
#include <mach/mach_host.h>
#include <mach/mach_port.h>
#include <mach/host_info.h>
*/
import "C"

type CPU struct {
	start         sync.Once
	previous      map[string]int
	current       map[string]int
	currentTotal  int
	previousTotal int
	lastUpdate    time.Time
}

func New() *CPU {
	return &CPU{}
}

func (c *CPU) User() *metric {
	return newMetric(c, "user")
}

func (c *CPU) System() *metric {
	return newMetric(c, "system")
}

func (c *CPU) Idle() *metric {
	return newMetric(c, "idle")
}

func (c *CPU) run(step time.Duration) {
	c.start.Do(func() {
		for _ = range time.NewTicker(step).C {
			c.collect()
		}
	})
}

func (c *CPU) collect() {
	c.lastUpdate = time.Now()
	c.previous = c.current

	// collect CPU stats for All cpus aggregated
	var cpuinfo C.host_cpu_load_info_data_t
	count := C.mach_msg_type_number_t(C.HOST_CPU_LOAD_INFO_COUNT)
	host := C.mach_host_self()

	ret := C.host_statistics(C.host_t(host), C.HOST_CPU_LOAD_INFO,
		C.host_info_t(unsafe.Pointer(&cpuinfo)), &count)

	if ret != C.KERN_SUCCESS {
		return
	}

	c.current = map[string]int{
		"user":   int(cpuinfo.cpu_ticks[C.CPU_STATE_USER]),
		"nice":   int(cpuinfo.cpu_ticks[C.CPU_STATE_NICE]),
		"system": int(cpuinfo.cpu_ticks[C.CPU_STATE_SYSTEM]),
		"idle":   int(cpuinfo.cpu_ticks[C.CPU_STATE_IDLE]),
	}

	if c.previous == nil {
		c.previous = c.current
	}

	c.previousTotal = c.currentTotal
	c.currentTotal = c.current["user"] + c.current["nice"] + c.current["system"] + c.current["idle"]
}

func (c *CPU) rate(name string) float64 {
	delta := c.current[name] - c.previous[name]
	total := c.currentTotal - c.previousTotal

	return float64(delta) / float64(total) * 100
}
