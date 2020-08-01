# Cardboard-box when you want to teleport away
---
Lightweight containers for development not deployment.

## Goals
1. Create Alpine linux fs (ALFS) without pre-installation in home directory
    > Save vanilla ALFS in home directory to save download time
    > ~/.cbox/data/{alpine-wsl}.tar.gz
2. List all file-systems created on system
    > ~/.cbox/labels.json => tags to box mappings
3. Use ALFS as a container. New ALFS for each container
    > ~/.cbox/boxes => store different containers
4. Install executable in /usr/local/bin