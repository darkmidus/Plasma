package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/ChromeTemp/Popup"
	"github.com/energye/systray"
)

func main() {
	isPlasmaRunning()
	systray.Run(onReady, onExit)
}

func checkProcessExistence(name string) (bool, int, error) {
	var processAmount int
	cmd := exec.Command("tasklist")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false, 0, err
	}
	output := out.String()
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, name) {
			processAmount++
		}
	}
	if processAmount > 0 {
		return true, processAmount, nil
	}
	return false, 0, nil
}

func doesFileExist(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	fmt.Println(err)
	return false
}

func documentUsage(date string, results string) {
	const logDir = "log"
	const totalTimeFile = logDir + "/total.txt"
	dailyTimeFile := logDir + "/" + date + ".txt"

	// Ensure the log directory exists
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if err := os.Mkdir(logDir, 0755); err != nil {
			fmt.Println("Error creating log directory:", err)
			return
		}
	}

	appendToFile(dailyTimeFile, results)

	timeSpentStr := strings.TrimSpace(strings.Split(results, "Time spent: ")[1])
	timeSpent, err := time.Parse("15:04:05", timeSpentStr)
	if err != nil {
		fmt.Println("Error parsing time spent:", err)
		return
	}

	if doesFileExist(totalTimeFile) {
		totalTimeContent, err := os.ReadFile(totalTimeFile)
		if err != nil {
			fmt.Println("Error reading total time file:", err)
			return
		}

		totalDuration, err := time.Parse("15:04:05", strings.TrimSpace(string(totalTimeContent)))
		if err != nil {
			fmt.Println("Error parsing total time:", err)
			return
		}

		// Add the new time spent to the total duration
		newTotalDuration := totalDuration.Add(timeSpent.Sub(time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)))
		if err := os.WriteFile(totalTimeFile, []byte(newTotalDuration.Format("15:04:05")), 0644); err != nil {
			fmt.Println("Error writing to total time file:", err)
		}
	} else {
		if err := os.WriteFile(totalTimeFile, []byte(timeSpentStr), 0644); err != nil {
			fmt.Println("Error writing to total time file:", err)
		}
	}
}

func appendToFile(filename, data string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(data); err != nil {
		fmt.Println("Error writing to file:", err)
	}
}

func onReady() {
	iconBytes, err := os.ReadFile("icon.ico")
	if err != nil {
		fmt.Println("Error reading icon file:", err)
		return
	}

	systray.SetIcon(iconBytes)
	systray.SetTitle("Plasma")
	systray.SetTooltip("Your personal code tracker")
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")
	mQuit.Enable()
	mQuit.Click(func() {
		fmt.Println("Requesting quit")
		systray.Quit()
		fmt.Println("Finished quitting")
		os.Exit(0)
	})

	go monitorProcess("Code.exe")
}

func monitorProcess(processName string) {
	for {
		running, _, err := checkProcessExistence(processName)
		if err != nil {
			fmt.Println("Error while checking process:", err)
			return
		}

		if running {
			start := time.Now()
			for {
				running, _, err := checkProcessExistence(processName)
				if err != nil {
					fmt.Println("Error while checking process:", err)
					return
				}
				if running {
					time.Sleep(1 * time.Second)
				} else {
					stop := time.Now()
					timeSpent := stop.Sub(start)
					hours := int(timeSpent.Hours())
					minutes := int(timeSpent.Minutes()) % 60
					seconds := int(timeSpent.Seconds()) % 60
					results := fmt.Sprintf(" Start: %s End: %s Time spent: %02d:%02d:%02d\n", start.Format("15:04:05"), stop.Format("15:04:05"), hours, minutes, seconds)
					documentUsage(start.Format("2006-01-02"), results)
					break
				}
			}
		} else {
			time.Sleep(1 * time.Second)
		}
	}
}

func isPlasmaRunning() {
	running, processAmount, err := checkProcessExistence("Plasma.exe")
	if err != nil {
		fmt.Println("Error while checking process:", err)
		return
	}
	if running && processAmount > 1 {
		Popup.Alert("Plasma Error", "Plasma is already running.")
		os.Exit(0)
	}
}

func onExit() {
	// clean up here
}
