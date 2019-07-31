package main

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/spf13/viper"
	"gitlab-ci.detik.com/datacore/gonotifikasi/models"
	"log"
	"net/http"
	"time"
)

func init()  {
	viper.SetConfigFile(`config/config.local.json`)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	if viper.GetBool(`debug`) {
		log.Println("Service RUN on DEBUG mode")
	}
}

func main()  {

	hostPort := viper.GetInt(`host.port`)

	dbHost := viper.GetString(`db.master.host`)
	dbUser := viper.GetString(`db.master.user`)
	dbPass := viper.GetString(`db.master.pass`)
	dbName := viper.GetString(`db.master.name`)
	dbPort := viper.GetInt(`db.master.port`)
	dbConn := viper.GetInt(`db.master.conn`)

	postgresConnection := fmt.Sprintf("host=%s port=%d user=%s "+ "password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	// make db connection
	db, err := models.DBConnection(postgresConnection)
	if err != nil {
		log.Panicf("Terjadi masalah pada koneksi database. %s\n", err.Error())
	}

	db.SetMaxOpenConns(dbConn)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi"))
	})

	srv := http.Server{Addr: fmt.Sprintf(":%d", hostPort)}

	log.Printf("Server listen to port %d\n", hostPort)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("Error ListenAndServe: %v", err)
	}
}
