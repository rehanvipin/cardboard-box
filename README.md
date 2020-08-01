# Cardboard-box when you want to teleport away
---
Lightweight containers for development not deployment.

## Goals
1. Create Ubuntu linux fs (ULFS) without pre-installation in home directory
    > Save vanilla ULFS in home directory to save download time
    > ~/.cbox/data/{Ubuntu-wsl}.tar.gz
2. List all file-systems created on system
    > ~/.cbox/labels.json => tags to box mappings
3. Use ULFS as a container. New ULFS for each container
    > ~/.cbox/boxes => store different containers
4. Install executable in /usr/local/bin

## Requirements
* Golang v1.14+
* amd64 architecture