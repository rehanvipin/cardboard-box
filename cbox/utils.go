package main

import (
	"fmt"
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

// Help prints out the usage message for the executable
func Help() {
	fmt.Println("Usage: cbox [command] [subcommand / args]")
	fmt.Println("Command can be one of: [run|create|start|list|delete]")
	fmt.Println("run - cbox run [absolute path command to run on container]")
	fmt.Println("create - create new permenant container, args: optional [box-name]")
	fmt.Println("start - run command on already created container, args: [command]")
	fmt.Println("list - list all created containers")
	fmt.Println("delete - delete containers from list + disk, args: [box1-name box2-name ...]")
}
