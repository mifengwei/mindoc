package commands

import (
	"encoding/json"
	"fmt"
	"github.com/mindoc-org/mindoc/models"
	"io"
	"net/http"
	"os"

	"github.com/mindoc-org/mindoc/pkg/logger"
	"github.com/mindoc-org/mindoc/conf"
)

// 检查最新版本.
func CheckUpdate() {

	fmt.Println("MinDoc current version => ", conf.VERSION)

	resp, err := http.Get("https://api.github.com/repos/mindoc-org/mindoc/tags")

	if err != nil {
		logger.Error("CheckUpdate => ", err)
		os.Exit(1)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("CheckUpdate => ", err)
		os.Exit(1)
	}

	var result []*struct {
		Name string `json:"name"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		logger.Error("CheckUpdate => ", err)
		os.Exit(0)
	}

	if len(result) > 0 {
		fmt.Println("MinDoc last version => ", result[0].Name)
	}

	os.Exit(0)

}

func Update() {
	fmt.Println("Update...")
	RegisterDataBase()
	RegisterModel()
	UpdateInitialization()
	fmt.Println("Update Successfully!")
	os.Exit(0)
}
func UpdateInitialization() {
	err := models.NewOption().Update()
	if err != nil {
		panic(err.Error())
	}
}
