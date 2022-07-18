package utils

import (
	"bufio"
	"io/ioutil"
	"os"
	"strings"

	_ "github.com/gofiber/jwt/v3"

	"github.com/gofiber/fiber/v2"
)

func GetNodeName(context *fiber.Ctx, index uint) string {
	var file *os.File
	var err error
	if file, err = os.Open("./nebulas_shuffled.txt"); err != nil {
		return "error"
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	contents, _ := ioutil.ReadAll(reader)
	lines := strings.Split(string(contents), "\n")
	return lines[index]
}
