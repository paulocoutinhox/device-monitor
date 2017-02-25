package utils

import (
	"os/exec"
	"errors"
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

func AdbInputSwipe(serial string, x string, y string, dx string, dy string, duration string) error {
	_, err := AdbShellExecute(serial, "input", "swipe", x, y, dx, dy, duration)

	if err != nil {
		return err
	}

	return nil
}
