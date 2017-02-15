# Changelog

## 1.1.0

 - Renamed from `devtools` to `rig`.
 - Renamed `DEVTOOLS_NAMESERVERS` to `RIG_NAMESERVER`
 - Renamed `.devtools-watch-ignore` file to `.rid-watch-ignore`
 - Moved `--name` to a global option on `rig` and the env var to `RIG_ACTIVE_MACHINE`
 - Upgraded `doctor` to check API compatibility instead of just docker versions

## 1.0.5

 - Reformatted `dns-records` to work be in `/etc/hosts` format
 - Added the `dashboard` command and launch of the dashboard on `start`

## 1.0.4

 - Fixed multiple fallback nameservers in dnsdock
 - Transitioned to labels for dnsdock name registration 

## 1.0.3

 - Fixed dnsdock argument specification
 - Pinned dnsdock to a well-known version

## 1.0.2

 - Fixed output of `doctor` command. A key change to IP address caused a formatting error in the output

## 1.0.1

 - Fixed output of `dns-records` command. A key change to IP address caused a formatting error in the output

## 1.0.0

 - Upgraded dnsdock and introduced DEVTOOLS_NAMESERVERS env var to provide additioanl DNS forwarder name servers
 - Removed dnsmasq. Primary and fallback resolution can now happen via dnsdock

## 0.4.0

 - Moved custom `/data` mounts from `docker-machine-nfs` script to `bootsync.sh`
 - Removed custom `docker-machine-nfs` script, now uses canonical script from `brew`
 - Added `doctor` check for docker-machine-nfs script
 - Added `doctor` check for docker cli and docker-machine version compatibility
 - Added `upgrade` command
 - Backup `data-backup` and restore `data-restore` commands now stream tar files to better preserve permissions
 - Introduced semver library to help with version comparisons
 - Upgraded to `github.com/urfave/cli` from `github.com/codegangsta/cli`
 - Widespread `go fmt` updates
 - Moved custom `/data` mounts from `docker-machine-nfs` script to `bootsync.sh`
 - Enable NAT DNS Proxy for VirtualBox

## 0.3.1

 - Flush DNS on OS X when DNS services are started
 - Build with the -cgo flag so that the Go binary uses host DNS resolution instead of internal
 - Expanded `doctor` to test DNS records and network connectivity
 - Added `dns-records` command