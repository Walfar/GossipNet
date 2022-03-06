package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/peer"
	"io"
	"io/ioutil"
	"net/http"
)

// NewAvatar returns a new initialized username.
func NewAvatar(node peer.Peer, log *zerolog.Logger) avatar {
	return avatar{
		node: node,
		log:  log,
	}
}

type avatar struct {
	node peer.Peer
	log  *zerolog.Logger
}

func (a avatar) MyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
		case http.MethodGet:
			a.getMyAvatar(w, r)
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
		}
	}
}

func (a avatar) getMyAvatar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	dataBytes, err := a.node.DownloadAvatar(a.node.GetLocalhost())

	if err != nil {
		http.Error(w, "No avatar found.",
			http.StatusNotFound)
		return
	}

	data := bytes.NewReader(dataBytes)
	if _, err = io.Copy(w, data); err != nil {
		http.Error(w, err.Error(),
			http.StatusNotFound)
		return
	}
}

func (a avatar) UploadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			a.uploadAvatar(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
		}
	}
}

func (a avatar) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to upload: %v", err), http.StatusBadRequest)
		return
	}

	reader := bytes.NewReader(buf)
	mh, err := a.node.UploadAvatar(reader)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to upload: %v", err), http.StatusBadRequest)
		return
	}

	w.Write([]byte(mh))
}

func (a avatar) DownloadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			a.downloadAvatar(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
		}
	}
}

func (a avatar) downloadAvatar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read body: %v", err),
			http.StatusInternalServerError)
		return
	}

	arguments := [1]string{}
	err = json.Unmarshal(buf, &arguments)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to unmarshal arguments: %v", err),
			http.StatusInternalServerError)
		return
	}

	dataBytes, err := a.node.DownloadAvatar(arguments[0])
	if err != nil {
		http.Error(w, "No avatar found.",
			http.StatusNotFound)
		return
	}
	
	data := bytes.NewReader(dataBytes)
	if _, err = io.Copy(w, data); err != nil {
		http.Error(w, err.Error(),
			http.StatusNotFound)
		return
	}
}
