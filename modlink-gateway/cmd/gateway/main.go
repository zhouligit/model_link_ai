package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/modlinkcloud/modlink-gateway/internal/config"
	"github.com/modlinkcloud/modlink-gateway/internal/gateway"
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
	db.SetMaxOpenConns(50)
	if err := db.Ping(); err != nil {
		log.Fatal("database ping:", err)
	}
	st := store.New(db)
	log.Printf("gateway listening %s", cfg.Server.GatewayListen)
	log.Fatal(http.ListenAndServe(cfg.Server.GatewayListen, gateway.NewRouter(cfg, st)))
}
