package main

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/valve"
	"github.com/spf13/viper"
	"gitlab-ci.detik.com/datacore/gonotifikasi/models"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	// signaling
	valv := valve.New()
	baseCtx := valv.Context()

	srv := http.Server{
		Addr: fmt.Sprintf(":%d", hostPort),
		Handler: chi.ServerBaseContext(baseCtx, r),
	}

	go func() {
		sigint := make(chan os.Signal, 1)

		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)
		// sigterm signal sent from kubernetes
		signal.Notify(sigint, syscall.SIGTERM)

		for range sigint{
			log.Println("Server is shutdown...please wait...")

			// first valv
			valv.Shutdown(20 * time.Second)

			// create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			// kill channel if exist bellow

			if err := srv.Shutdown(ctx); err != nil {
				// Error from closing listeners, or context timeout:
				log.Printf("HTTP server Shutdown with error: %v", err)
			}
			select {
			case <-time.After(21 * time.Second):
				log.Println("Not all connections done")
			case <-ctx.Done():

			}
		}
	}()


	log.Printf("Server listen to port %d\n", hostPort)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("Error ListenAndServe: %v", err)
	}

	log.Println("Server exited")
}
