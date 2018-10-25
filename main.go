package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {
	output := flag.String("output", "", "Path to output file")
	listen := flag.String("listen", ":8080", "Address to listen on")
	pidfile := flag.String("pidfile", "", "Path to pid file")
	flag.Parse()

	if *pidfile != "" {
		ioutil.WriteFile(*pidfile, []byte(strconv.Itoa(os.Getpid())), 0644)
	}

	s, err := NewServer(*output)
	if err != nil {
		log.Fatal(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1)
	go func() {
		for {
			<-sigChan
			log.Printf("Received SIGUSR1")
			err := s.OpenFile()
			if err != nil {
				log.Fatal(err)
			}
		}
	}()

	log.Printf("Listening %s", *listen)
	err = http.ListenAndServe(*listen, s)
	if err != nil {
		log.Fatal(err)
	}
}
