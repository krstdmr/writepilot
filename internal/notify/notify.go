//go:build windows

package notify

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// Toast displays a Windows 10+ toast notification with the given title and message.
// It uses PowerShell's WinRT bindings to show a native notification.
// Notification failures are non-fatal; callers should log the returned error.
func Toast(title, message string) error {
	// Title and message are built from fixed strings plus provider names from
	// config.json (never LLM output). They are placed inside single-quoted
	// PowerShell literals, where the only special character is the single
	// quote itself, so doubling it keeps the script intact.
	title = strings.ReplaceAll(title, "'", "''")
	message = strings.ReplaceAll(message, "'", "''")
	script := fmt.Sprintf(
		`$ErrorActionPreference='Stop'
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType=WindowsRuntime]|Out-Null
$t=[Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent(
    [Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$x=[xml]$t.GetXml()
$x.GetElementsByTagName('text')[0].AppendChild($x.CreateTextNode('%s'))|Out-Null
$x.GetElementsByTagName('text')[1].AppendChild($x.CreateTextNode('%s'))|Out-Null
$d=New-Object Windows.Data.Xml.Dom.XmlDocument
$d.LoadXml($x.OuterXml)
$n=[Windows.UI.Notifications.ToastNotification]::new($d)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('WritePilot').Show($n)`,
		title, message,
	)

	cmd := exec.Command(
		"powershell",
		"-WindowStyle", "Hidden",
		"-NonInteractive",
		"-NoProfile",
		"-Command", script,
	)
	// CREATE_NO_WINDOW prevents the black console window from flashing
	// when PowerShell is spawned from a windowless background process.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("toast notification failed: %w", err)
	}
	return nil
}
