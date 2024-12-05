package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

func Login() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter access token: ")
	accessToken, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading access token:", err)
		return err
	}
	accessToken = strings.TrimSpace(accessToken)
	config.SetAccessToken(accessToken)

	fmt.Print("Enter config backend url: ")
	cb, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading access token:", err)
		return err
	}

	cb = strings.TrimSpace(cb)
	config.SetConfigBackendURL(cb)

	return nil
}
