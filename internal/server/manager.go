package server

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ServerStatus represents the current state of the server
type ServerStatus string

const (
	StatusStopped  ServerStatus = "stopped"
	StatusStarting ServerStatus = "starting"
	StatusRunning  ServerStatus = "running"
	StatusStopping ServerStatus = "stopping"
	StatusError    ServerStatus = "error"
)

// Manager handles Suwayomi server process lifecycle
type Manager struct {
	mu sync.RWMutex

	// Server configuration
	executablePath string
	args           []string
	workDir        string

	// Process management
	cmd     *exec.Cmd
	status  ServerStatus
	pid     int
	startedAt time.Time

	// Logging
	logs      []string
	maxLogs   int
	logFile   *os.File

	// Callbacks
	onStatusChange func(ServerStatus)
	onLog          func(string)
}

// ManagerConfig holds configuration for the server manager
type ManagerConfig struct {
	ExecutablePath string   // Path to Suwayomi server JAR or binary
	Args           []string // Additional arguments
	WorkDir        string   // Working directory for the server
	LogFile        string   // Path to log file (optional)
	MaxLogs        int      // Maximum number of logs to keep in memory
}

// DefaultManagerConfig returns default server manager configuration
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		ExecutablePath: "", // Will be set by user
		Args:           []string{},
		WorkDir:        "",
		LogFile:        "",
		MaxLogs:        1000,
	}
}

// NewManager creates a new server manager
func NewManager(config *ManagerConfig) *Manager {
	if config == nil {
		config = DefaultManagerConfig()
	}

	return &Manager{
		executablePath: config.ExecutablePath,
		args:           config.Args,
		workDir:        config.WorkDir,
		status:         StatusStopped,
		logs:           make([]string, 0),
		maxLogs:        config.MaxLogs,
	}
}

// Start starts the Suwayomi server
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status == StatusRunning || m.status == StatusStarting {
		return fmt.Errorf("server is already running")
	}

	if m.executablePath == "" {
		return fmt.Errorf("server executable path not configured")
	}

	// Check if executable exists
	if _, err := os.Stat(m.executablePath); os.IsNotExist(err) {
		return fmt.Errorf("server executable not found: %s", m.executablePath)
	}

	// Determine command based on file extension
	var cmdPath string
	var cmdArgs []string

	if strings.HasSuffix(m.executablePath, ".jar") {
		// Java JAR file
		cmdPath = "java"
		cmdArgs = append([]string{"-jar", m.executablePath}, m.args...)
	} else {
		// Assume it's a binary
		cmdPath = m.executablePath
		cmdArgs = m.args
	}

	// Create command
	m.cmd = exec.Command(cmdPath, cmdArgs...)

	if m.workDir != "" {
		m.cmd.Dir = m.workDir
	}

	// Set up stdout/stderr pipes
	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := m.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := m.cmd.Start(); err != nil {
		m.setStatus(StatusError)
		return fmt.Errorf("failed to start server: %w", err)
	}

	m.pid = m.cmd.Process.Pid
	m.startedAt = time.Now()
	m.setStatus(StatusStarting)

	// Start log readers
	go m.readLogs(stdout, "STDOUT")
	go m.readLogs(stderr, "STDERR")

	// Monitor process
	go m.monitorProcess()

	m.addLog(fmt.Sprintf("Server started (PID: %d)", m.pid))

	return nil
}

// Stop stops the Suwayomi server
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status != StatusRunning && m.status != StatusStarting {
		return fmt.Errorf("server is not running")
	}

	if m.cmd == nil || m.cmd.Process == nil {
		return fmt.Errorf("no server process found")
	}

	m.setStatus(StatusStopping)
	m.addLog("Stopping server...")

	// Try graceful shutdown first
	if err := m.cmd.Process.Signal(os.Interrupt); err != nil {
		// If graceful shutdown fails, force kill
		if err := m.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to stop server: %w", err)
		}
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- m.cmd.Wait()
	}()

	select {
	case <-done:
		m.setStatus(StatusStopped)
		m.addLog("Server stopped")
		m.cmd = nil
		m.pid = 0
	case <-time.After(10 * time.Second):
		// Timeout - force kill
		if err := m.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to force kill server: %w", err)
		}
		m.setStatus(StatusStopped)
		m.addLog("Server force killed (timeout)")
		m.cmd = nil
		m.pid = 0
	}

	return nil
}

// Restart restarts the server
func (m *Manager) Restart() error {
	if err := m.Stop(); err != nil {
		return err
	}

	// Wait a bit before starting
	time.Sleep(time.Second)

	return m.Start()
}

// GetStatus returns the current server status
func (m *Manager) GetStatus() ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// GetPID returns the server process ID
func (m *Manager) GetPID() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pid
}

// GetUptime returns how long the server has been running
func (m *Manager) GetUptime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.status != StatusRunning {
		return 0
	}

	return time.Since(m.startedAt)
}

// GetLogs returns recent server logs
func (m *Manager) GetLogs(count int) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if count <= 0 || count > len(m.logs) {
		count = len(m.logs)
	}

	// Return last N logs
	start := len(m.logs) - count
	if start < 0 {
		start = 0
	}

	logs := make([]string, count)
	copy(logs, m.logs[start:])
	return logs
}

// ClearLogs clears all logs from memory
func (m *Manager) ClearLogs() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = make([]string, 0)
}

// SetCallbacks sets callback functions
func (m *Manager) SetCallbacks(onStatusChange func(ServerStatus), onLog func(string)) {
	m.onStatusChange = onStatusChange
	m.onLog = onLog
}

// readLogs reads logs from a pipe
func (m *Manager) readLogs(pipe io.Reader, prefix string) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		m.addLog(fmt.Sprintf("[%s] %s", prefix, line))
	}
}

// addLog adds a log entry
func (m *Manager) addLog(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	logEntry := fmt.Sprintf("[%s] %s", timestamp, msg)

	m.logs = append(m.logs, logEntry)

	// Keep only maxLogs entries
	if len(m.logs) > m.maxLogs {
		m.logs = m.logs[len(m.logs)-m.maxLogs:]
	}

	// Call log callback
	if m.onLog != nil {
		go m.onLog(logEntry)
	}
}

// setStatus sets the server status and calls callback
func (m *Manager) setStatus(status ServerStatus) {
	m.status = status

	if m.onStatusChange != nil {
		go m.onStatusChange(status)
	}
}

// monitorProcess monitors the server process
func (m *Manager) monitorProcess() {
	// Wait a bit to let server start
	time.Sleep(2 * time.Second)

	m.mu.Lock()
	if m.status == StatusStarting {
		m.setStatus(StatusRunning)
		m.addLog("Server is now running")
	}
	m.mu.Unlock()

	// Wait for process to exit
	if err := m.cmd.Wait(); err != nil {
		m.mu.Lock()
		if m.status != StatusStopping && m.status != StatusStopped {
			m.setStatus(StatusError)
			m.addLog(fmt.Sprintf("Server exited with error: %v", err))
		}
		m.mu.Unlock()
	} else {
		m.mu.Lock()
		if m.status != StatusStopping && m.status != StatusStopped {
			m.setStatus(StatusStopped)
			m.addLog("Server exited normally")
		}
		m.mu.Unlock()
	}
}
