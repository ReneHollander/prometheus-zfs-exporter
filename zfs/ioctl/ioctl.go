// Package ioctl provides a pure-Go low-level wrapper around ZFS's ioctl interface and basic wrappers around common
// ioctls to make them usable from normal Go code.
package ioctl

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

type ZFSHandle struct {
	zfsHandle *os.File
}

func NewZFSHandleWithPath(path string) (*ZFSHandle, error) {
	zfsHandle, err := os.Open(path)
	if os.IsNotExist(err) {
		unix.Mknod(path, 666, int(unix.Mkdev(10, 54)))
	}
	zfsHandle, err = os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to open or create ZFS device node: %v", err)
	}
	return &ZFSHandle{
		zfsHandle: zfsHandle,
	}, nil
}

func NewZFSHandle() (*ZFSHandle, error) {
	return NewZFSHandleWithPath("/dev/zfs")
}

// ZfsIoctl issues a low-level ioctl syscall with only some common wrappers. All unsafety is contained in here.
func (h *ZFSHandle) Ioctl(ioctl Ioctl, cmd *Cmd, request []byte, config []byte, resp *[]byte) error {
	for {
		// WARNING: Here be dragons! This is completely outside of Go's safety net and uses various
		// criticial runtime workarounds to make sure that memory is safely handled
		if resp != nil && *resp != nil {
			cmd.Nvlist_dst = uint64(uintptr(unsafe.Pointer(&(*resp)[0])))
			cmd.Nvlist_dst_size = uint64(len(*resp))
		}
		if request != nil {
			cmd.Nvlist_src = uint64(uintptr(unsafe.Pointer(&request[0])))
			cmd.Nvlist_src_size = uint64(len(request))
		}
		if config != nil {
			cmd.Nvlist_conf = uint64(uintptr(unsafe.Pointer(&config[0])))
			cmd.Nvlist_conf_size = uint64(len(config))
		}
		_, _, errno := unix.Syscall(unix.SYS_IOCTL, h.zfsHandle.Fd(), uintptr(ioctl), uintptr(unsafe.Pointer(cmd)))
		if request != nil {
			runtime.KeepAlive(request)
		}
		if config != nil {
			runtime.KeepAlive(config)
		}
		if resp != nil && *resp != nil {
			runtime.KeepAlive(*resp)
		}
		runtime.KeepAlive(cmd)
		if errno == unix.ENOMEM && resp != nil && *resp != nil {
			requiredLength := cmd.Nvlist_dst_size
			*resp = make([]byte, requiredLength)
			continue
		}
		if errno != 0 {
			return errno
		}
		break
	}
	return nil
}

func (c *Cmd) Clear() {
	*c = *(&Cmd{})
}

func (c *Cmd) SetName(name string) {
	stringToDelimitedBuf(name, c.Name[:])
}

func (c *Cmd) GetName() string {
	return delimitedBufToString(c.Name[:])
}

func delimitedBufToString(buf []byte) string {
	i := 0
	for ; i < len(buf); i++ {
		if buf[i] == 0x00 {
			break
		}
	}
	return string(buf[:i])
}

func stringToDelimitedBuf(str string, buf []byte) error {
	if len(str) > len(buf)-1 {
		return fmt.Errorf("String longer than target buffer (%v > %v)", len(str), len(buf)-1)
	}
	for i := 0; i < len(str); i++ {
		if str[i] == 0x00 {
			return errors.New("String contains null byte, this is unsupported by ZFS")
		}
		buf[i] = str[i]
	}
	return nil
}
