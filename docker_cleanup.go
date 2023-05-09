package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	dockerRunningTimeStr := os.Getenv("DOCKER_RUNNING_TIME")
	if dockerRunningTimeStr == "" {
		dockerRunningTimeStr = "60"
	}
	dockerRunningTime, err := strconv.Atoi(dockerRunningTimeStr)
	if err != nil {
		log.Fatalf("Failed to parse DOCKER_RUNNING_TIME: %v", err)
	}

	logFilePath := os.Getenv("DOCKER_CLEANUP_LOG")
	if logFilePath == "" {
		logFilePath = "docker_cleanup.log"
	}

	logFileDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logFileDir, 0755); err != nil {
		log.Fatalf("Failed to create log file directory: %v", err)
	}

	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.41"))
	if err != nil {
		log.Fatalf("Failed to connect to Docker API: %v", err)
	}

	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		log.Fatalf("Failed to list Docker containers: %v", err)
	}

	now := time.Now()
	for _, container := range containers {
		containerInfo, err := dockerClient.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			log.Printf("Failed to inspect container %s: %v", container.ID, err)
			continue
		}
		containerCreatedAt, err := time.Parse(time.RFC3339Nano, containerInfo.Created)
		if err != nil {
			log.Printf("Failed to parse container created time for container %s: %v", container.ID, err)
			continue
		}
		containerRunningTime := now.Sub(containerCreatedAt).Minutes()
		if containerRunningTime > float64(dockerRunningTime) {
			log.Printf("Deleting container %s that has been running for %.2f minutes", container.ID, containerRunningTime)
			if err := dockerClient.ContainerRemove(context.Background(), container.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
				log.Printf("Failed to remove container %s: %s\n", container.ID, err)
			} else {
				logFile.WriteString(fmt.Sprintf("%s %s %.2f minutes\n", now.Format(time.RFC3339), container.ID, containerRunningTime))
			}
		}
	}
}
