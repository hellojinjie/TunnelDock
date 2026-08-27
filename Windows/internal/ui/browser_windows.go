package ui

import "golang.org/x/sys/windows"

var shellOpenURL = shellOpen

func OpenBrowser(url string) error { return shellOpenURL(url) }

func shellOpen(url string) error {
	file, err := windows.UTF16PtrFromString(url)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, nil, file, nil, nil, 1)
}
