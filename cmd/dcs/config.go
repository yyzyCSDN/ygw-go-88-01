package main

import "flag"

type Config struct {
	Addr string
	Dir  string
}

func LoadConfig() Config {
	addr := flag.String("addr", "127.0.0.1:8090", "listen address")
	dir := flag.String("dir", "./data", "data directory")
	flag.Parse()
	return Config{Addr: *addr, Dir: *dir}
}
