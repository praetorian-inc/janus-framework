package util

import (
	"os/exec"
)

func EmptyChannel[T any](in <-chan T) {
	for range in {
	}
}

func CheckBinaryExists(binaryName string) bool {
	_, err := exec.LookPath(binaryName)
	return err == nil
}
