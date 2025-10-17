package main

import (
	"embed"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"mf/internal"

	"github.com/gin-gonic/gin"
)

var (
	BuildTime string
	GitCommit string = "running la"
)

var (
	listenPort     int    = 80
	proxyDownload  bool   = false
	downloadFolder string = "downloads"
)

func parseArgs() {
	flag.IntVar(&listenPort, "listenPort", listenPort, "Server port")
	flag.BoolVar(&proxyDownload, "proxyDownload", proxyDownload, "Enable download proxy")
	flag.StringVar(&downloadFolder, "downloadFolder", downloadFolder, "Download directory")
	flag.Bool("v", false, "Show version information")

	flag.Parse()
}

func printVersion() {
	fmt.Printf("┌───────────────────────────────────────────────┐\n")
	fmt.Printf("│\t\t\t\t\t\t│\n")
	fmt.Printf("│\t\tModbus Web\t\t\t│\n")
	fmt.Printf("│\t\t\t\t\t\t│\n")
	fmt.Printf("│\t• Build Time: %s\t│\n", BuildTime)
	fmt.Printf("│\t• Git Commit: %s\t\t\t│\n", GitCommit[:8])
	fmt.Printf("│\t\t\t\t\t\t│\n")
	fmt.Printf("│\t\t\t\tBy TwoMental\t│\n")
	fmt.Printf("│\t\t\t\t\t\t│\n")
	fmt.Printf("└───────────────────────────────────────────────┘\n")
}

//go:embed static/*
var staticFiles embed.FS

func main() {
	parseArgs()
	if flag.Lookup("v").Value.String() == "true" {
		printVersion()
		return
	}

	r := gin.Default()

	r.POST("/set-server", setServerHandler)
	r.POST("/get-value", getValueHandler)
	r.POST("/set-value", setValueHandler)

	r.GET("/version-info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"build_time": BuildTime, "git_commit": GitCommit[:8]})
	})

	r.GET("/allow-download", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"allow": proxyDownload})
	})
	r.GET("/serial-ports", serialPortsHandler)
	r.GET("/connection-status", connectionStatusHandler)

	if proxyDownload {
		r.Static("/downloads", downloadFolder)
		r.GET("/resource-list", resourceListHandler)
	}

	// Serve the main page
	static, _ := fs.Sub(staticFiles, "static")
	r.StaticFS("/home", http.FS(static))

	r.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/home")
	})

	// Start the server and open the home page
	go openBrowser(fmt.Sprintf("http://127.0.0.1:%d/home", listenPort))
	r.Run(fmt.Sprintf(":%d", listenPort))
}

var commands = map[string]string{
	"windows": "start",
	"darwin":  "open",
	"linux":   "xdg-open",
}

func openBrowser(url string) {
	run, ok := commands[runtime.GOOS]
	if !ok {
		return
	}

	cmd := exec.Command(run, url)
	cmd.Start()
}

func setServerHandler(c *gin.Context) {
	go internal.CleanConn()
	userID := internal.GetUserID(c)
	// Remove any existing connection for this user
	internal.DeleteConn(userID)

	// Connect to the Modbus server
	config := internal.ModbusConfig{}
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, err := internal.ConnModbus(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to Modbus server", "details": err.Error()})
		return
	}

	// Save the new connection
	internal.SaveConn(userID, client)
	c.JSON(http.StatusOK, gin.H{"message": "Connected to Modbus server"})
}

type getValueRequest struct {
	IDs []addressInfo `json:"ids" binding:"required"`
}

type addressInfo struct {
	RegisterType internal.RegisterType `json:"register_type"`
	Address      uint16                `json:"address"`
}

type valueDetail struct {
	Decimal uint16   `json:"decimal"`
	Bytes   []string `json:"bytes"`
}

type setValueRequest struct {
	Items []writeValueRequest `json:"items" binding:"required"`
}

type writeValueRequest struct {
	RegisterType internal.RegisterType `json:"register_type"`
	Address      uint16                `json:"address"`
	Value        uint16                `json:"value"`
}

type writeResult struct {
	RegisterType internal.RegisterType `json:"register_type"`
	Address      uint16                `json:"address"`
	Status       string                `json:"status"`
	Error        string                `json:"error,omitempty"`
}

// getValueHandler retrieves values from the Modbus server
func getValueHandler(c *gin.Context) {
	req := getValueRequest{}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Connect to Modbus server if not already connected
	client, ok := internal.GetConn(internal.GetUserID(c))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No Modbus server connected"})
		return
	}

	// Read values from Modbus server
	results := make([]valueDetail, len(req.IDs))
	for i, id := range req.IDs {
		rr, err := readWithAutoReconnect(client, id.RegisterType, id.Address)
		if err != nil {
			c.JSON(http.StatusInternalServerError,
				gin.H{"error": "Failed to read value from Modbus server", "details": err.Error()},
			)
			return
		}
		results[i] = valueDetail{
			Decimal: decodeRegisterValue(rr),
			Bytes:   internal.BytesToHexStrings(rr),
		}
	}
	c.JSON(http.StatusOK, results)
}
func resourceListHandler(c *gin.Context) {
	// List the files available in the download folder
	files, err := os.ReadDir(downloadFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read directory"})
		return
	}

	res := []string{}
	for _, file := range files {
		res = append(res, file.Name())
	}
	c.JSON(http.StatusOK, gin.H{"files": res})
}

func serialPortsHandler(c *gin.Context) {
	ports, err := internal.ListSerialPorts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list serial ports", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ports": ports})
}

func connectionStatusHandler(c *gin.Context) {
	client, ok := internal.GetConn(internal.GetUserID(c))
	if !ok {
		c.JSON(http.StatusOK, gin.H{"connected": false})
		return
	}

	_ = client.EnsureConnection(0)
	status := client.Status()
	response := gin.H{
		"connected": status.Connected,
		"mode":      status.Mode,
	}
	if !status.LastAlive.IsZero() {
		response["last_alive"] = status.LastAlive.Format(time.RFC3339)
	}
	if status.LastError != "" {
		response["last_error"] = status.LastError
	}
	c.JSON(http.StatusOK, response)
}

func setValueHandler(c *gin.Context) {
	req := setValueRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items must not be empty"})
		return
	}

	client, ok := internal.GetConn(internal.GetUserID(c))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No Modbus server connected"})
		return
	}

	results := make([]writeResult, len(req.Items))
	hasError := false
	for i, item := range req.Items {
		result := writeResult{RegisterType: item.RegisterType, Address: item.Address}
		err := writeWithAutoReconnect(client, item)
		if err != nil {
			hasError = true
			result.Status = "error"
			result.Error = err.Error()
		} else {
			result.Status = "ok"
		}

		results[i] = result
	}

	if hasError {
		c.JSON(http.StatusPartialContent, gin.H{"results": results})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func decodeRegisterValue(data []byte) uint16 {
	switch len(data) {
	case 0:
		return 0
	case 1:
		return uint16(data[0])
	default:
		return binary.BigEndian.Uint16(data[:2])
	}
}

func readWithAutoReconnect(server *internal.ModbusServer, registerType internal.RegisterType, address uint16) ([]byte, error) {
	data, err := server.ReadRegister(registerType, address)
	if err == nil {
		server.MarkSuccess()
		return data, nil
	}

	firstErr := err
	if recErr := server.Reconnect(); recErr != nil {
		server.MarkFailure(recErr)
		return nil, fmt.Errorf("read failed (%v) and reconnect failed: %w", firstErr, recErr)
	}

	data, err = server.ReadRegister(registerType, address)
	if err != nil {
		server.MarkFailure(err)
		return nil, err
	}
	server.MarkSuccess()
	return data, nil
}

func writeWithAutoReconnect(server *internal.ModbusServer, item writeValueRequest) error {
	err := server.WriteSingle(item.RegisterType, item.Address, item.Value)
	if err == nil {
		server.MarkSuccess()
		return nil
	}

	if errors.Is(err, internal.ErrReadOnly) {
		server.MarkFailure(err)
		return err
	}

	firstErr := err
	if recErr := server.Reconnect(); recErr != nil {
		server.MarkFailure(recErr)
		return fmt.Errorf("write failed (%v) and reconnect failed: %w", firstErr, recErr)
	}

	err = server.WriteSingle(item.RegisterType, item.Address, item.Value)
	if err != nil {
		server.MarkFailure(err)
		return err
	}
	server.MarkSuccess()
	return nil
}
