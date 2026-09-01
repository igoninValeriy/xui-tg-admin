package services

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"xui-tg-admin/internal/constants"
	"xui-tg-admin/internal/helpers"
	"xui-tg-admin/internal/models"
)

// NodeTargets collects every file and tool the node screens read. The bot runs
// on the same host as the VPN tooling, so it asks that tooling rather than
// reimplementing it; the targets are declared once here instead of being
// spread across the handlers.
type NodeTargets struct {
	DestFile       string
	WatchdogLog    string
	SubStatsScript string
	Services       []string
	ChannelPort    string
}

// DefaultNodeTargets returns the layout of the production host.
func DefaultNodeTargets() NodeTargets {
	return NodeTargets{
		DestFile:       constants.NodeDestFile,
		WatchdogLog:    constants.NodeWatchdogLog,
		SubStatsScript: constants.NodeSubStatsScript,
		Services:       constants.NodeServices,
		ChannelPort:    constants.NodeChannelPort,
	}
}

// NodeService reads the state of the host the bot runs on. It is pure I/O: all
// parsing and formatting lives in helpers.
type NodeService struct {
	targets NodeTargets
	logger  *logrus.Logger
}

// NewNodeService creates a node service over the default host layout.
func NewNodeService(logger *logrus.Logger) *NodeService {
	return NewNodeServiceWithTargets(DefaultNodeTargets(), logger)
}

// NewNodeServiceWithTargets creates a node service over an explicit layout.
func NewNodeServiceWithTargets(targets NodeTargets, logger *logrus.Logger) *NodeService {
	return &NodeService{targets: targets, logger: logger}
}

// Status collects a snapshot of the host. It never fails as a whole: a probe
// that cannot be run leaves its field empty so the screen can say "no data".
func (s *NodeService) Status(ctx context.Context) models.NodeStatus {
	status := models.NodeStatus{ChannelPort: s.targets.ChannelPort}

	if out, err := s.run(ctx, constants.NodeCommandTimeout, "ss", "-ltn"); err != nil {
		s.logger.Errorf("Node probe 'ss -ltn' failed: %v", err)
	} else if helpers.PortIsListening(out, s.targets.ChannelPort) {
		status.Channel = models.ProbeUp
	} else {
		status.Channel = models.ProbeDown
	}

	if content, err := os.ReadFile(s.targets.DestFile); err != nil {
		s.logger.Errorf("Failed to read %s: %v", s.targets.DestFile, err)
	} else {
		status.Dest = strings.TrimSpace(string(content))
	}

	if content, err := os.ReadFile(s.targets.WatchdogLog); err != nil {
		s.logger.Errorf("Failed to read %s: %v", s.targets.WatchdogLog, err)
	} else {
		status.Watchdog = helpers.LastLine(string(content))
	}

	// systemctl exits non-zero when any unit is inactive but still prints one
	// state per unit, so the output matters more than the exit code here.
	out, err := s.run(ctx, constants.NodeCommandTimeout, "systemctl", append([]string{"is-active"}, s.targets.Services...)...)
	if err != nil && strings.TrimSpace(out) == "" {
		s.logger.Errorf("Node probe 'systemctl is-active' failed: %v", err)
		status.Services = helpers.ParseServiceStates(s.targets.Services, "")
	} else {
		status.Services = helpers.ParseServiceStates(s.targets.Services, out)
	}

	if out, err := s.run(ctx, constants.NodeCommandTimeout, "uptime", "-p"); err != nil {
		s.logger.Errorf("Node probe 'uptime -p' failed: %v", err)
	} else {
		status.Uptime = strings.TrimSpace(out)
	}

	if out, err := s.run(ctx, constants.NodeCommandTimeout, "free", "-m"); err != nil {
		s.logger.Errorf("Node probe 'free -m' failed: %v", err)
	} else {
		status.Memory = helpers.FormatMemory(out)
	}

	if out, err := s.run(ctx, constants.NodeCommandTimeout, "df", "-h", "/"); err != nil {
		s.logger.Errorf("Node probe 'df -h /' failed: %v", err)
	} else {
		status.Disk = helpers.FormatDisk(out)
	}

	return status
}

// SubscriptionStats runs the host's subscription statistics script and returns
// its stdout untouched; the caller decides how to display it.
func (s *NodeService) SubscriptionStats(ctx context.Context) (string, error) {
	out, err := s.run(ctx, constants.NodeSubStatsTimeout, s.targets.SubStatsScript)
	if err != nil {
		return "", fmt.Errorf("failed to run %s: %w", s.targets.SubStatsScript, err)
	}
	return out, nil
}

// run executes a local command under the caller's context plus a timeout, and
// returns its stdout. stderr is folded into the error so failures are legible
// in the log.
func (s *NodeService) run(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(cmdCtx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			err = fmt.Errorf("%w: %s", err, msg)
		}
		return stdout.String(), err
	}
	return stdout.String(), nil
}
