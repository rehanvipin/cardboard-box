package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
const tagFile = "tags.json"

func main() {
	if os.Geteuid() != 0 {
		fmt.Println("Please run as root to avoid problems")
		return
	}
	switch os.Args[1] {
	case "run":
		// Create and execute command on a temporary container
		container, _ := create()
		run(container, os.Args[2:], true)
		fmt.Println("Contained")
	case "child":
		// Required for run, cannot change hostname without being container
		child()
	case "create":
		// Create a container but do not execute, user can name it if they want
		register()
	case "start":
		// Execute a command on an existing container, exit if doesn't exist
		if len(os.Args) < 4 {
			fmt.Println("Usage: [cbox] start [box-name] [command]")
		}
		start()
	case "delete":
		// Delete a particular container
		deleteContainer()
	case "list":
		list()
	default:
		panic("Wrong usage. Use 'run' as argument")
	}
}

func list() {
	home, homerr := os.UserHomeDir()
	safeExec(homerr)
	workDir := path.Join(home, ".cbox")
	tagLoc := path.Join(workDir, tagFile)

	var locations = make(map[string]string)
	data, err := ioutil.ReadFile(tagLoc)
	safeExec(err)
	safeExec(json.Unmarshal(data, &locations))

	// Loop through locations and print the keys
	fmt.Println("Containers in storage:")
	fmt.Println("--------")
	for k := range locations {
		fmt.Println(k)
	}
	fmt.Println("--------")
}

func deleteContainer() {
	home, homerr := os.UserHomeDir()
	safeExec(homerr)
	workDir := path.Join(home, ".cbox")
	tagLoc := path.Join(workDir, tagFile)

	var locations = make(map[string]string)
	data, err := ioutil.ReadFile(tagLoc)
	safeExec(err)
	safeExec(json.Unmarshal(data, &locations))

	for i := range os.Args[2:] {
		containerName := os.Args[2+i]
		if _, ok := locations[containerName]; !ok {
			fmt.Printf("A container with the name %s does not exist\n", containerName)
			continue
		}
		container := locations[containerName]
		safeExec(os.RemoveAll(container))
		delete(locations, containerName)
		fmt.Println("Deleted", containerName)
	}

	encoded, _ := json.Marshal(locations)
	safeExec(ioutil.WriteFile(tagLoc, encoded, 0644))
	fmt.Println("Sucessfully deleted container(s)")
}

func start() {
	// Commands and tag-name
	tagName := os.Args[2]
	// Load json file to check if tag exists
	home, homerr := os.UserHomeDir()
	safeExec(homerr)
	workDir := path.Join(home, ".cbox")
	tagLoc := path.Join(workDir, tagFile)

	var location = make(map[string]string)
	data, err := ioutil.ReadFile(tagLoc)
	safeExec(err)
	safeExec(json.Unmarshal(data, &location))

	boxPath, ok := location[tagName]
	if !ok {
		fmt.Printf("The container with the name %v does not exist.\n", tagName)
		return
	}

	run(boxPath, os.Args[3:], false)
}

func register() {
	// Safety check
	safeExec(fetch())

	// Save it to the json file
	home, homerr := os.UserHomeDir()
	safeExec(homerr)
	workDir := path.Join(home, ".cbox")
	tagLoc := path.Join(workDir, tagFile)

	// Load and update json data
	var locations = make(map[string]string)
	if _, err := os.Stat(tagLoc); os.IsNotExist(err) {
		fmt.Println("Could not find the file")
		file, _ := os.Create(tagLoc)
		file.Write([]byte("{}"))
		file.Close()
	}
	data, err := ioutil.ReadFile(tagLoc)
	safeExec(err)
	safeExec(json.Unmarshal(data, &locations))

	if len(os.Args) == 3 {
		if _, ok := locations[os.Args[2]]; ok {
			fmt.Println("Tag already exists")
			return
		}
	}

	// Check args for custom container name
	container, _ := create()
	containerSplit := strings.Split(container, "/")
	containerTag := containerSplit[len(containerSplit)-1]
	// fmt.Printf("The new container is %v \n", containerTag)
	if len(os.Args) == 3 {
		containerTag = os.Args[2]
	}

	locations[containerTag] = container
	encoded, _ := json.Marshal(locations)

	safeExec(ioutil.WriteFile(tagLoc, encoded, 0644))
}

func run(containerLoc string, args []string, temporary bool) {
	fetcherr := fetch()
	safeExec(fetcherr)

	// Delete container after execution or no?
	var delContainer string
	if temporary {
		delContainer = "delete"
	} else {
		delContainer = "preserve"
	}

	// Run the child process with new namespaces
	cmd := exec.Command("/proc/self/exe",
		append([]string{"child", containerLoc, delContainer}, args...)...)
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
	container := os.Args[2]
	temporary := os.Args[3]
	// containerSplit := strings.Split(container, "/")
	// containerTag := containerSplit[len(containerSplit)-1]

	cmd := exec.Command(os.Args[4], os.Args[5:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set properties
	safeExec(unix.Sethostname([]byte("container")))
	exit, chrootError := Chroot(container)
	safeExec(chrootError)
	safeExec(unix.Chdir("/"))
	safeExec(unix.Mount("proc", "proc", "proc", 0, ""))

	safeExec(cmd.Run())

	// Clean up
	safeExec(unix.Unmount("/proc", 0))
	// Exit the chroot, cannot delete a directory in use
	safeExec(exit())

	if temporary == "delete" {
		safeExec(os.RemoveAll(container))
	}
}

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

	// Announce its presence
	fmt.Printf("The new container is %v \n", containerName)

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
