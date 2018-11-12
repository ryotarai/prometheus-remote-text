package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	output := flag.String("output", "", "Path to output file")
	listen := flag.String("listen", ":8080", "Address to listen on")
	reopenTriggerPath := flag.String("reopen-trigger", "", "Path to trigger reopening")
	flag.Parse()

	s, err := NewServer(*output, *reopenTriggerPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening %s", *listen)
	err = http.ListenAndServe(*listen, s)
	if err != nil {
		log.Fatal(err)
	}
}
