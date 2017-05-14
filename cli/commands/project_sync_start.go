package commands

import (
  "fmt"
  "io/ioutil"
  "net"
  "os"
  "os/exec"
  "path"
  "strings"
  "time"

  containertypes 	"github.com/docker/docker/api/types/container"
  volumetypes 		"github.com/docker/docker/api/types/volume"
  "github.com/docker/docker/client"
  "github.com/phase2/rig/cli/commands/project"
  "github.com/urfave/cli"
  "golang.org/x/net/context"
  "gopkg.in/yaml.v2"
)

type ProjectSyncStart struct {
  BaseCommand
}

// Minimal compose file struct to discover volumes
type ComposeFile struct {
  Volumes map[string]Volume
}

type Volume struct {
  External bool
}

const UNISON_PORT = 5000

// TODO: In Start we need to "sudo sysctl fs.inotify.max_user_watches=100000" add it to /etc/boot2docker/bootsync.sh
// TODO: Check for the fs.inotify.max_user_watches configuration in doctor command "sudo sysctl fs.inotify.max_user_watches"

func (cmd *ProjectSyncStart) Commands() cli.Command {
  command := cli.Command{
    Name:        "sync:start",
    Usage:       "Start a unison sync on local project directory. Optionally provide a volume name. Volume name will be discovered in the following order: argument, outrigger project config, docker-compose file, current directory name",
    ArgsUsage:   "[optional volume name]",
    Before:      cmd.Before,
    Action:      cmd.Run,
  }

  return command
}

// Start the unison sync process
func (cmd *ProjectSyncStart) Run(ctx *cli.Context) error {
  project.ConfigInit()
  config := project.GetProjectConfigFromFile(project.GetConfigPath())
  volumeName := cmd.GetVolumeName(ctx, config)
  cmd.out.Verbose.Printf("Sync with volume: %s", volumeName)

  docker, err := client.NewEnvClient()
  if err != nil {
  	cmd.out.Error.Fatal("Unable to create Docker Client")
  }

  cmd.out.Info.Printf("Starting sync volume: %s", volumeName)
  docker.VolumeCreate(context.Background(), volumetypes.VolumesCreateBody{ Name: volumeName })


  cmd.out.Info.Println("Starting unison container")
  hostConfig := &containertypes.HostConfig{
  	AutoRemove: true,
  	Binds: []string{fmt.Sprintf("%s:/unison", volumeName)},
  }
  containerConfig := &containertypes.Config{
  	Env: []string{"UNISON_DIR=/unison"},
  	Labels: map[string]string{
  		"com.dnsdock.name": volumeName,
  		"com.dnsdock.image": "volume.outrigger",
  	},
  	Image: "outrigger/unison:latest",
  }
  container, err := docker.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, volumeName)
  if err != nil {
  	cmd.out.Error.Fatalf("Error starting sync container %s: %v", volumeName, err)
  }

  cmd.WaitForUnisonContainer(docker, container.ID)

  cmd.out.Info.Println("Initializing sync")

  // Remove the log file, the existence of the log file will mean that sync is up and running
  var logFile = fmt.Sprintf("%s.log", volumeName)
  exec.Command("rm", "-f", logFile).Run()

  // Start unison local process
  exec.Command("unison",
    ".",
    fmt.Sprintf("socket://%s.volume.outrigger.vm:%d/", volumeName, UNISON_PORT),
    "-auto", "-batch", "-silent", "-contactquietly",
    "-repeat watch",
    "-prefer .",
    fmt.Sprintf("-ignore 'Name %s'", logFile),
    fmt.Sprintf("-logfile %s", logFile),
  ).Start()

  cmd.WaitForSyncInit(logFile)

  return nil
}

// Find the volume name through a variety of fall backs
func (cmd *ProjectSyncStart) GetVolumeName(ctx *cli.Context, config project.ProjectConfig) string {
  // 1. Check for argument
  if ctx.Args().Present() {
    return ctx.Args().First()
  }

  // 2. Check for config
  if config.Sync != nil && config.Sync.Volume != "" {
    return config.Sync.Volume
  }

  // 3. Parse compose file looking for an external volume named *-sync
  if composeConfig, err := cmd.LoadComposeFile(); err == nil {
    for name, volume := range composeConfig.Volumes {
      cmd.out.Verbose.Printf("Volume: Name: %s External: %t", name, volume.External);
      if strings.HasSuffix(name, "-sync") && volume.External{
        return name
      }
    }
  }

  // 4. Use local dir for the volume name
  if dir, err := os.Getwd(); err == nil {
    var _, folder = path.Split(dir)
    return folder
  } else {
    cmd.out.Error.Println(err)
  }

  cmd.out.Error.Fatal("Unable to determine a name for the sync volume")
  return ""
}

// Load the proper compose file
func (cmd *ProjectSyncStart) LoadComposeFile() (*ComposeFile, error) {
  yamlFile, err := ioutil.ReadFile("./docker-compose.yml")

  if err == nil {
    var config ComposeFile
    if err := yaml.Unmarshal(yamlFile, &config); err != nil {
      cmd.out.Error.Fatalf("YAML Parsing Error: %s", err)
    }
    return &config, nil
  }

  return nil, err
}

// Wait for the unison container port to allow connections
// Due to the fact that we don't compile with -cgo (so we can build using Docker),
// we need to discover the IP address of the container instead of using the DNS name
// when compiled without -cgo this executable will not use the native mac dns resolution
// which is how we have configured dnsdock to provide names for containers.
func (cmd *ProjectSyncStart) WaitForUnisonContainer(client *client.Client, containerId string) {
  cmd.out.Info.Println("Waiting for container to start")
  containerData, err := client.ContainerInspect(context.Background(), containerId)
  if err != nil {
    cmd.out.Error.Fatalf("Error inspecting sync container %s: %v", containerId, err)
  }
  for i := 1; i <= 100; i++ {
    //if err := exec.Command("nc", "-z", containerData.NetworkSettings.IPAddress, UNISON_PORT).Run(); err != nil {
    if _, err := net.Dial("tcp", fmt.Sprintf("%s:%s", containerData.NetworkSettings.IPAddress, UNISON_PORT)); err == nil {
      return
    } else {
      time.Sleep(time.Duration(100) * time.Millisecond)
    }
  }
  cmd.out.Error.Fatal("Sync container failed to start!")
}

// The local unison process is finished initializing when the log file exists
func (cmd *ProjectSyncStart) WaitForSyncInit(logFile string) {
  cmd.out.Info.Print("Waiting for initial sync to finish...")

  var tempFile = fmt.Sprintf(".%s.tmp", logFile)

  // Create a temp file to cause a sync action
  exec.Command("touch", tempFile).Run()

  // Lets check for 60 seconds, while waiting for initial sync to complete
  for i := 1; i <= 600; i++ {
    os.Stdout.WriteString(".")
    if _, err := os.Stat(logFile); err == nil {
      // Remove the temp file now that we are running
      os.Stdout.WriteString("done\n")
      exec.Command("rm", "-f", tempFile).Run()
      return
    } else {
      time.Sleep(time.Duration(100) * time.Millisecond)
    }
  }

  // The log file was not created, the sync has not started yet
  exec.Command("rm", "-f", tempFile).Run()
  cmd.out.Error.Fatal("Sync container failed to start!")
}

