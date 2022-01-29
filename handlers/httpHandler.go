package handlers

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net/http"
)

func PeekHttpReq(reader io.Reader) (string, io.Reader, error) {
	peekedBytes := new(bytes.Buffer)
	hello, err := parseHttpReq(io.TeeReader(reader, peekedBytes))
	if err != nil {
		return "", nil, err
	}
	return hello.Host, io.MultiReader(peekedBytes, reader), nil
}

func parseHttpReq(reader io.Reader) (*http.Request, error) {
	buf := bufio.NewReader(reader)

	req, err := http.ReadRequest(buf)

	if req == nil || err != nil {
		log.Print(err)
		return nil, err
	}

	return req, nil
}
