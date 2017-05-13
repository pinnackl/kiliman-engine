package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	middle "github.com/go-kiliman/kiliman/middlewares"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"io/ioutil"
)

type RequestCustomer struct {
	Name  string `json:"name"`
	Offer string `json:"offer"`
	Email string `json:"email"`
}

type ResponseCustomer struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	Ip_address     string `json:"ip_address"`
	Container_name string `json:"container_name"`
	Path           string `json:"path"`
	Offer          string `json:"offer"`
	Api_Key        string `json:"api-key"`
}

type Customer struct {
	ID             bson.ObjectId
	Name           string
	Ip_address     string
	Container_name string
	Path           string
	Offer          string
	Email          string
	Api_Key        string
	Created_at     time.Time
}

func main() {

	router := mux.NewRouter()
	log.Println("Router running and listening")

	Myserver := &middle.MyServer{router}

	router.HandleFunc("/new-cms", CreateContainerEndpoint).Methods("POST")
	http.Handle("/", Myserver)

	log.Fatal(http.ListenAndServe(":12345", nil))
}

func CreateContainerEndpoint(w http.ResponseWriter, req *http.Request) {

	var reqC RequestCustomer
	var idContainer string
	_ = json.NewDecoder(req.Body).Decode(&reqC)

	defer req.Body.Close()
	log.Println(reqC)

	switch reqC.Offer {

	case "bronze":
		idContainer = RunContainerInBackground("alpine")
	case "silver":
		idContainer = RunContainerInBackground("debian")
	case "gold":
		idContainer = RunContainerInBackground("portainer/portainer")
	}

	cmdStr := "docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' " + idContainer
	out, _ := exec.Command("/bin/sh", "-c", cmdStr).Output()
	tmp_ip_address_container := fmt.Sprintf("%s", out)
	ip_address_container := strings.Replace(tmp_ip_address_container, "\n", "", 2)

	cmdStrName := "docker inspect -f '{{.Name}}' " + idContainer
	outName, _ := exec.Command("/bin/sh", "-c", cmdStrName).Output()
	tmp_name_container := fmt.Sprintf("%s", outName)
	tmp_containerName := strings.Replace(tmp_name_container, "/", "", 1)
	containerName := strings.Replace(tmp_containerName, "\n", "", 2)

	val, err := exists("./srv")
	check(err)

	if !val {
		err := os.Mkdir("./srv", os.FileMode(0755))
		check(err)
	}

	CreateDirectoryAndCopyConfFile(containerName)

	customer := &ResponseCustomer{
		Name:           reqC.Name,
		Email:          reqC.Email,
		Ip_address:     ip_address_container,
		Container_name: containerName,
		Path:           containerName,
		Offer:          reqC.Offer,
		Api_Key:        RandStringRunes(64),
	}

	go insertInDB(*customer)

	if err := json.NewEncoder(w).Encode(customer); err != nil {
		panic(err)
	}

	log.Println("Response Send")

}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil { return true, nil }
	if os.IsNotExist(err) { return false, nil }
	return true, err
}

func CreateDirectoryAndCopyConfFile(containerName string) {
	srcConfigFile, err := os.Open("./templates/config/config-dev.json")
	check(err)
	defer srcConfigFile.Close()

	directoryContainerPath := fmt.Sprintf("./srv/%s", containerName )
	os.Mkdir(directoryContainerPath, os.FileMode(0755))

	directoryHzPath := fmt.Sprintf("%s/.hz", directoryContainerPath)
	os.Mkdir(directoryHzPath, os.FileMode(0755))

	directoryConfigPath := fmt.Sprintf("%s/config", directoryContainerPath)
	os.Mkdir(directoryConfigPath, os.FileMode(0755))

	configFilePath := fmt.Sprintf("%s/config-dev.json", directoryConfigPath)
	destConfigFile, err := os.Create(configFilePath) // creates if file doesn't exist
	check(err)
	defer destConfigFile.Close()

	_, err = io.Copy(destConfigFile, srcConfigFile) // check first var for number of bytes copied
	check(err)
	err = destConfigFile.Sync()
	check(err)

	input, err := ioutil.ReadFile("./templates/.hz/config-dev.toml")
	if err != nil {
		log.Fatalln(err)
	}
	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, "project_name =") {
			lines[i] = "project_name = '" + containerName +"'"
		}
		if strings.Contains(line, "token_secret =") {
			log.Println("coucou")
			lines[i] = "token_secret = '" +  RandStringRunes(64) +"'"
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(directoryHzPath+"/config-dev.toml", []byte(output), 0644)
	if err != nil {
		log.Fatalln(err)
	}


}

func check(err error) {
	if err != nil {
		fmt.Println("Error : %s", err.Error())
		os.Exit(1)
	}
}

func RunContainerInBackground(imageName string) string {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, out)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
	}, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	log.Println("Container is runnig")

	return resp.ID

}

func insertInDB(customer ResponseCustomer) {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)

	c := session.DB("test-go").C("customer-cms")

	err = c.Insert(&Customer{bson.NewObjectId(), customer.Name,
				customer.Ip_address, customer.Container_name,
				customer.Path, customer.Offer, customer.Email,
				customer.Api_Key,time.Now()})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Inserted in DB")

}
