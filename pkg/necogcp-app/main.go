// This server can run on Google App Engine.
package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp"
	"github.com/cybozu-go/neco/gcp/app"
	"github.com/cybozu-go/well"
	yaml "gopkg.in/yaml.v2"
)

const (
	cfgFile    = ".necogcp.yml"
	listenAddr = "0.0.0.0"
)

var (
	// NOTE: listen port is randomly assigned, It has to get a port from PORT environment variable
	listenPort = os.Getenv("PORT")
)

func main() {
	well.LogConfig{}.Apply()

	// seed math/random
	rand.Seed(time.Now().UnixNano())

	err := subMain()
	if err != nil {
		log.ErrorExit(err)
	}
	well.Stop()
	err = well.Wait()
	if !well.IsSignaled(err) && err != nil {
		log.ErrorExit(err)
	}
}

func subMain() error {
	cfg := gcp.NewConfig()
	f, err := os.Open(cfgFile)
	if err != nil {
		return err
	}
	err = yaml.NewDecoder(f).Decode(cfg)
	if err != nil {
		return err
	}
	f.Close()

	server := app.NewServer(cfg)
	s := &well.HTTPServer{
		Server: &http.Server{
			Addr:    fmt.Sprintf("%s:%s", listenAddr, listenPort),
			Handler: server,
		},
		ShutdownTimeout: 3 * time.Minute,
	}

	return s.ListenAndServe()
}
