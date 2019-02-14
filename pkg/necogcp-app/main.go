// This server can run on Google App Engine.
package main

import (
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
	listenHTTP = "127.0.0.1:8080"
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
			Addr:    listenHTTP,
			Handler: server,
		},
		ShutdownTimeout: 3 * time.Minute,
	}

	return s.ListenAndServe()
}
