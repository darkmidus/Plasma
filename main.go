package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ChromeTemp/Popup"
	"github.com/energye/systray"
)

func main() {
	isPlasmaRunning()
	systray.Run(onReady, onExit)
	for {
		running, _, err := checkProcessExistence("Code.exe")

		if err != nil {
			fmt.Println("Error while checking process: ", err)
			return
		}

		if running {
			start := time.Now()
			for {
				running, _, err := checkProcessExistence("Code.exe")

				if err != nil {
					fmt.Println("Error while checking process: ", err)
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
					results := " Start: " + start.Format("15:04:05") + " End: " + stop.Format("15:04:05") + fmt.Sprintf(" Time spent: %02d:%02d:%02d\n", hours, minutes, seconds)
					documentUsage(start.Format("2006-01-02"), results)
					break
				}
			}
		} else {
			time.Sleep(1 * time.Second)
		}

	}

}

func checkProcessExistence(name string) (bool, int, error) {
	var processAmount int
	cmd := exec.Command("tasklist")
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
	if _, err := os.Stat(filename); err == nil {
		return true

	} else if errors.Is(err, os.ErrNotExist) {
		return false

	} else {
		println(err.Error())
		return false
	}
}

func documentUsage(title string, results string) {
	filename := "log/" + title + ".txt"
	if doesFileExist(filename) {
		file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer file.Close()

		_, err = file.WriteString(results)
		if err != nil {
			fmt.Println(err.Error())
		}
	} else if !doesFileExist(filename) {
		file, err := os.Create(filename)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer file.Close()

		_, err = file.WriteString(results)
		if err != nil {
			fmt.Println(err.Error())
		}
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

}

func isPlasmaRunning() {
	running, processAmount, err := checkProcessExistence("Plasma.exe")
	if err != nil {
		fmt.Println("Error while checking process: ", err)
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
