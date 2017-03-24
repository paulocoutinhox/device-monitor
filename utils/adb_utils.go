package utils

import (
	"errors"
	"os/exec"

	"regexp"
)

func AdbExecute(serial string, args ...string) ([]byte, error) {
	commandArgs := []string{"-s", serial}
	commandArgs = append(commandArgs, args...)
	return exec.Command("adb", commandArgs...).Output()
}

func AdbShellExecute(serial string, args ...string) ([]byte, error) {
	commandArgs := []string{"shell"}
	commandArgs = append(commandArgs, args...)
	return AdbExecute(serial, commandArgs...)
}

func AdbGetScreenshot(serial string) ([]byte, error) {
	output, err := AdbShellExecute(serial, "screencap", "-p")

	if err != nil {
		return nil, err
	}

	var imageBuffer []byte
	totalBytesInOutputRaw := len(output)
	ignoreByteIndex := -1

	for i, currentByte := range output {
		if i < ignoreByteIndex {
			continue
		}

		var nextByte byte
		nextByteIndex := i + 1

		if nextByteIndex < totalBytesInOutputRaw {
			nextByte = output[nextByteIndex]
		}

		if currentByte == 0x0D && nextByte == 0x0A {
			ignoreByteIndex = nextByteIndex
			continue
		}

		imageBuffer = append(imageBuffer, currentByte)
	}

	if len(imageBuffer) == 0 {
		return nil, errors.New("Empty image data")
	}

	return imageBuffer, nil
}

func AdbInputTap(serial string, x string, y string) error {
	_, err := AdbShellExecute(serial, "input", "tap", x, y)

	if err != nil {
		return err
	}

	return nil
}

func AdbInputKeyEvent(serial string, keyEvent string) error {
	_, err := AdbShellExecute(serial, "input", "keyevent", keyEvent)

	if err != nil {
		return err
	}

	return nil
}

func AdbInputSwipe(serial string, x string, y string, dx string, dy string, duration string) error {
	_, err := AdbShellExecute(serial, "input", "swipe", x, y, dx, dy, duration)

	if err != nil {
		return err
	}

	return nil
}

func AdbTurnOnScreen(serial string) error {
	return AdbInputKeyEvent(serial, "26")
}

func AdbIsDevicePowerOn(serial string) (bool, error) {
	output, err := AdbShellExecute(serial, "dumpsys", "power")

	if err != nil {
		return false, err
	}

	// get state
	r1, err := regexp.Compile("(mScreenOn=|state=)(\\w.*)")

	if err != nil {
		return false, err
	}

	outputStr := string(output)
	state := r1.FindString(outputStr)

	// get state info
	r2, err := regexp.Compile("(true|false|ON|OFF)")

	if err != nil {
		return false, err
	}

	state = r2.FindString(state)

	if state == "ON" || state == "true" {
		return true, nil
	} else if state == "OFF" || state == "false" {
		return false, nil
	}

	return false, errors.New("Not found")
}
