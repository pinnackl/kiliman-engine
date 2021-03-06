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
	"io/ioutil"

	"errors"
	"github.com/go-kiliman/kiliman/config"
	"github.com/go-kiliman/kiliman/utils"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
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
	Db_password    string `json:"db_password"`
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
	Db_password    string
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

	err := searchIfUserExist(reqC.Email, 0, 50)

	if err != nil {
		log.Println("error")
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	containerName := utils.RandomPassword(12)
	log.Println("Container Name : " + containerName)
	userPasswordDb := utils.RandomPassword(22)

	utils.InsertUserInRethinkDB(reqC.Name, userPasswordDb)
	utils.CreateAndGrantUserInDB(containerName, reqC.Name, userPasswordDb)

	switch reqC.Offer {
	case "bronze":
		idContainer = RunContainerInBackground("antoinehumbert/kiliman-horizon:1.1", containerName, reqC.Name, userPasswordDb)
	case "silver":
		idContainer = RunContainerInBackground("antoinehumbert/kiliman-horizon:1.1", containerName, reqC.Name, userPasswordDb)
	case "gold":
		idContainer = RunContainerInBackground("antoinehumbert/kiliman-horizon:1.1", containerName, reqC.Name, userPasswordDb)
	}

	cmdStr := "docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' " + idContainer
	out, _ := exec.Command("/bin/sh", "-c", cmdStr).Output()
	tmp_ip_address_container := fmt.Sprintf("%s", out)
	ip_address_container := strings.Replace(tmp_ip_address_container, "\n", "", 2)

	cmdStrName := "docker inspect -f '{{.Name}}' " + idContainer
	outName, _ := exec.Command("/bin/sh", "-c", cmdStrName).Output()
	tmp_name_container := fmt.Sprintf("%s", outName)
	tmp_containerName := strings.Replace(tmp_name_container, "/", "", 1)
	containerName = strings.Replace(tmp_containerName, "\n", "", 2)

	cmdHz := "docker exec -ti " + containerName + " hz schema apply --project-name " + containerName + " .hz/schema.toml -c " + config.CNF["IP_HOST"] + ":28015"
	log.Println("command : " + cmdHz)
	output, _ := exec.Command("/bin/sh", "-c", cmdHz).Output()
	log.Println(output)


	customer := &ResponseCustomer{
		Name:           reqC.Name,
		Email:          reqC.Email,
		Ip_address:     ip_address_container,
		Container_name: containerName,
		Path:           containerName,
		Offer:          reqC.Offer,
		Api_Key:        utils.RandomPassword(64),
		Db_password:    userPasswordDb,
	}

	insertInDB(*customer)

	if err := json.NewEncoder(w).Encode(customer); err != nil {
		log.Println(err)
	}

	js, err := json.Marshal(customer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusCreated)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

	log.Println("Response Send")
	log.Println(customer)
}

func RunContainerInBackground(imageName string, containerName string, idUser string, Db_password string) string {

	CreateDirectoryAndCopyConfFile(containerName, idUser, Db_password)

	time.Sleep(6000)

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		log.Println(err)
	}

	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		log.Println(err)
	}
	io.Copy(os.Stdout, out)

	volumes := map[string]struct{}{
		os.Getenv("PWD") + "/srv/" + containerName + "/.hz/config-dev.toml":    {},
		os.Getenv("PWD") + "/srv/" + containerName + "/config/config-dev.json": {},
		os.Getenv("PWD") + "/srv/" + containerName + "/config.js":              {},
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:   imageName,
		Volumes: volumes,
		Env: []string{
			"VIRTUAL_HOST=" + config.CNF["IP_HOST"],
			"CONTAINER_NAME=" + containerName,
		},
	}, &container.HostConfig{
		Binds: []string{
			os.Getenv("PWD") + "/srv/" + containerName + "/.hz/config-dev.toml:/srv/horizon/.hz/config-dev.toml",
			os.Getenv("PWD") + "/srv/" + containerName + "/config/config-dev.json:/srv/horizon/config/config-dev.json",
			os.Getenv("PWD") + "/srv/" + containerName + "/chateau/config.js:/srv/horizon/config.js",
		},
	}, nil, containerName)
	if err != nil {
		log.Println(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Println(err)
	}

	log.Println("Container is runnig")

	return resp.ID

}

func CreateDirectoryAndCopyConfFile(containerName string, idUser string, Db_password string) {

	val, err := utils.Exists("./srv")
	utils.Check(err)

	if !val {
		err := os.Mkdir("./srv", os.FileMode(0755))
		utils.Check(err)
	}

	srcConfigFile, err := os.Open("templates/config/config-dev.json")
	utils.Check(err)
	defer srcConfigFile.Close()

	directoryContainerPath := fmt.Sprintf(os.Getenv("PWD")+"/srv/%s", containerName)
	os.Mkdir(directoryContainerPath, os.FileMode(0755))

	directoryHzPath := fmt.Sprintf("%s/.hz", directoryContainerPath)
	os.Mkdir(directoryHzPath, os.FileMode(0755))

	directoryChateauPath := fmt.Sprintf("%s/chateau", directoryContainerPath)
	os.Mkdir(directoryChateauPath, os.FileMode(0755))

	directoryConfigPath := fmt.Sprintf("%s/config", directoryContainerPath)
	os.Mkdir(directoryConfigPath, os.FileMode(0755))

	configFilePath := fmt.Sprintf("%s/config-dev.json", directoryConfigPath)
	destConfigFile, err := os.Create(configFilePath) // creates if file doesn't exist
	utils.Check(err)

	defer destConfigFile.Close()

	_, err = io.Copy(destConfigFile, srcConfigFile) // check first var for number of bytes copied
	utils.Check(err)
	err = destConfigFile.Sync()
	utils.Check(err)

	input, err := ioutil.ReadFile("templates/.hz/config-dev.toml")
	if err != nil {
		log.Fatalln(err)
	}
	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, "project_name =") {
			lines[i] = "project_name = '" + containerName + "'"
		}
		if strings.Contains(line, "token_secret =") {
			lines[i] = "token_secret = '" + utils.RandomPassword(64) + "'"
		}

		if strings.Contains(line, "rdb_user=") {
			lines[i] = "rdb_user= '" + idUser + "'"
		}
		if strings.Contains(line, "rdb_password=") {
			lines[i] = "rdb_password= '" + Db_password + "'"
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(directoryHzPath+"/config-dev.toml", []byte(output), 0644)
	if err != nil {
		log.Fatalln(err)
	}

	input, err = ioutil.ReadFile("templates/chateau/config.js")
	if err != nil {
		log.Fatalln(err)
	}
	lines = strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, "exports.user = ") {
			lines[i] = "exports.user = '" + idUser + "'"
		}
		if strings.Contains(line, "exports.password = ") {
			lines[i] = "exports.password = '" + Db_password + "'"
		}

	}
	output = strings.Join(lines, "\n")
	err = ioutil.WriteFile(directoryChateauPath+"/config.js", []byte(output), 0644)
	if err != nil {
		log.Println(err)
	}

	log.Println("Directory created")
}

func insertInDB(customer ResponseCustomer) {
	session, err := mgo.Dial("localhost")
	if err != nil {
		log.Println(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)

	c := session.DB("go-kiliman").C("customer-cms")

	err = c.Insert(&Customer{bson.NewObjectId(), customer.Name,
		customer.Ip_address, customer.Container_name,
		customer.Path, customer.Offer, customer.Email,
		customer.Api_Key, customer.Db_password,
		time.Now()})

	if err != nil {
		log.Fatal(err)
	}

	log.Println("User Inserted in MongoDB")

}

func searchIfUserExist(email string, skip int, limit int) error {
	searchResults, searchErr := SearchPerson(bson.M{"email": email}, skip, limit)

	if len(searchResults) > 0 {
		log.Println("User already exist")
		return errors.New("User already exist")
	} else {
		log.Println("User doesn't exist ")
		return nil
	}
	log.Println(searchErr)

	return nil
}

func SearchPerson(q interface{}, skip int, limit int) (searchResults []Customer, searchErr string) {
	searchErr = ""
	searchResults = []Customer{}
	query := func(c *mgo.Collection) error {
		fn := c.Find(q).Skip(skip).Limit(limit).All(&searchResults)
		if limit < 0 {
			fn = c.Find(q).Skip(skip).All(&searchResults)
		}
		return fn
	}
	search := func() error {
		return withCollection("customer-cms", query)
	}
	err := search()
	if err != nil {
		searchErr = "Database Error"
	}
	return
}

func withCollection(collection string, s func(*mgo.Collection) error) error {
	session, err := mgo.Dial("localhost")
	if err != nil {
		log.Println(err)
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("go-kiliman").C(collection)
	return s(c)
}
