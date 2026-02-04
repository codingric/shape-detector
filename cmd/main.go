package main

import (
	"flag"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/codingric/shape-detector/server"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	var port string
	flag.StringVar(&port, "port", "8080", "Port to listen on")
	flag.Parse()
	if err := server.StartServer(port); err != nil {
		log.Fatal().Err(err).Msg("Server error")
	}
}
