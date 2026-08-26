package util

import "syscall"

const DiskFloorBytes uint64 = 30 * 1024 * 1024 * 1024

func GetFreeSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}
