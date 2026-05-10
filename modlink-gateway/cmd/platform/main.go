package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/modlinkcloud/modlink-gateway/internal/config"
	"github.com/modlinkcloud/modlink-gateway/internal/platform"
	"github.com/modlinkcloud/modlink-gateway/internal/store"
)

func main() {
	cfgPath := os.Getenv("MODLINK_CONFIG")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("mysql", cfg.Database.DSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(20)
	if err := db.Ping(); err != nil {
		log.Fatal("database ping:", err)
	}
	st := store.New(db)
	log.Printf("platform listening %s", cfg.Server.PlatformListen)
	log.Fatal(http.ListenAndServe(cfg.Server.PlatformListen, platform.NewRouter(cfg, st)))
}
