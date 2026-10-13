package main

import (
	"flag"

	"do-template-app/pkg/migrate"
)

func main() {
	confPath := flag.String("conf", "config/admin/local.yml", "config path")
	flag.Parse()

	migrate.Run(*confPath, "admin")
}
