# Cardboard-box for when you want to teleport away

Lightweight boxes for development not just deployment.  
Boxes are meant for 'longer'-term usage than usual.  
Boxes are, in essence, containers.

## Installation
1. [Install Go](https://golang.org/doc/install)
2. Clone this repo:  
  - Git clone or download and unzip the source code  
  - Download a release from the releases
3. Fetch dependencies and build (in the cbox directory):  
`go get`  
`go build`
4. Add link "cbox" to path, to call from anywhere
`ln -s "absoulte-path-of-cbox" /usr/local/bin/cbox`

## Usage
1. Install as per steps given above:
    * Continue as current user and continue to the next step  
    * Elevate priveleges (if you want to use resource restrictions) - `sudo su`
2. Run commands on temporary containers:  
`cbox run /bin/bash` -> Note full path of executable  
!All data files are stored in /$USER/.cbox/
3. Create custom box:   
`cbox create` or a named version `cbox create [your box name]`
4. List available boxes:  
`cbox list`
5. Run commands on created boxes:  
`cbox start [box-name] [command]`
6. Delete unwanted boxes, use names given in the list:  
`cbox delete [box1 box2 box3]`
* For windows, prefix all steps with `wsl`

## Notes
1. The current directory will be mounted under the `/work` directory during `run`/`start`
2. Installing packages through apt is not supported as of now

## To be added -
- [x] Cgroup support  
- [ ] Network namespace
- [x] Bind mounts

## Requirements
* Go v1.14+
* amd64 architecture
* Linux based OS or WSL
* Internet connection for initial downloads
