package main

import (
	"flag"
	"log"
	"os"

	"github.com/PolarGeospatialCenter/pgcboot/pkg/distromux"
	"github.com/gorilla/mux"
)

func main() {
	flag.Parse()
	localDistro := flag.Arg(0)
	if localDistro == "" {
		localDistro, _ = os.Getwd()
	}
	r := mux.NewRouter()
	distro := distromux.NewDistroMux(localDistro, r)

	testResults, err := distro.Test()
	if err != nil {
		log.Fatalf("error while run tests: %v", err)
	}

	failed := false
	for p, r := range testResults {
		if r.Failed {
			log.Printf("Test %s failed.", p)
			log.Print(r.Output)
			failed = true
		} else {
			log.Printf("Test %s Succeeded", p)
		}
	}

	if failed {
		log.Fatal("*** Tests FAILED ***")
	}

	log.Print("*** Tests PASSED ***")
}
