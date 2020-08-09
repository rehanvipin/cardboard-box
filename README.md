# Cardboard-box for when you want to teleport away

Lightweight boxes for development not just deployment.  
Boxes are meant for 'longer'-term usage than usual.  
Boxes are, in essence, containers.

## Installation
1. Install Go:  
`sudo apt install golang` for Debian based OSes
2. Clone this repo:  
Git clone or download and unzip the source code
3. Build (in the cbox directory):  
`go build`
4. Link "cbox" to cardboad-box to call from anywhere
`ln -s "absoulte-path-of-cbox" /usr/local/bin/cbox`

## Usage
1. Install as per steps given above:
    * Elevate privelages (if you want to use resource restrictions) - `sudo su`
    * Continue as current user
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

## To be added -
- [x] Cgroup support  
- [ ] Network namespace
- [ ] Bind mounts

## Requirements
* Go v1.14+
* amd64 architecture
* Linux based OS or WSL
