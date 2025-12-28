package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type psClient struct {
	server string
}

func (c *psClient) isWindows() bool {
	return runtime.GOOS == "windows"
}

func (c *psClient) commandPrefix() string {
	if c.server == "" {
		return ""
	}
	return fmt.Sprintf("$vmm = Get-SCVMMServer -ComputerName '%s'; ", escapeSingleQuotes(c.server))
}

func escapeSingleQuotes(input string) string {
	return strings.ReplaceAll(input, "'", "''")
}

func (c *psClient) runPSJSON(ctx context.Context, script string) (map[string]interface{}, error) {
	if !c.isWindows() {
		return nil, errors.New("PowerShell cmdlets require Windows")
	}

	full := c.commandPrefix() + script + " | ConvertTo-Json -Depth 6"
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", full)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("powershell error: %w: %s", err, string(out))
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(out, &decoded); err != nil {
		return nil, fmt.Errorf("json parse error: %w: %s", err, string(out))
	}

	return decoded, nil
}

func (c *psClient) runPS(ctx context.Context, script string) error {
	if !c.isWindows() {
		return errors.New("PowerShell cmdlets require Windows")
	}

	full := c.commandPrefix() + script
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", full)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("powershell error: %w: %s", err, string(out))
	}

	return nil
}
