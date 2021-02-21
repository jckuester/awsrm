package internal

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/fatih/color"
)

// UserConfirmedDeletion asks the user to confirm before destroying any resources.
func UserConfirmedDeletion(r io.Reader) bool {
	log.Info("Are you sure you want to delete these resources (cannot be undone)? Only YES will be accepted.")
	fmt.Print(fmt.Sprintf("%23v", "Enter a value: "))

	var response string

	_, err := fmt.Fscanln(r, &response)
	if err != nil {
		fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))
		return false
	}

	if strings.ToLower(response) == "yes" {
		return true
	}

	return false
}
