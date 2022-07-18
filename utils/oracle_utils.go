package utils

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	_ "github.com/gofiber/jwt/v3"

	"github.com/gofiber/fiber/v2"
)

func CallBSC(context *fiber.Ctx) (string, error) {
	type Data struct {
		Result struct {
			Ethusd string
		}
	}
	resp, err := http.Get("https://api.bscscan.com/api?module=stats&action=bnbprice&apikey=3HFRCFC4.NtUQwHcUUGbdVaqohVEzLigNPEOoI6S7")
	if err != nil {
		return "520", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "520", err
	}

	d := Data{}
	json.Unmarshal([]byte(body), &d)

	return d.Result.Ethusd, nil
}
