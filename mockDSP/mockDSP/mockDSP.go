package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

type DSPRes struct {
	Request_id string
	Url        string
	Price      int
}

func DSPHandle(w http.ResponseWriter, req *http.Request) {

	if req.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	length, err := strconv.Atoi(req.Header.Get("Content-Length"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body := make([]byte, length)
	length, err = req.Body.Read(body)
	if err != nil && err != io.EOF {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var sspreq map[string]interface{}
	err = json.Unmarshal(body[:length], &sspreq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Printf("%v\n", sspreq)

	dspResponse := DSPRes{"SSP-Name-UUID", "http://example.com/ad/image/test/123", 50}

	res, err := json.Marshal(dspResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(res)
	io.WriteString(w, "\n")

	w.WriteHeader(http.StatusOK)

}

func main() {
	http.HandleFunc("/dsp", DSPHandle)
	log.Fatal(http.ListenAndServe(":8084", nil))
}
