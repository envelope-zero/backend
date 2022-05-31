package main

import (
	"github.com/envelope-zero/backend/internal/router"
	"github.com/rs/zerolog/log"
)

func main() {
	r, err := router.Router()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	if err := r.Run(); err != nil {
		log.Fatal().Msg(err.Error())
	}
}
