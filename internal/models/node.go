package models

// ProbeState is the outcome of a yes/no probe. A probe whose command could not
// be run stays Unknown, so the UI can say "no data" instead of claiming the
// answer is "no".
type ProbeState int

const (
	// ProbeUnknown means the probe could not be answered.
	ProbeUnknown ProbeState = iota
	// ProbeUp means the probe answered yes.
	ProbeUp
	// ProbeDown means the probe answered no.
	ProbeDown
)

// ServiceState is one systemd unit and whatever systemctl said about it.
// State is empty when systemctl gave no answer at all.
type ServiceState struct {
	Name  string
	State string
}

// IsActive reports whether systemctl called the unit active.
func (s ServiceState) IsActive() bool {
	return s.State == "active"
}

// NodeStatus is a snapshot of the host the bot runs on. Each field comes from
// its own probe, and a probe that fails leaves its field empty (or Unknown)
// rather than aborting the whole snapshot.
type NodeStatus struct {
	// ChannelPort is the TCP port that was probed.
	ChannelPort string
	// Channel tells whether anything listens on ChannelPort.
	Channel ProbeState
	// Dest is the REALITY donor currently configured.
	Dest string
	// Watchdog is the last line of the watchdog log.
	Watchdog string
	// Services are the systemd units and their reported states.
	Services []ServiceState
	// Uptime, Memory and Disk are already-formatted one-liners.
	Uptime string
	Memory string
	Disk   string
}
