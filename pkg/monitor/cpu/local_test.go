package cpu

import (
	"os"
	"testing"
)

func TestCPU(t *testing.T) {
	mon := NewLocalCPUMonitor(int32(os.Getpid()))

	v, err := mon.Read()
	if err != nil {
		t.Fatal(err)
	}

	if v <= 0 {
		t.Fatal("CPU percentage less than or equal to zero, that's weird")
	}
	t.Log("read CPU percentage:", v)
}