package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func AreYouSure(msg string) bool {
	fmt.Printf("%s [y/n] ", msg)

	var reader = bufio.NewReader(os.Stdin)
	read, _ := reader.ReadString('\n')

	return strings.TrimFunc(read, func(r rune) bool {
		return r == '\n'
	}) == "y"
}
