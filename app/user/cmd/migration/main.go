package main

import (
	"flag"

	"do-template-app/pkg/migrate"
)

func main() {
	confPath := flag.String("conf", "config/user/local.yml", "config path")
	flag.Parse()

	migrate.Run(*confPath, "user")
}
