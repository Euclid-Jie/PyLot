package service

import (
	"errors"
	"fmt"
	"math/bits"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	afInet                    = 2
	afInet6                   = 23
	tcpTableOwnerPIDListener  = 3
	errorInsufficientBuffer   = syscall.Errno(122)
	maxProcessImagePathLength = 32768
)

var getExtendedTCPTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

type PortOwner struct {
	PID         int    `json:"pid"`
	ProcessName string `json:"process_name"`
	ProcessPath string `json:"process_path"`
}

type mibTCPRowOwnerPID struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPID  uint32
}

type mibTCP6RowOwnerPID struct {
	LocalAddr     [16]byte
	LocalScopeID  uint32
	LocalPort     uint32
	RemoteAddr    [16]byte
	RemoteScopeID uint32
	RemotePort    uint32
	State         uint32
	OwningPID     uint32
}

func FindPortOwner(port int) (*PortOwner, error) {
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", port)
	}

	pid, err := findListenerPID(port, afInet)
	if err != nil {
		return nil, err
	}
	if pid == 0 {
		pid, err = findListenerPID(port, afInet6)
		if err != nil {
			return nil, err
		}
	}
	if pid == 0 {
		return nil, nil
	}

	name, path := processInfo(pid)
	return &PortOwner{PID: pid, ProcessName: name, ProcessPath: path}, nil
}

func KillPortOwner(port, expectedPID int) error {
	owner, err := FindPortOwner(port)
	if err != nil {
		return err
	}
	if owner == nil {
		return nil
	}
	if owner.PID != expectedPID {
		return fmt.Errorf("端口 %d 的占用进程已变化（当前 PID %d）", port, owner.PID)
	}
	if expectedPID <= 4 {
		return fmt.Errorf("不能结束系统进程 PID %d", expectedPID)
	}
	if err := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", expectedPID)).Run(); err != nil {
		return fmt.Errorf("结束 PID %d 失败: %w", expectedPID, err)
	}
	return WaitPortFree(port, 5*time.Second)
}

func WaitPortFree(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		owner, err := FindPortOwner(port)
		if err != nil {
			return err
		}
		if owner == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("端口 %d 未在 %s 内释放，仍由 PID %d 占用", port, timeout, owner.PID)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func findListenerPID(port, addressFamily int) (int, error) {
	var size uint32
	r1, _, _ := getExtendedTCPTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		1,
		uintptr(addressFamily),
		tcpTableOwnerPIDListener,
		0,
	)
	if syscall.Errno(r1) != errorInsufficientBuffer && r1 != 0 {
		return 0, fmt.Errorf("query TCP table size: %w", syscall.Errno(r1))
	}

	buf := make([]byte, size)
	r1, _, _ = getExtendedTCPTable.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		1,
		uintptr(addressFamily),
		tcpTableOwnerPIDListener,
		0,
	)
	if r1 != 0 {
		return 0, fmt.Errorf("query TCP table: %w", syscall.Errno(r1))
	}

	count := *(*uint32)(unsafe.Pointer(&buf[0]))
	base := unsafe.Pointer(&buf[4])
	if addressFamily == afInet {
		rows := unsafe.Slice((*mibTCPRowOwnerPID)(base), int(count))
		for _, row := range rows {
			if tcpPort(row.LocalPort) == port {
				return int(row.OwningPID), nil
			}
		}
		return 0, nil
	}

	rows := unsafe.Slice((*mibTCP6RowOwnerPID)(base), int(count))
	for _, row := range rows {
		if tcpPort(row.LocalPort) == port {
			return int(row.OwningPID), nil
		}
	}
	return 0, nil
}

func tcpPort(raw uint32) int {
	return int(bits.ReverseBytes16(uint16(raw)))
}

func processInfo(pid int) (string, string) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err == nil {
		defer windows.CloseHandle(h)
		buf := make([]uint16, maxProcessImagePathLength)
		size := uint32(len(buf))
		if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err == nil {
			path := windows.UTF16ToString(buf[:size])
			return filepath.Base(path), path
		}
	}

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return "", ""
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return "", ""
	}
	for {
		if int(entry.ProcessID) == pid {
			return windows.UTF16ToString(entry.ExeFile[:]), ""
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			if errors.Is(err, syscall.ERROR_NO_MORE_FILES) {
				break
			}
			break
		}
	}
	return "", ""
}
