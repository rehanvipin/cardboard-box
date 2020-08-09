package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const basefs = "http://cdimage.ubuntu.com/ubuntu-base/releases/16.04/release/ubuntu-base-16.04.6-base-amd64.tar.gz"
const dirNameLen = 8
const image = "ubuntu16fs.tar.gz"
const tagFile = "tags.json"

// Credential is a replacement for unix.Credential
type Credential struct {
	UID         uint32   // User ID.
	GID         uint32   // Group ID.
	Groups      []uint32 // Supplementary group IDs.
	NoSetGroups bool     // If true, don't set supplementary groups
}

func main() {
	// if os.Geteuid() != 0 {
	// 	fmt.Println("Must run as root. Quitting...")
	// 	return
	// }
	if len(os.Args) < 2 {
		Help()
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
		Help()
	}
}

func list() {
	workDir := WorkingDir()
	tagLoc := path.Join(workDir, tagFile)

	if !FileExists(tagLoc) {
		safeExec(fetch())
		file, _ := os.Create(tagLoc)
		file.Write([]byte("{}"))
		file.Close()
	}

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
	workDir := WorkingDir()
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
	workDir := WorkingDir()
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
	workDir := WorkingDir()
	tagLoc := path.Join(workDir, tagFile)

	// Load and update json data
	var locations = make(map[string]string)
	if !FileExists(tagLoc) {
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

	var runAsRoot string
	if os.Geteuid() == 0 {
		runAsRoot = "true"
	} else {
		runAsRoot = "false"
	}

	// Run the child process with new namespaces
	cmd := exec.Command("/proc/self/exe",
		append([]string{"child", containerLoc, runAsRoot}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &unix.SysProcAttr{
		Cloneflags:   unix.CLONE_NEWUTS | unix.CLONE_NEWPID | unix.CLONE_NEWNS | unix.CLONE_NEWUSER,
		Unshareflags: unix.CLONE_NEWNS,
		Credential:   &syscall.Credential{Uid: 0, Gid: 0},
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
	}

	safeExec(cmd.Run())

	if temporary {
		safeExec(os.RemoveAll(containerLoc))
	}
}

func child() {
	// Create new chroot rootfs
	container := os.Args[2]
	runAsRoot := os.Args[3]
	// containerSplit := strings.Split(container, "/")
	// containerTag := containerSplit[len(containerSplit)-1]

	cmd := exec.Command(os.Args[4], os.Args[5:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Cgroups only work as root
	if runAsRoot == "true" {
		cg()
	}

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
}

// create makes a temporary root fs directory
// returns the path to the directory
func create() (string, error) {
	safeExec(fetch())
	// Unique name for new container root
	containerName := RandRoot(dirNameLen)
	workDir := WorkingDir()
	// The path where the fs image is stored
	// FQCN -> fully quailified container name
	FQCN := path.Join(workDir, containerName)
	// Create empty container dir
	mkerr := os.MkdirAll(FQCN, 0755)
	safeExec(mkerr)

	imagePath := path.Join(workDir, image)
	// move files from image into directory
	untarerr := Untar(imagePath, FQCN)
	safeExec(untarerr)

	// Announce its presence
	fmt.Printf("The new container is %v \n", containerName)

	return FQCN, nil
}

// fetch downloads the base file-system image if necessary
func fetch() error {
	// Create directory if necessary ~/.cbox
	workDir := WorkingDir()
	mkerr := os.MkdirAll(workDir, 0755)
	safeExec(mkerr)
	// The path where the fs image is stored
	fsStore := path.Join(workDir, image)

	// Skip process if file exists
	if FileExists(fsStore) {
		return nil
	}

	fmt.Println("Need to fetch the file-system image once")
	cmd := exec.Command("curl", basefs, "-o", fsStore)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func safeExec(e error) {
	if e != nil {
		panic(e)
	}
}

func cg() {
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")
	os.Mkdir(filepath.Join(pids, "cbox"), 0755)
	safeExec(ioutil.WriteFile(filepath.Join(pids, "cbox/pids.max"), []byte("20"), 0700))
	// Removes the new cgroup in place after the container exits
	safeExec(ioutil.WriteFile(filepath.Join(pids, "cbox/notify_on_release"), []byte("1"), 0700))
	safeExec(ioutil.WriteFile(filepath.Join(pids, "cbox/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}
