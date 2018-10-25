package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

type Server struct {
	path   string
	writer io.WriteCloser
	mutex  sync.Mutex
}

func NewServer(path string) (*Server, error) {
	s := &Server{
		path:  path,
		mutex: sync.Mutex{},
	}
	err := s.OpenFile()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) handleWrite(w http.ResponseWriter, r *http.Request) {
	compressed, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reqBuf, err := snappy.Decode(nil, compressed)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req prompb.WriteRequest
	if err := proto.Unmarshal(reqBuf, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.writeTimeseries(req.Timeseries)
}

func (s *Server) writeTimeseries(tss []*prompb.TimeSeries) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, ts := range tss {
		for _, sample := range ts.Samples {
			fields := []string{
				fmt.Sprintf("timestamp:%d", sample.Timestamp),
				fmt.Sprintf("value:%f", sample.Value),
			}
			for _, l := range ts.Labels {
				name := strings.Replace(l.Name, ":", "__comma__", -1)
				fields = append(fields, fmt.Sprintf("%s:%s", name, l.Value))
			}

			fmt.Fprintln(s.writer, strings.Join(fields, "\t"))
		}
	}
	return nil
}

func (s *Server) OpenFile() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.writer != nil {
		err := s.writer.Close()
		if err != nil {
			log.Printf("Closing a opened file failed: %s", err)
		}
	}

	log.Printf("Opening an output file %s", s.path)
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	s.writer = f
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/write" {
		s.handleWrite(w, r)
	} else {
		http.NotFound(w, r)
	}
}
