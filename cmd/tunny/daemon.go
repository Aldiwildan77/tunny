package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	daemonLib "github.com/sevlyar/go-daemon"
)

const daemonStopTimeout = 10 * time.Second

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the tunny daemon process",
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of the tunny daemon process",
	RunE: func(cmd *cobra.Command, args []string) error {
		return checkDaemonStatus(daemonPID)
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the tunny daemon process",
	RunE: func(cmd *cobra.Command, args []string) error {
		return stopDaemon(daemonPID)
	},
}

func init() {
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonStopCmd)
}

func readPID(PIDFilePath string) (int, error) {
	data, err := os.ReadFile(PIDFilePath)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file: %s", PIDFilePath)
	}

	if pid <= 0 {
		return 0, fmt.Errorf("invalid PID file %s value: %d", PIDFilePath, pid)
	}

	return pid, nil
}

func processRunning(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func daemonize(isDaemonMode bool, daemonPID string, daemonLog string) (bool, func(), error) {
	if !isDaemonMode {
		return false, func() {}, nil
	}

	workDir, err := os.Getwd()
	if err != nil {
		return false, nil, fmt.Errorf("get working directory: %w", err)
	}

	daemonCtx := &daemonLib.Context{
		PidFileName: daemonPID,
		PidFilePerm: 0644,
		LogFileName: daemonLog,
		LogFilePerm: 0640,
		WorkDir:     workDir,
		Umask:       027,
		Args:        os.Args,
	}

	child, err := daemonCtx.Reborn()
	if err != nil {
		return false, nil, fmt.Errorf("failed to daemonize: %w", err)
	}

	if child != nil {
		log.Printf("Daemon process started with PID %d", child.Pid)
		return true, func() {}, nil
	}

	return false, func() {
		if err := daemonCtx.Release(); err != nil {
			log.Printf("Failed to release daemon context: %v", err)
		}
	}, nil
}

func checkDaemonStatus(daemonPID string) error {
	pid, err := readPID(daemonPID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("Daemon process is not running")
			return nil
		}

		return err
	}

	if !processRunning(pid) {
		log.Printf("Daemon process with PID %d is not running", pid)
		_ = os.Remove(daemonPID)
		return nil
	}

	log.Printf("Daemon process with PID %d is running", pid)
	return nil
}

func stopDaemon(daemonPID string) error {
	pid, err := readPID(daemonPID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("Daemon process is not running")
			return nil
		}

		return err
	}

	if !processRunning(pid) {
		log.Printf("Daemon process with PID %d is not running", pid)
		_ = os.Remove(daemonPID)
		return nil
	}

	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to stop daemon process with PID %d: %w", pid, err)
	}

	deadline := time.Now().Add(daemonStopTimeout)
	for time.Now().Before(deadline) {
		if !processRunning(pid) {
			log.Printf("Daemon process with PID %d has been stopped", pid)
			_ = os.Remove(daemonPID)
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("failed to stop daemon process with PID %d within the deadline", pid)
}
