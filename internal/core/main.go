package core

import (
	"fmt"

	"github.com/farhapartex/bolt-load-balancer/internal/config"
	"github.com/farhapartex/bolt-load-balancer/internal/logger"
)

const (
	VERSION           = "1.0.0"
	DeafultConfigFile = "config.yaml"
)

type Application struct {
	config        *config.Config
	load_balancer *LB
	logger        *logger.Logger
}

func main() {
	fmt.Println("Hello LB")
}
