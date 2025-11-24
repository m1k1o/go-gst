package gst

import (
	"fmt"
	"runtime"
	"runtime/pprof"
	"unsafe"
	"sync"
	"time"
)

var padprobesProfile *pprof.Profile
var gstGCProfile *pprof.Profile

var aliveObjects = make(map[string]int)
var aliveObjectsMu sync.RWMutex

func init() {
	padprobes := "go-gst-active-pad-probes"
	padprobesProfile = pprof.Lookup(padprobes)
	if padprobesProfile == nil {
		padprobesProfile = pprof.NewProfile(padprobes)
	}

	gstGC := "go-gst-gst-gc"
	gstGCProfile = pprof.Lookup(gstGC)
	if gstGCProfile == nil {
		gstGCProfile = pprof.NewProfile(gstGC)
	}

	go func() {
		for {
			<-time.After(1 * time.Second)
			aliveObjectsMu.RLock()
			fmt.Printf("GST: alive objects:\n")
			for name, count := range aliveObjects {
				fmt.Printf("GST:   %s: %d\n", name, count)
			}
			aliveObjectsMu.RUnlock()
		}
	}()
}


func WrapFinalizer[T any](name string, obj *T, finalizer func(*T)) {
	aliveObjectsMu.Lock()
	aliveObjects[name]++
	gstGCProfile.Add(uintptr(unsafe.Pointer(obj)), 1)
	aliveObjectsMu.Unlock()
	//fmt.Printf("FINALIZER[%s]: setting up\n", name)
	runtime.SetFinalizer(obj, func(o *T) {
		aliveObjectsMu.Lock()
		aliveObjects[name]--
		gstGCProfile.Remove(uintptr(unsafe.Pointer(o)))
		aliveObjectsMu.Unlock()
		//fmt.Printf("FINALIZER[%s]: running, alive objects: %d\n", name, aliveObjects[name])
		finalizer(o)
	})
}
