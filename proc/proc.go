package proc

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/cubbitgg/oom-tracker/mem"
)

var (
	// WarningSignal is the signal sent to the process once we reach what
	// is considered a Warning threshold.
	WarningSignal = syscall.SIGUSR1

	// CriticalSignal is the signal sent to the process once we reach what
	// is considered a Critical threshold.
	CriticalSignal = syscall.SIGUSR2

	supportedSignals = map[string]syscall.Signal{
		"SIGABRT": syscall.SIGABRT,
		"SIGCONT": syscall.SIGCONT,
		"SIGHUP":  syscall.SIGHUP,
		"SIGINT":  syscall.SIGINT,
		"SIGIOT":  syscall.SIGIOT,
		"SIGKILL": syscall.SIGKILL,
		"SIGQUIT": syscall.SIGQUIT,
		"SIGSTOP": syscall.SIGSTOP,
		"SIGTERM": syscall.SIGTERM,
		"SIGTSTP": syscall.SIGTSTP,
		"SIGUSR1": syscall.SIGUSR1,
		"SIGUSR2": syscall.SIGUSR2,
	}
)

// Abstraction over process
type Process interface {
	Pid() int
	Signal(os.Signal) error
	MemoryUsagePercent() (uint64, error)
	NumaStat() (string, error)
}

// CmdLine returns the command line for proc.
func CmdLine(proc Process) (string, error) {
	cmdFile := fmt.Sprintf("/proc/%d/cmdline", proc.Pid())
	cmdAsB, err := os.ReadFile(cmdFile)
	if err != nil {
		return "", err
	}
	cmdAsStr := strings.TrimSuffix(string(cmdAsB), "\n")
	return cmdAsStr, nil
}

// Others return a list of all other processes running on the system, excluding
// the current one.
func Others() ([]*os.Process, error) {
	files, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	ps := make([]*os.Process, 0)
	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(file.Name())
		if err != nil {
			continue
		}

		if pid == os.Getpid() {
			continue
		}

		proccess, err := os.FindProcess(pid)
		if err != nil {
			return nil, err
		}

		ps = append(ps, proccess)
	}

	if len(ps) == 0 {
		return nil, fmt.Errorf("unable to find any process")
	}

	return ps, nil
}

func PrintWarningFor(p Process) error {
	pct, error := p.MemoryUsagePercent()
	if error != nil {
		return error
	}
	log.Printf("=== Warining memory usage on pid %d's cgroup: %d%% ===", p.Pid(), pct)
	return printNuma(p)
}

func printNuma(p Process) error {
	stat, error := p.NumaStat()
	if error != nil {
		return error
	}
	log.Printf("numa:\n%s\n", stat)
	return nil
}

func PrintCriticalFor(p Process) error {
	pct, error := p.MemoryUsagePercent()
	if error != nil {
		return error
	}
	log.Printf("=== Critical memory usage on pid %d's cgroup: %d%% ===", p.Pid(), pct)
	return printNuma(p)

}

type OsProcess struct {
	process *os.Process
}

func NewOsProcess(p *os.Process) OsProcess {
	return OsProcess{
		process: p,
	}
}

func (p OsProcess) Pid() int {
	return p.process.Pid
}

func (p OsProcess) Signal(s os.Signal) error {
	return p.process.Signal(s)
}

func (p OsProcess) MemoryUsagePercent() (uint64, error) {
	limit, usage, err := mem.LimitAndUsageForProc(p.process)
	if err != nil {
		return 0, err
	} else if limit == 0 {
		return 0, nil
	}
	return (usage * 100) / limit, nil
}

func (p OsProcess) NumaStat() (string, error) {
	stat, err := mem.NumaStatForProc(p.process)
	if err != nil {
		return "", err
	}
	return stat, nil
}
