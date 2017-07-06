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
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	r "gopkg.in/gorethink/gorethink.v3"

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
	log.Println(reqC)

	containerName := RandomPassword(12)
	fmt.Println(containerName)
	userPasswordDb := RandomPassword(22)
	insertUserInRethinkDB(reqC.Name, userPasswordDb)
	CreateAndGrantUserInDB(containerName, reqC.Name, userPasswordDb)

	switch reqC.Offer {
		case "bronze":
			fmt.Println("bronze case")
			idContainer = RunContainerInBackground("antoinehumbert/kiliman-horizon:1.0", containerName, reqC.Name, userPasswordDb)
		case "silver":
			fmt.Println("Silver case")
			//idContainer = RunContainerInBackground("debian")
		case "gold":
			fmt.Println("bronze case")
		//idContainer = RunContainerInBackground("portainer/portainer")
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

	val, err := exists("./srv")
	check(err)

	if !val {
		err := os.Mkdir("./srv", os.FileMode(0755))
		check(err)
	}

	customer := &ResponseCustomer{
		Name:           reqC.Name,
		Email:          reqC.Email,
		Ip_address:     ip_address_container,
		Container_name: containerName,
		Path:           containerName,
		Offer:          reqC.Offer,
		Api_Key:        RandomPassword(64),
		Db_password:    userPasswordDb,
	}

	go insertInDB(*customer)

	if err := json.NewEncoder(w).Encode(customer); err != nil {
		panic(err)
	}

	log.Println("Response Send")

}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6
	letterIdxMask = 1<<letterIdxBits - 1
	letterIdxMax  = 63 / letterIdxBits
)

var src = rand.NewSource(time.Now().UnixNano())

func RandomPassword(n int) string {
	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}

		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}

		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}


func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func insertUserInRethinkDB(idUser string, userPassword string) {

	fmt.Println("insert user in rethinkdb")
	session, err := r.Connect(r.ConnectOpts{
		Address: "172.16.21.75:28015",
		//Database: "titouhorizon19",
		//Username: "john",
		//Password: "p455w0rd",
	})


	err = r.DB("rethinkdb").Table("users").Insert(map[string]string{
		"id": idUser,
		"password": userPassword,
	}).Exec(session)


	if err != nil {
		log.Fatalln(err)
	}



	if err != nil {

		log.Fatalln(err.Error())
	}

	fmt.Println("user : " +  idUser + " insert in rethinkDb and password : "+ userPassword)

}

func CreateAndGrantUserInDB(Db_name string, idUser string, userPassword string) {

	session, err := r.Connect(r.ConnectOpts{
		Address: "172.16.21.75:28015",
	})

	resp, err := r.DBCreate(Db_name).RunWrite(session)
	if err != nil {
		fmt.Print(err)
	}

	fmt.Printf("%d DB created", resp.DBsCreated)

	err = r.DB(Db_name).Grant(idUser, map[string]bool{
		"read": true,
		"write": true,
	}).Exec(session)

	if err != nil {
		log.Fatalln(err)
	}

	sessionBis, err := r.Connect(r.ConnectOpts{
		Address: "172.16.21.75:28015",
		Database: Db_name,
		Username: idUser,
		Password: userPassword,
	})

	fmt.Println(sessionBis)

	fmt.Println("user Granted in DB " + Db_name)
}



func check(err error) {
	if err != nil {
		log.Println("Error : %s", err.Error())
		os.Exit(1)
	}
}

func RunContainerInBackground(imageName string, containerName string, idUser string, Db_password string) string {

	CreateDirectoryAndCopyConfFile(containerName, idUser, Db_password)

	//time.Sleep(2000)

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

	volumes := map[string]struct{}{
		"/Users/titou/Projets/go/src/github.com/go-kiliman/kiliman/srv/"+containerName+"/.hz/config-dev.toml": {},
		"/Users/titou/Projets/go/src/github.com/go-kiliman/kiliman/srv/"+containerName+"/config/config-dev.json": {},
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Volumes: volumes,
	}, &container.HostConfig{
		Binds: []string{
			"/Users/titou/Projets/go/src/github.com/go-kiliman/kiliman/srv/"+containerName+"/.hz/config-dev.toml:/srv/horizon/.hz/config-dev.toml",
			"/Users/titou/Projets/go/src/github.com/go-kiliman/kiliman/srv/"+containerName+"/config/config-dev.json:/srv/horizon/config/config-dev.json",
		},
	}, nil, containerName)
	if err != nil {
		fmt.Println(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		fmt.Println(err)
	}


	log.Println("Container is runnig")

	return resp.ID

}


func CreateDirectoryAndCopyConfFile(containerName string, idUser string, Db_password string) {
	srcConfigFile, err := os.Open("templates/config/config-dev.json")
	check(err)
	defer srcConfigFile.Close()

	directoryContainerPath := fmt.Sprintf("./srv/%s", containerName)
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
			lines[i] = "token_secret = '" + RandomPassword(64) + "'"
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

	if err != nil {
		log.Println(err)
	}

	fmt.Println("directory create")
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
		customer.Api_Key, customer.Db_password,
		time.Now()})

	if err != nil {
		log.Fatal(err)
	}

	log.Println("User Inserted in MongoDB")

}
