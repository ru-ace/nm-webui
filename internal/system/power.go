package system

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Power actions supported by the Power section of the web UI.
const (
	PowerActionReboot   = "reboot"
	PowerActionPoweroff = "poweroff"
)

// PowerController executes reboot/poweroff commands against the host.
//
//   - running as root it invokes the shutdown binary (or systemctl) directly;
//   - running unprivileged it escalates through sudo with a tightly scoped
//     rule installed via deploy/sudoers/nm-webui-power, which permits exactly
//     the shutdown commands used here and nothing else.
//
// The sudo call is non-interactive (-n) so a missing/misconfigured sudoers
// rule fails fast with a clear error instead of hanging on a password prompt.
type PowerController struct {
	shutdownPath string
	useSudo      bool
}

// NewPowerController resolves the shutdown binary's location once so the
// exact command line stays stable across requests (and matches the sudoers
// rule, which is written against an absolute path).
func NewPowerController() *PowerController {
	return &PowerController{
		shutdownPath: lookupShutdownPath(),
		useSudo:      os.Geteuid() != 0,
	}
}

// lookupShutdownPath returns an absolute path to the shutdown binary, falling
// back to the classic /sbin and /usr/sbin locations when it is not on PATH.
func lookupShutdownPath() string {
	if p, err := exec.LookPath("shutdown"); err == nil {
		return p
	}
	for _, p := range []string{"/usr/sbin/shutdown", "/sbin/shutdown"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// Reboot restarts the host.
func (p *PowerController) Reboot() error { return p.action(PowerActionReboot) }

// PowerOff shuts the host down.
func (p *PowerController) PowerOff() error { return p.action(PowerActionPoweroff) }

// Check verifies, without performing it, that the given action can actually
// be executed, so the web UI can surface a clear error (e.g. missing shutdown
// binary, sudoers rule not installed) before the user believes a reboot was
// scheduled. Root needs no real check beyond the binary being present;
// unprivileged runs consult `sudo -l` for the exact command line.
func (p *PowerController) Check(action string) error {
	args := p.commandFor(action)
	if len(args) == 0 {
		return errors.New("no shutdown/systemctl binary available on this system")
	}
	if !p.useSudo {
		return nil
	}
	out, err := exec.Command("sudo", append([]string{"-n", "-l"}, args...)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("sudo not allowed for %s (install the bundled sudoers rule, see deploy/sudoers): %s",
			strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return nil
}

func (p *PowerController) action(action string) error {
	return p.run(p.commandFor(action))
}

// commandFor builds the privileged command line for an action, preferring the
// shutdown binary and falling back to systemctl when it is unavailable.
func (p *PowerController) commandFor(action string) []string {
	switch action {
	case PowerActionReboot:
		if p.shutdownPath != "" {
			return []string{p.shutdownPath, "-r", "now"}
		}
		return []string{"systemctl", "reboot"}
	case PowerActionPoweroff:
		if p.shutdownPath != "" {
			return []string{p.shutdownPath, "-h", "now"}
		}
		return []string{"systemctl", "poweroff"}
	}
	return nil
}

func (p *PowerController) run(args []string) error {
	if len(args) == 0 {
		return errors.New("power: no shutdown/systemctl binary available")
	}
	if p.useSudo {
		args = append([]string{"sudo", "-n"}, args...)
	}
	cmd := exec.Command(args[0], args[1:]...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
