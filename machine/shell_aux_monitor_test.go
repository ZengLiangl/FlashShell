package machine

import (
	"math"
	"testing"
)

func TestParseCPUPercentPlainNumber(t *testing.T) {
	got := parseCPUPercent("12.3")
	if math.Abs(got-12.3) > 0.01 {
		t.Fatalf("got %v, want 12.3", got)
	}
	if parseCPUPercent("150") != 100 {
		t.Fatalf("should clamp to 100")
	}
	if parseCPUPercent("-1") != 0 {
		t.Fatalf("should clamp to 0")
	}
}

func TestParseCPUPercentLegacyTopLine(t *testing.T) {
	line := "%Cpu(s):  3.2 us,  1.1 sy,  0.0 ni, 95.5 id,  0.2 wa"
	got := parseCPUPercent(line)
	if math.Abs(got-4.5) > 0.01 {
		t.Fatalf("got %v, want 4.5 (100-95.5)", got)
	}
}

func TestParseTopProcessesInstantRows(t *testing.T) {
	block := `31522 root 2.1 4.1 java
4368 root 1.8 2.0 mongod
3880 root 0.8 8.5 java`
	got := parseTopProcesses(block, 5)
	if len(got) != 3 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].PID != "31522" || math.Abs(got[0].CPU-2.1) > 0.01 || math.Abs(got[0].Mem-4.1) > 0.01 {
		t.Fatalf("row0=%+v", got[0])
	}
	if got[0].Command != "java" {
		t.Fatalf("cmd=%q", got[0].Command)
	}
}

func TestParseTopBatchSecondFrame(t *testing.T) {
	raw := `
top - 22:00:00 up 1 day,  1:00,  1 user,  load average: 0.00, 0.01, 0.05
Tasks: 100 total,   1 running,  99 sleeping,   0 stopped,   0 zombie
%Cpu(s):  0.5 us,  0.2 sy,  0.0 ni, 99.3 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st
MiB Mem :  15800.0 total,   7000.0 free,   8000.0 used,    800.0 buff/cache
MiB Swap:      0.0 total,      0.0 free,      0.0 used.   7000.0 avail Mem

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
    1 root      20   0  169416  11420   8420 S   0.0   0.1   0:01.23 systemd
  100 root      20   0  999999  99999   9999 S  50.0   9.9   1:00.00 busy

top - 22:00:00 up 1 day,  1:00,  1 user,  load average: 0.00, 0.01, 0.05
Tasks: 100 total,   1 running,  99 sleeping,   0 stopped,   0 zombie
%Cpu(s):  3.0 us,  1.0 sy,  0.0 ni, 95.5 id,  0.5 wa,  0.0 hi,  0.0 si,  0.0 st
MiB Mem :  15800.0 total,   7000.0 free,   8000.0 used,    800.0 buff/cache
MiB Swap:      0.0 total,      0.0 free,      0.0 used.   7000.0 avail Mem

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
31522 root      20   0  222222  33333   4444 S  12.5   4.1   9:99.99 java
 4368 root      20   0  111111  22222   3333 S   1.8   2.0   1:11.11 mongod
 3880 root      20   0  111111  22222   3333 S   0.8   8.5   1:11.11 java
`
	sysCPU, procs := parseTopBatch(raw, 5)
	if math.Abs(sysCPU-4.5) > 0.01 {
		t.Fatalf("sysCPU=%v, want 4.5", sysCPU)
	}
	if len(procs) < 3 {
		t.Fatalf("procs=%d %+v", len(procs), procs)
	}
	if procs[0].PID != "31522" || math.Abs(procs[0].CPU-12.5) > 0.01 || math.Abs(procs[0].Mem-4.1) > 0.01 {
		t.Fatalf("row0=%+v", procs[0])
	}
	if procs[0].Command != "java" {
		t.Fatalf("cmd=%q", procs[0].Command)
	}
	if procs[1].PID != "4368" || math.Abs(procs[1].CPU-1.8) > 0.01 {
		t.Fatalf("row1=%+v", procs[1])
	}
}
