package server

import (
	"context"
	"errors"
	"fmt"
	_ "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gin-gonic/gin"
	"github.com/hspgit/DockFormer/internal/database"
	"github.com/hspgit/DockFormer/internal/models"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"mime/multipart"
	_ "mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ContainersConfig struct {
	Containers []ContainerConfig `yaml:"containers"`
}

// ContainerConfig represents the YAML configuration for container creation
type ContainerConfig struct {
	Name     string            `yaml:"name"`
	Image    string            `yaml:"image"`
	Ports    string            `yaml:"ports"`
	Env      map[string]string `yaml:"env,omitempty"`
	Volumes  []string          `yaml:"volumes,omitempty"`
	Command  string            `yaml:"command,omitempty"`
	Networks []string          `yaml:"networks,omitempty"`
}

var dockerClient *client.Client

// InitDocker initializes the Docker client
func InitDocker() error {
	var err error
	dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	return err
}

// StartServer initializes and starts the HTTP server on the specified address
func StartServer(addr string) {
	if err := InitDocker(); err != nil {
		log.Fatalf("Failed to initialize Docker client: %v", err)
	}
	log.Println("Docker client initialized successfully")

	router := gin.Default()
	router.LoadHTMLGlob("web/templates/*.html")
	router.Static("/static", "web/static")
	setupRoutes(router)

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("Server starting on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRoutes(router *gin.Engine) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/", dashboardHandler)
	router.POST("/upload", uploadYamlHandler)
	router.GET("/container/:id/start", startContainerHandler)
	router.GET("/container/:id/stop", stopContainerHandler)
	router.GET("/container/:id/restart", restartContainerHandler)
	router.GET("/container/:id/logs", containerLogsHandler)

	api := router.Group("/api")
	{
		containers := api.Group("/containers")
		{
			containers.GET("", getContainers)
			containers.GET("/:id", getContainer)
			containers.POST("", createContainer)
			containers.PUT("/:id", updateContainer)
			containers.DELETE("/:id", deleteContainer)
			containers.POST("/:id/start", apiStartContainer)
			containers.POST("/:id/stop", apiStopContainer)
			containers.POST("/:id/restart", apiRestartContainer)
		}
	}
}

// Web UI handlers
func dashboardHandler(c *gin.Context) {
	var containerList []models.Container

	// Get containers from the database
	result := database.GetDB().Find(&containerList)
	if result.Error != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": result.Error.Error(),
		})
		return
	}

	// Update container statuses from Docker
	ctx := context.Background()
	for i := range containerList {
		if dockerClient != nil {
			containerInfo, err := dockerClient.ContainerInspect(ctx, containerList[i].Name)
			if err == nil {
				containerList[i].Status = models.ContainerStatus(containerInfo.State.Status)
			}
		}
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"containers": containerList,
	})
}

func uploadYamlHandler(c *gin.Context) {
	// Get the file from the request
	file, err := c.FormFile("yamlFile")
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Failed to get file: " + err.Error(),
		})
		return
	}

	// Validate file is a YAML file
	ext := filepath.Ext(file.Filename)
	if ext != ".yaml" && ext != ".yml" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "File must be a YAML file (.yaml or .yml)",
		})
		return
	}

	// Open the uploaded file
	openFile, err := file.Open()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to open file: " + err.Error(),
		})
		return
	}
	defer func(openFile multipart.File) {
		err := openFile.Close()
		if err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}(openFile)

	// Read the YAML content
	yamlData, err := io.ReadAll(openFile)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to read file: " + err.Error(),
		})
		return
	}

	// Parse the YAML into ContainersConfig
	var config ContainersConfig
	if err := yaml.Unmarshal(yamlData, &config); err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Failed to parse YAML: " + err.Error(),
		})
		return
	}

	// Iterate over each container configuration and create containers
	for _, containerConfig := range config.Containers {
		containerID, err := createDockerContainer(containerConfig)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"error": fmt.Sprintf("Failed to create container '%s': %v", containerConfig.Name, err),
			})
			return
		}

		// Save to the database
		containerObj := models.Container{
			Name:        containerConfig.Name,
			Image:       containerConfig.Image,
			Ports:       containerConfig.Ports,
			ContainerID: containerID,
			Status:      "created",
		}
		if result := database.GetDB().Create(&containerObj); result.Error != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"error": fmt.Sprintf("Failed to save container '%s' to database: %v", containerConfig.Name, result.Error),
			})
			return
		}
	}

	// Redirect back to dashboard
	c.Redirect(http.StatusSeeOther, "/")
}

func startContainerHandler(c *gin.Context) {
	id := c.Param("id")
	var containerObj models.Container

	if err := database.GetDB().First(&containerObj, id).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Container not found",
		})
		return
	}

	// Start Docker container
	ctx := context.Background()
	if err := dockerClient.ContainerStart(ctx, containerObj.Name, container.StartOptions{}); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to start container: " + err.Error(),
		})
		return
	}

	// Update status in database
	containerObj.Status = "running"
	database.GetDB().Save(&containerObj)

	// Redirect back to dashboard
	c.Redirect(http.StatusSeeOther, "/")
}

func stopContainerHandler(c *gin.Context) {
	id := c.Param("id")
	var containerObj models.Container

	if err := database.GetDB().First(&containerObj, id).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Container not found",
		})
		return
	}

	// Stop Docker container
	ctx := context.Background()
	if err := dockerClient.ContainerStop(ctx, containerObj.Name, container.StopOptions{}); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to stop container: " + err.Error(),
		})
		return
	}

	// Update status in database
	containerObj.Status = "stopped"
	database.GetDB().Save(&containerObj)

	// Redirect back to dashboard
	c.Redirect(http.StatusSeeOther, "/")
}

func restartContainerHandler(c *gin.Context) {
	id := c.Param("id")
	var containerObj models.Container

	if err := database.GetDB().First(&containerObj, id).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Container not found",
		})
		return
	}

	// Restart Docker container
	ctx := context.Background()
	if err := dockerClient.ContainerRestart(ctx, containerObj.Name, container.StopOptions{}); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to restart container: " + err.Error(),
		})
		return
	}

	// Update status in database
	containerObj.Status = "running"
	database.GetDB().Save(&containerObj)

	// Redirect back to dashboard
	c.Redirect(http.StatusSeeOther, "/")
}

func containerLogsHandler(c *gin.Context) {
	id := c.Param("id")
	var containerObj models.Container

	if err := database.GetDB().First(&containerObj, id).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Container not found",
		})
		return
	}

	// Get container logs from Docker
	ctx := context.Background()
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "100",
	}
	logReader, err := dockerClient.ContainerLogs(ctx, containerObj.Name, options)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to get container logs: " + err.Error(),
		})
		return
	}
	defer func(logReader io.ReadCloser) {
		err := logReader.Close()
		if err != nil {
			log.Printf("Failed to close log reader: %v", err)
		}
	}(logReader)

	// Read logs
	logs, err := io.ReadAll(logReader)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to read container logs: " + err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "logs.html", gin.H{
		"container": containerObj,
		"logs":      string(logs),
	})
}

// API handlers
func getContainers(c *gin.Context) {
	var containerList []models.Container

	result := database.GetDB().Find(&containerList)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// Update container statuses from Docker
	if dockerClient != nil {
		ctx := context.Background()
		for i := range containerList {
			containerInfo, err := dockerClient.ContainerInspect(ctx, containerList[i].Name)
			if err == nil {
				containerList[i].Status = models.ContainerStatus(containerInfo.State.Status)
			}
		}
	}

	c.JSON(http.StatusOK, containerList)
}

func getContainer(c *gin.Context) {
	id := c.Param("id")
	var containerObj models.Container

	result := database.GetDB().First(&containerObj, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Container not found"})
		return
	}

	// Get latest status from Docker
	if dockerClient != nil {
		ctx := context.Background()
		containerInfo, err := dockerClient.ContainerInspect(ctx, containerObj.Name)
		if err == nil {
			containerObj.Status = models.ContainerStatus(containerInfo.State.Status)
		}
	}

	c.JSON(http.StatusOK, containerObj)
}

func createContainer(c *gin.Context) {
	var containerObj models.Container

	// Bind JSON payload to containerObj
	if err := c.ShouldBindJSON(&containerObj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if containerObj.Name == "" || containerObj.Image == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name and image are required"})
		return
	}

	// Create Docker container
	config := ContainerConfig{
		Name:  containerObj.Name,
		Image: containerObj.Image,
		Ports: containerObj.Ports,
	}

	containerID, err := createDockerContainer(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Docker container: " + err.Error()})
		return
	}

	// Set initial status and ContainerID
	containerObj.Status = "created"
	containerObj.ContainerID = containerID

	// Save to the database
	result := database.GetDB().Create(&containerObj)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// Respond with the created container object
	c.JSON(http.StatusCreated, containerObj)
}

func updateContainer(c *gin.Context) {
	id := c.Param("id")
	var containerObj models.Container

	// Check if container exists
	if err := database.GetDB().First(&containerObj, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Container not found"})
		return
	}

	// Save current values to update only provided fields
	currentContainer := containerObj

	// Bind new data
	if err := c.ShouldBindJSON(&containerObj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure ID remains the same
	containerObj.ID = currentContainer.ID

	// Update container in database
	result := database.GetDB().Save(&containerObj)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, containerObj)
}

func deleteContainer(c *gin.Context) {
	id := c.Param("id")
	var containerObj models.Container

	// Get container details first to know the name
	if err := database.GetDB().First(&containerObj, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Container not found"})
		return
	}

	// Remove Docker container
	ctx := context.Background()
	if err := dockerClient.ContainerRemove(ctx, containerObj.Name, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}); err != nil {
		log.Printf("Error removing Docker container: %v", err)
	}

	// Remove from database
	result := database.GetDB().Delete(&containerObj)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Container deleted successfully"})
}

func apiStartContainer(c *gin.Context) {
	id := c.Param("id")
	var containerObj models.Container

	if err := database.GetDB().First(&containerObj, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Container not found"})
		return
	}

	// Start Docker container
	ctx := context.Background()
	if err := dockerClient.ContainerStart(ctx, containerObj.Name, container.StartOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start container: " + err.Error()})
		return
	}

	// Update status in database
	containerObj.Status = "running"
	database.GetDB().Save(&containerObj)

	c.JSON(http.StatusOK, gin.H{"message": "Container started successfully"})
}

func apiStopContainer(c *gin.Context) {
	id := c.Param("id")
	var containerObj models.Container

	if err := database.GetDB().First(&containerObj, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Container not found"})
		return
	}

	// Stop Docker container
	ctx := context.Background()
	if err := dockerClient.ContainerStop(ctx, containerObj.Name, container.StopOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop container: " + err.Error()})
		return
	}

	// Update status in database
	containerObj.Status = "stopped"
	database.GetDB().Save(&containerObj)

	c.JSON(http.StatusOK, gin.H{"message": "Container stopped successfully"})
}

func apiRestartContainer(c *gin.Context) {
	id := c.Param("id")
	var containerObj models.Container

	if err := database.GetDB().First(&containerObj, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Container not found"})
		return
	}

	// Restart Docker container
	ctx := context.Background()
	if err := dockerClient.ContainerRestart(ctx, containerObj.Name, container.StopOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restart container: " + err.Error()})
		return
	}

	// Update status in database
	containerObj.Status = "running"
	database.GetDB().Save(&containerObj)

	c.JSON(http.StatusOK, gin.H{"message": "Container restarted successfully"})
}

// Helper functions

// createDockerContainer creates a container in Docker based on the provided configuration
func createDockerContainer(config ContainerConfig) (string, error) {
	ctx := context.Background()

	// Pull image if it doesn't exist
	_, err := dockerClient.ImageInspect(ctx, config.Image)
	if err != nil {
		reader, pullErr := dockerClient.ImagePull(ctx, config.Image, image.PullOptions{})
		if pullErr != nil {
			return "", pullErr
		}
		defer func(reader io.ReadCloser) {
			err := reader.Close()
			if err != nil {
				log.Printf("Failed to close image pull reader: %v", err)
			}
		}(reader)

		// Stream the image pull progress
		if _, copyErr := io.Copy(os.Stdout, reader); copyErr != nil {
			return "", copyErr
		}
	}

	// Parse port mappings
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}

	if config.Ports != "" {
		for _, portStr := range strings.Split(config.Ports, ",") {
			portStr = strings.TrimSpace(portStr)
			if portStr == "" {
				continue
			}

			parts := strings.Split(portStr, ":")
			if len(parts) != 2 {
				return "", errors.New("invalid port mapping format")
			}

			hostPort := parts[0]
			containerPort := parts[1]

			// Check if container port includes protocol
			if !strings.Contains(containerPort, "/") {
				containerPort = containerPort + "/tcp"
			}

			natPort, err := nat.NewPort(strings.Split(containerPort, "/")[1], strings.Split(containerPort, "/")[0])
			if err != nil {
				return "", err
			}

			exposedPorts[natPort] = struct{}{}
			portBindings[natPort] = []nat.PortBinding{{HostPort: hostPort}}
		}
	}

	// Prepare environment variables
	var env []string
	for k, v := range config.Env {
		env = append(env, k+"="+v)
	}

	// Check if the container already exists
	_, err = dockerClient.ContainerInspect(ctx, config.Name)
	if err == nil {
		// Container exists, remove it first
		if err := dockerClient.ContainerRemove(ctx, config.Name, container.RemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		}); err != nil {
			return "", err
		}
	}

	// Create container
	containerConfig := &container.Config{
		Image:        config.Image,
		Env:          env,
		ExposedPorts: exposedPorts,
	}

	if config.Command != "" {
		containerConfig.Cmd = strings.Split(config.Command, " ")
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
	}

	if len(config.Volumes) > 0 {
		hostConfig.Binds = config.Volumes
	}

	response, err := dockerClient.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		config.Name,
	)
	if err != nil {
		return "", err
	}

	return response.ID, nil
}

// syncContainersWithDocker synchronizes the database with existing Docker containers
func syncContainersWithDocker() error {
	ctx := context.Background()

	// Get all containers from Docker
	containerList, err := dockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err
	}

	// Get all containers from database
	var dbContainers []models.Container
	if err := database.GetDB().Find(&dbContainers).Error; err != nil {
		return err
	}

	// Map for quick lookup of db containers
	dbContainerMap := make(map[string]models.Container)
	for _, c := range dbContainers {
		dbContainerMap[c.Name] = c
	}

	// Update or create entries
	for _, c := range containerList {
		// Use the container name without the leading slash
		name := strings.TrimPrefix(c.Names[0], "/")

		// Skip system containers
		if strings.HasPrefix(name, "k8s_") || name == "POD" {
			continue
		}

		ports := ""
		if len(c.Ports) > 0 {
			var portStrings []string
			for _, p := range c.Ports {
				if p.PublicPort != 0 {
					portStrings = append(portStrings,
						fmt.Sprintf("%d:%d", p.PublicPort, p.PrivatePort))
				}
			}
			ports = strings.Join(portStrings, ",")
		}

		if dbContainer, exists := dbContainerMap[name]; exists {
			// Update existing container
			dbContainer.Status = models.ContainerStatus(c.State)
			dbContainer.Image = c.Image
			dbContainer.Ports = ports
			database.GetDB().Save(&dbContainer)
			delete(dbContainerMap, name)
		} else {
			// Create new container entry
			newContainer := models.Container{
				Name:   name,
				Image:  c.Image,
				Status: models.ContainerStatus(c.State),
				Ports:  ports,
			}
			database.GetDB().Create(&newContainer)
		}
	}

	return nil
}
