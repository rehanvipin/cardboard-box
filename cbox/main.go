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

const basefs = "http://cdimage.ubuntu.com/ubuntu-base/releases/16.04/release/ubuntu-base-16.04.6-base-amd64.tar.gz"
const dirNameLen = 8
const image = "ubuntu16fs.tar.gz"

func main() {
	switch os.Args[1] {
	case "run":
		run()
		fmt.Println("Contained")
	case "child":
		child()
	default:
		panic("Wrong usage. Use 'run' as argument")
	}
}

func run() {
	fetcherr := fetch()
	safeExec(fetcherr)

	// Run the child process with new namespaces
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &unix.SysProcAttr{
		Cloneflags:   unix.CLONE_NEWUTS | unix.CLONE_NEWPID | unix.CLONE_NEWNS,
		Unshareflags: unix.CLONE_NEWNS,
	}

	safeExec(cmd.Run())
}

func child() {
	// Create new chroot rootfs
	container, _ := create()
	containerSplit := strings.Split(container, "/")
	containerTag := containerSplit[len(containerSplit)-1]
	fmt.Printf("The new container is %v \n", containerTag)

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set properties
	safeExec(unix.Sethostname([]byte("container")))
	safeExec(unix.Chroot(container))
	safeExec(unix.Chdir("/"))
	safeExec(unix.Mount("proc", "proc", "proc", 0, ""))

	safeExec(cmd.Run())

	// Clean up
	safeExec(unix.Unmount("/proc", 0))
	safeExec(os.RemoveAll(container))
}

// create makes a temporary root fs directory
// returns the path to the directory
func create() (string, error) {
	// Unique name for new container root
	containerName := randRoot(dirNameLen)
	home, homerr := os.UserHomeDir()
	safeExec(homerr)
	workDir := path.Join(home, ".cbox")
	// The path where the fs image is stored
	// FQCN -> fully quailified container name
	FQCN := path.Join(workDir, containerName)
	// Create empty container dir
	mkerr := os.MkdirAll(FQCN, 0755)
	safeExec(mkerr)

	imagePath := path.Join(workDir, image)
	// move files from image into directory
	untarerr := untar(imagePath, FQCN)
	safeExec(untarerr)

	return FQCN, nil
}

// randRoot gives a random directory name for the container root
func randRoot(n int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("abcdefghijklmnopqrstuvwxyz1234567890")
	c := len(chars)
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteRune(chars[rand.Intn(c)])
	}
	return b.String()
}

// untar extracts the image into the container root
func untar(src, dest string) error {
	cmd := exec.Command("tar", "-xf", src, "-C", dest, "--exclude=dev")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// fetch downloads the base file-system image if necessary
func fetch() error {
	// Create directory if necessary ~/.cbox
	home, homerr := os.UserHomeDir()
	safeExec(homerr)
	workDir := path.Join(home, ".cbox")
	mkerr := os.MkdirAll(workDir, 0755)
	safeExec(mkerr)
	// The path where the fs image is stored
	fsStore := path.Join(workDir, image)

	// Skip process if file exists
	if fileExists(fsStore) {
		return nil
	}

	fmt.Println("Need to fetch the file-system image once")
	cmd := exec.Command("curl", basefs, "-o", fsStore)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

// fileExists check if a file with that name exists
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func safeExec(e error) {
	if e != nil {
		panic(e)
	}
}
