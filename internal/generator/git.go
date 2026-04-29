package generator

import (
	"errors"
	"fmt"
	"os/exec"
)

func GetDiff() (string, error) {
	staging, err := haveStagingChanges()
	if err != nil {
		return "", err
	}
	if !staging {
		return "", fmt.Errorf("There is not staged changes. Please add the files you want to commit and run commitgen again.")
	}

	var diff string
	diffCmd := exec.Command("git", "diff", "--cached")
	out, err := diffCmd.Output()
	if err != nil {
		return "", err
	}
	diff = string(out)

	return diff, nil
}

func haveStagingChanges() (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 1 {
				return true, nil
			}
		}
		return false, err
	}
	return false, nil
}
