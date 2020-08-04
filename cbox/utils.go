package main

import (
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// Chroot that exits too, to delete directory
func Chroot(path string) (func() error, error) {
	root, err := os.Open("/")
	safeExec(err)

	if err := unix.Chroot(path); err != nil {
		root.Close()
		return nil, err
	}

	return func() error {
		defer root.Close()
		if err := root.Chdir(); err != nil {
			return err
		}
		return unix.Chroot(".")
	}, nil
}

// RandRoot gives a random directory name for the container root
func RandRoot(n int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("abcdefghijklmnopqrstuvwxyz1234567890")
	c := len(chars)
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteRune(chars[rand.Intn(c)])
	}
	return b.String()
}

// Untar extracts the image into the container root
func Untar(src, dest string) error {
	cmd := exec.Command("tar", "-xf", src, "-C", dest, "--exclude=dev")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// FileExists check if a file with that name exists
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// WorkingDir Returns the full path of the .cbox directory
func WorkingDir() string {
	home, homerr := os.UserHomeDir()
	safeExec(homerr)
	return path.Join(home, ".cbox")
}
