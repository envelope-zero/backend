package main

import (
	"github.com/envelope-zero/backend/internal/routing"
)

func main() {
	r := routing.Router()
	r.Run()
}
