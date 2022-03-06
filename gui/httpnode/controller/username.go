package controller

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/peer"
	"io/ioutil"
	"net/http"
)

// NewUsername returns a new initialized username.
func NewUsername(node peer.Peer, log *zerolog.Logger) username {
	return username{
		node: node,
		log:  log,
	}
}

type username struct {
	node peer.Peer
	log  *zerolog.Logger
}

func (u username) MyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method != http.MethodGet {
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
			return
		}
		name, err := u.node.GetUsername()
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Write([]byte(name))
	}
}

func (u username) SetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
		case http.MethodPost:
			u.set(w, r)
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
		}
	}
}

func (u username) set(w http.ResponseWriter, r *http.Request) {
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

	err = u.node.SetUsername(arguments[0])
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to setup the username: %v", err), http.StatusBadRequest)
		return
	}
}

func (u username) SearchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
		case http.MethodPost:
			u.search(w, r)
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
		}
	}
}

func (u username) search(w http.ResponseWriter, r *http.Request) {
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

	results, err := u.node.SearchUsername(arguments[0])
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to find a match: %v", err), http.StatusBadRequest)
		return
	}

	json, err := json.Marshal(results)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal the results: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}
