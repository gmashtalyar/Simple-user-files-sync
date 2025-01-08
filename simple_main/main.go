package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var (
	selectedDir       string
	emailHint         *widget.Label
	statusText        *widget.Label
	selectedDays      []string
	selectedTime      string
	organization      string
	organizationEntry *widget.Entry
)

func main() {
	myApp := app.NewWithID("com.simpleboard.simple_main")
	myWindow := myApp.NewWindow("SimpleBoard Синхронизация")
	myWindow.Resize(fyne.NewSize(600, 200))

	dirButton := widget.NewButton("1. Сделайте выбор папки", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				selectedDir = uri.Path()
				statusText.SetText(fmt.Sprintf("Выбранная папка: %s", selectedDir))
			}
		}, myWindow)
	})

	organizationButton := widget.NewButton("2. Укажите вашу корпоративную почту", func() {
		openOrganizationWindow(myApp)
	})
	emailHint = widget.NewLabel("Необходимо, чтобы определить организацию.")

	dateTimeButton := widget.NewButton("3. Сделайте выбор времени и дней", func() {
		openDateTimeWindow(myApp)
	})

	startDaemonButton := widget.NewButton("4. Начать синхронизацию", func() {
		if selectedDir == "" {
			dialog.ShowInformation("Ошибка", "Пожалуйста сделайте выбор папки", myWindow)
			return
		}
		if len(selectedDays) == 0 || selectedTime == "" {
			dialog.ShowInformation("Ошибка", "Пожалуйтса сделайте выбор времени и дней синхронизации", myWindow)
			return
		}
		if organization == "" {
			dialog.ShowInformation("Ошибка", "Пожалуйста введите название организации", myWindow)
			return
		}
		startDaemon()
		statusText.SetText("Синхронизация началась")
	})

	stopDaemonButton := widget.NewButton("Отмена синхронизации", func() {
		stopDaemon()
		statusText.SetText("Синхронизация остановлена")
	})

	statusText = widget.NewLabel("Синхронизация не установлена")

	myWindow.SetContent(container.NewVBox(
		dirButton,
		organizationButton,
		emailHint,
		dateTimeButton,
		startDaemonButton,
		stopDaemonButton,
		statusText,
	))

	myWindow.ShowAndRun()
}

func openOrganizationWindow(app fyne.App) {
	organizationWindow := app.NewWindow("Укажите вашу корпоративную почту")

	organizationEntry = widget.NewEntry()
	organizationEntry.SetPlaceHolder("Укажите вашу корпоративную почту")

	setButton := widget.NewButton("Установить", func() {
		organization = organizationEntry.Text
		if organization == "" {
			dialog.ShowInformation("Ошибка", "Пожалуйста укажите вашу корпоративную почту", organizationWindow)
			return
		}
		domain := extractDomain(organization)
		statusText.SetText(fmt.Sprintf("Организация: %s", domain))
		organizationWindow.Close()
	})

	organizationWindow.SetContent(container.NewVBox(
		widget.NewLabel("Введите название организации:"),
		organizationEntry,
		setButton,
	))

	organizationWindow.Show()
}

func openDateTimeWindow(app fyne.App) {
	dateTimeWindow := app.NewWindow("Сделайте выбор дней и времени")

	days := []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресение"}
	daysSelect := widget.NewCheckGroup(days, nil)

	timeEntry := widget.NewEntry()
	timeEntry.SetPlaceHolder("Введине время в формате ЧЧ:ММ")

	setButton := widget.NewButton("Установить", func() {
		selectedDays = daysSelect.Selected
		selectedTime = timeEntry.Text
		if len(selectedDays) == 0 || selectedTime == "" {
			dialog.ShowInformation("Ошибка", "Пожалуйста сделайте выбор дней и времени синхронизации", dateTimeWindow)
			return
		}
		statusText.SetText(fmt.Sprintf("Выбранные дни: %v, время: %s", selectedDays, selectedTime))
		dateTimeWindow.Close()
	})

	dateTimeWindow.SetContent(container.NewVBox(
		widget.NewLabel("Сделайте выбор дней синхронизации:"),
		daysSelect,
		widget.NewLabel("Сделайте выбор времени синхронизации:"),
		timeEntry,
		setButton,
	))

	dateTimeWindow.Show()
}

func startDaemon() {
	switch runtime.GOOS {
	case "windows":
		setupWindowsTaskScheduler()
	case "darwin":
		setupMacOSLaunchAgent()
	default:
		fmt.Println("OS not supported")
	}
}

func stopDaemon() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("schtasks", "/Delete", "/TN", "SimpleBoardEmailScheduler", "/F")
		cmd.Run()
	case "darwin":
		cmd := exec.Command("launchctl", "unload", os.ExpandEnv("$HOME/Library/LaunchAgents/com.simpleboard.emailscheduler.plist"))
		cmd.Run()
	default:
		fmt.Println("OS not supported")
	}
}

func setupWindowsTaskScheduler() {
	taskName := "SimpleBoardEmailScheduler"
	executablePath := filepath.Join(selectedDir, "sync_task.exe")
	days := strings.Join(selectedDays, ",")
	time := selectedTime

	cmd := exec.Command("schtasks", "/Create",
		"/TN", taskName,
		"/TR", executablePath,
		"/SC", "WEEKLY",
		"/D", days,
		"/ST", time,
		"/RI", "15",
		"/DU", "24:00",
		"/RL", "HIGHEST",
		"/ARG", organization,
	)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error creating Windows task:", err)
	} else {
		fmt.Println("Windows task created successfully")
	}
}

func setupMacOSLaunchAgent() {
	plistPath := os.ExpandEnv("$HOME/Library/LaunchAgents/com.simpleboard.emailscheduler.plist")
	timeParts := strings.Split(selectedTime, ":")
	hour := timeParts[0]
	minute := timeParts[1]

	syncTaskPath := filepath.Join(selectedDir, "sync_task")
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
	<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
	<plist version="1.0">
	<dict>
    <key>Label</key>
    <string>com.simpleboard.emailscheduler</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>%s</string>
        <string>%s</string>
    </array>
    <key>StartCalendarInterval</key>
    <array>`, syncTaskPath, selectedDir, organization)

	for _, day := range selectedDays {
		plistContent += fmt.Sprintf(`
        <dict>
            <key>Weekday</key>
            <integer>%d</integer>
            <key>Hour</key>
            <integer>%s</integer>
            <key>Minute</key>
            <integer>%s</integer>
        </dict>`, mapDayToInteger(day), hour, minute)
	}
	plistContent += `
    </array>
	<key>WakeForEvent</key>
    <true/>
	</dict>
	</plist>`

	err := os.WriteFile(plistPath, []byte(plistContent), 0644)
	if err != nil {
		fmt.Println("Error writing plist file:", err)
		return
	}

	cmd := exec.Command("launchctl", "load", plistPath)
	err = cmd.Run()
	if err != nil {
		fmt.Println("Error loading plist:", err)
	} else {
		fmt.Println("macOS Launch Agent created successfully")
	}
}

func mapDayToInteger(day string) int {
	switch day {
	case "Понедельник":
		return 1
	case "Вторник":
		return 2
	case "Среда":
		return 3
	case "Четверг":
		return 4
	case "Пятница":
		return 5
	case "Суббота":
		return 6
	case "Воскресение":
		return 7
	}
	return 0
}

func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}
