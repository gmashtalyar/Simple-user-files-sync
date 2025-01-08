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
	selectedDir  string
	statusText   *widget.Label
	selectedDays []string
	selectedTime string
)

func main() {
	myApp := app.NewWithID("com.simpleboard.simple_main")
	myWindow := myApp.NewWindow("SimpleBoard Synchronization")

	dirButton := widget.NewButton("Сделайте выбор папки", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				selectedDir = uri.Path()
				statusText.SetText(fmt.Sprintf("Выбранная папка: %s", selectedDir))
			}
		}, myWindow)
	})

	dateTimeButton := widget.NewButton("Сделайте выбор времени и дней", func() {
		openDateTimeWindow(myApp)
	})

	startDaemonButton := widget.NewButton("Начать синхронизацию", func() {
		if selectedDir == "" {
			dialog.ShowInformation("Ошибка", "Пожалуйста сделайте выбор папки", myWindow)
			return
		}
		if len(selectedDays) == 0 || selectedTime == "" {
			dialog.ShowInformation("Ошибка", "Пожалуйтса сделайте выбор времени и дней синхронизации", myWindow)
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
		dateTimeButton,
		startDaemonButton,
		stopDaemonButton,
		statusText,
	))

	myWindow.ShowAndRun()
}

func openDateTimeWindow(app fyne.App) {
	dateTimeWindow := app.NewWindow("Сделайте выбор дней и времени")

	days := []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресение"}
	daysSelect := widget.NewCheckGroup(days, nil)

	timeEntry := widget.NewEntry()
	timeEntry.SetPlaceHolder("Введине время в формате ЧЧ:ММ")

	setButton := widget.NewButton("Set", func() {
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

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.simpleboard.emailscheduler</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>StartCalendarInterval</key>
	<array>`, filepath.Join(selectedDir, "sync_task"))
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
	<key>StartInterval</key>
	<integer>900</integer>
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
		return 2
	case "Вторник":
		return 3
	case "Среда":
		return 4
	case "Четверг":
		return 5
	case "Пятница":
		return 6
	case "Суббота":
		return 7
	case "Воскресение":
		return 1
	}
	return 0
}
