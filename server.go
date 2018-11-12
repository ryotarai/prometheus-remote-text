package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

type Server struct {
	path               string
	data               io.WriteCloser
	mutex              sync.Mutex
	processingRequests int64
	triggerFile        *TriggerFile

	ReopenTriggerPath string
}

type sampleRecord struct {
	Timestamp int64             `json:"timestamp"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
}

func NewServer(path string, triggerFilePath string) (*Server, error) {
	s := &Server{
		path:  path,
		mutex: sync.Mutex{},
	}

	err := s.ReopenFile()
	if err != nil {
		return nil, err
	}

	if triggerFilePath != "" {
		s.triggerFile, err = NewTriggerFile(triggerFilePath)
		if err != nil {
			return nil, err
		}
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

	atomic.AddInt64(&s.processingRequests, 1)
	log.Printf("Writing %d timeseries... (Processing requests: %d)", len(req.Timeseries), s.processingRequests)
	s.writeTimeseries(req.Timeseries)
	atomic.AddInt64(&s.processingRequests, -1)
}

func (s *Server) writeTimeseries(tss []*prompb.TimeSeries) error {
	err := s.reopenFileIfTriggered()
	if err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, ts := range tss {
		for _, sample := range ts.Samples {
			r := sampleRecord{}
			r.Timestamp = sample.Timestamp
			r.Value = sample.Value
			r.Labels = map[string]string{}
			for _, l := range ts.Labels {
				r.Labels[l.Name] = l.Value
			}

			j := json.NewEncoder(s.data)
			err := j.Encode(r)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Server) reopenFileIfTriggered() error {
	if s.triggerFile != nil {
		touched, err := s.triggerFile.CheckIfTouched()
		if err != nil {
			return err
		}

		if touched {
			err = s.ReopenFile()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Server) ReopenFile() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data != nil {
		err := s.data.Close()
		if err != nil {
			log.Printf("Closing a opened file failed: %s", err)
		}
	}

	log.Printf("Opening an output file %s", s.path)
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	s.data = f
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/write" {
		s.handleWrite(w, r)
	} else {
		http.NotFound(w, r)
	}
}
