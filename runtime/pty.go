package runtime

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// openPTY creates a new pseudo-terminal pair.
// Returns the master and slave file descriptors.
// The master is used by the parent (terminal side).
// The slave is used by the child (container side).
func openPTY() (*os.File, *os.File, error) {
	// Open the PTY master (multiplexor)
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open /dev/ptmx: %w", err)
	}

	// Get the slave PTY name and unlock it
	slaveName, err := ptsname(master)
	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("ptsname failed: %w", err)
	}

	if err = unlockpt(master); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("unlockpt failed: %w", err)
	}

	// Open the slave PTY
	slave, err := os.OpenFile(slaveName, os.O_RDWR, 0)
	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("failed to open slave pty: %w", err)
	}

	return master, slave, nil
}

// ptsname returns the name of the slave pseudo-terminal device
// corresponding to the given master.
func ptsname(master *os.File) (string, error) {
	n, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/dev/pts/%d", n), nil
}

// unlockpt unlocks the slave pseudo-terminal device.
// Must be called before the slave can be opened.
func unlockpt(master *os.File) error {
	return unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0)
}

// setRawMode puts the terminal into raw mode and returns a function to restore the original settings.
func setRawMode(fd int) (func(), error) {
	// Save current terminal settings
	oldState, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal state: %w", err)
	}

	// Create raw mode settings
	newState := *oldState
	// Disable canonical mode (line buffering) and echo
	newState.Lflag &^= unix.ICANON | unix.ECHO | unix.ISIG | unix.IEXTEN
	// Disable input processing
	newState.Iflag &^= unix.BRKINT | unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXON
	// Disable output processing
	newState.Oflag &^= unix.OPOST
	// Set character size to 8 bits
	newState.Cflag &^= unix.CSIZE | unix.PARENB
	newState.Cflag |= unix.CS8
	// Read returns immediately with whatever is available
	newState.Cc[unix.VMIN] = 1
	newState.Cc[unix.VTIME] = 0

	// Apply raw mode
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &newState); err != nil {
		return nil, fmt.Errorf("failed to set raw mode: %w", err)
	}

	// Return restore function
	return func() {
		unix.IoctlSetTermios(fd, unix.TCSETS, oldState)
	}, nil
}
