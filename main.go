package main

import (
	"github.com/rs/zerolog/log"

	"github.com/envelope-zero/backend/internal/controllers"
)

func main() {
	r, err := controllers.Router()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	if err := r.Run(); err != nil {
		log.Fatal().Msg(err.Error())
	}
}
