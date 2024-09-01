package main

import (
	"context"
	"fmt"
	"os"

	"github.com/vikaskumar1187/saas-project/services/publisher/pkg/logger"
)

var build = "develop"

func main() {

	log, err := logger.New("PUBLISHER-API")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer log.Sync()

	if err := run(log); err != nil {
		log.Errorw("startup", "ERROR", err)
		log.Sync()
		os.Exit(1)
	}

	// -------------------------------------------------------------------------

	ctx := context.Background()

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "startup", "msg", err)
		return
	}
}
