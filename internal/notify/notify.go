//go:build windows

package notify

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// toastScript builds the notification from $env:WP_TOAST_TITLE and
// $env:WP_TOAST_MESSAGE. The text travels through environment variables,
// never through the script source, so nothing in it (quotes, $(...), LLM
// output) can be interpreted as PowerShell code.
const toastScript = `$ErrorActionPreference='Stop'
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType=WindowsRuntime]|Out-Null
$t=[Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent(
    [Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$x=[xml]$t.GetXml()
$x.GetElementsByTagName('text')[0].AppendChild($x.CreateTextNode($env:WP_TOAST_TITLE))|Out-Null
$x.GetElementsByTagName('text')[1].AppendChild($x.CreateTextNode($env:WP_TOAST_MESSAGE))|Out-Null
$d=New-Object Windows.Data.Xml.Dom.XmlDocument
$d.LoadXml($x.OuterXml)
$n=[Windows.UI.Notifications.ToastNotification]::new($d)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('WritePilot').Show($n)`

// Toast displays a Windows 10+ toast notification with the given title and message.
// It uses PowerShell's WinRT bindings to show a native notification.
// Notification failures are non-fatal; callers should log the returned error.
func Toast(title, message string) error {
	cmd := exec.Command(
		"powershell",
		"-WindowStyle", "Hidden",
		"-NonInteractive",
		"-NoProfile",
		"-Command", toastScript,
	)
	cmd.Env = append(os.Environ(),
		"WP_TOAST_TITLE="+title,
		"WP_TOAST_MESSAGE="+message,
	)
	// CREATE_NO_WINDOW prevents the black console window from flashing
	// when PowerShell is spawned from a windowless background process.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("toast notification failed: %w", err)
	}
	return nil
}
