package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
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
	dirname, err := os.UserHomeDir()
	if err != nil {
		println(err)
	}
	logDir := dirname + "/AppData/Roaming/plasma/log/"
	totalTimeFile := logDir + "total.txt"
	dailyTimeFile := logDir + date + ".txt"

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

	var totalDuration time.Duration

	if doesFileExist(totalTimeFile) {
		totalTimeContent, err := os.ReadFile(totalTimeFile)
		if err != nil {
			fmt.Println("Error reading total time file:", err)
			return
		}

		totalTimeParts := strings.Split(strings.TrimSpace(string(totalTimeContent)), ":")
		if len(totalTimeParts) == 4 {
			days, _ := strconv.Atoi(totalTimeParts[0])
			hours, _ := strconv.Atoi(totalTimeParts[1])
			minutes, _ := strconv.Atoi(totalTimeParts[2])
			seconds, _ := strconv.Atoi(totalTimeParts[3])
			totalDuration = time.Duration(days*24)*time.Hour + time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second
		} else {
			fmt.Println("Error parsing total time file format")
			return
		}

		totalDuration += timeSpent.Sub(time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC))
	} else {
		totalDuration = timeSpent.Sub(time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC))
	}

	hours := int(totalDuration.Hours())
	minutes := int(totalDuration.Minutes()) % 60
	seconds := int(totalDuration.Seconds()) % 60
	days := hours / 24
	hours = hours % 24

	newTotalDuration := fmt.Sprintf("%03d:%02d:%02d:%02d", days, hours, minutes, seconds)
	if err := os.WriteFile(totalTimeFile, []byte(newTotalDuration), 0644); err != nil {
		fmt.Println("Error writing to total time file:", err)
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
	dirname, err := os.UserHomeDir()
	if err != nil {
		println(err)
	}
	plasmaData := dirname + "/AppData/Roaming/plasma/"
	if _, err := os.Stat(plasmaData); os.IsNotExist(err) {
		if err := os.Mkdir(plasmaData, 0755); err != nil {
			fmt.Println("Error creating log directory:", err)
			return
		}
	}
	if doesFileExist(plasmaData + "icon.ico") {
		iconBytes, err := os.ReadFile(plasmaData + "icon.ico")
		if err != nil {
			fmt.Println("Error reading icon file:", err)
			return
		} else {
			systray.SetIcon(iconBytes)
		}
	} else if !doesFileExist(plasmaData + "icon.ico") {
		DownloadFile(plasmaData+"icon.ico", "https://raw.githubusercontent.com/darkmidus/Plasma/main/icon.ico")
		iconBytes, err := os.ReadFile("icon.ico")
		if err != nil {
			fmt.Println("Error reading icon file:", err)
			return
		} else {
			time.Sleep(1 * time.Second)
			systray.SetIcon(iconBytes)
		}
	}
	systray.SetTitle("Plasma")
	systray.SetTooltip("Your personal code tracker")
	mStats := systray.AddMenuItem("Stats", "Stats for you coding.")
	mStats.Click(func() {
		totalTime := statsChecker()
		Popup.Alert("Plasma Stats", totalTime)
		fmt.Println(totalTime)
	})
	mAbout := systray.AddMenuItem("About", "About Plasma")
	mAbout.Click(func() {
		Popup.Alert("About Plasma v0.06", "A litte FOSS stats tracker <3 -DarkMidus")
	})
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

func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func statsChecker() string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		println(err)
	}
	totalTimeFile := dirname + "/AppData/Roaming/plasma/log/total.txt"
	if doesFileExist(totalTimeFile) {
		totalTimeContent, err := os.ReadFile(totalTimeFile)
		if err != nil {
			fmt.Println("Error reading total time file:", err)
			return "Error reading total time file"
		}

		totalTimeParts := strings.Split(strings.TrimSpace(string(totalTimeContent)), ":")
		if len(totalTimeParts) != 4 {
			fmt.Println("Error parsing total time file format")
			return "Error parsing total time file format"
		}

		days, _ := strconv.Atoi(totalTimeParts[0])
		hours, _ := strconv.Atoi(totalTimeParts[1])
		minutes, _ := strconv.Atoi(totalTimeParts[2])
		seconds, _ := strconv.Atoi(totalTimeParts[3])

		formattedTime := fmt.Sprintf("%03d:%02d:%02d.%02d", days, hours, minutes, seconds)
		return "Total time spent coding: " + formattedTime
	} else if !doesFileExist(totalTimeFile) {
		return "No Stats Found."
	}
	return ""
}
