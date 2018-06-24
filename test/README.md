# Linux Images for testing

The Dockerfiles in this directory are used for testing certain Linux commands. Some things don't work perfect, such as
restarting systemd services (there is no systemd in the container) or connecting dnsdock to the Docker Bridge IP, but 
other code that checks for platform, etc. should run fine.

The way you use them is as follows.

## Build the Linux images

* For each Dockerfile, build the image 
    * `docker build -t test-fedora -f Dockerfile.fedora .`
    * `docker build -t test-ubuntu -f Dockerfile.ubuntu .`
    * `docker build -t test-centos -f Dockerfile.centos .`
   
## Build rig for Linux

* `GOARCH=amd64 GOOS=linux go build -o build/linux/rig cmd/main.go`

## Run the images (and Docker in Docker)

* Start Docker in Docker
    * `docker run --privileged -it --name dind -d docker:dind`
* Start the container for the distro you want, mounting a linux targeted `rig` into it and linking it to the Docker in Docker image.
    * `docker run -it -v $PWD/build/linux/rig:/usr/bin/rig --link dind:docker test-centos bash`
* You are now at a shell in the Linux container with `rig` in `/usr/bin/rig`
