package utils

import (
	"log"
	r "gopkg.in/gorethink/gorethink.v3"
	"github.com/go-kiliman/kiliman/config"
)

func CreateAndGrantUserInDB(Db_name string, idUser string, userPassword string) {

	session, err := r.Connect(r.ConnectOpts{
		Address: config.CNF["IP_HOST"]+":28015",
	})

	resp, err := r.DBCreate(Db_name).RunWrite(session)
	if err != nil {
		log.Println(err)
	}
	

	log.Printf("%d DB created", resp.DBsCreated)

	err = r.DB(Db_name).Grant(idUser, map[string]bool{
		"read":  true,
		"write": true,
		"config": true,
	}).Exec(session)

	if err != nil {
		log.Fatalln(err)
	}

	_, err = r.Connect(r.ConnectOpts{
		Address:  config.CNF["IP_HOST"]+":28015",
		Database: Db_name,
		Username: idUser,
		Password: userPassword,
	})

	log.Println("User Granted for DB " + Db_name)
}



func InsertUserInRethinkDB(idUser string, userPassword string) {

	session, err := r.Connect(r.ConnectOpts{
		Address: config.CNF["IP_HOST"]+":28015",
	})

	err = r.DB("rethinkdb").Table("users").Insert(map[string]string{
		"id":       idUser,
		"password": userPassword,
	}).Exec(session)

	if err != nil {
		log.Fatalln(err)
	}

	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Println("user : " + idUser + " insert in rethinkDb and password : " + userPassword)
}
