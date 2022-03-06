package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/gui/httpnode/types"
	"go.dedis.ch/cs438/peer"

	"go.dedis.ch/cs438/transport"
)

// NewMessaging returns a new initialized messaging.
func NewMessaging(node peer.Peer, log *zerolog.Logger) messaging {
	return messaging{
		node: node,
		log:  log,
	}
}

type messaging struct {
	node peer.Peer
	log  *zerolog.Logger
}

func (m messaging) PeerHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			m.peerPost(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			return
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
			return
		}
	}
}

func (m messaging) RoutingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			m.routingGet(w, r)
		case http.MethodPost:
			m.routingPost(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			return
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
			return
		}
	}
}

func (m messaging) UnicastHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			m.unicastPost(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			return
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
			return
		}
	}
}

func (m messaging) BroadcastHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			m.broadcastPost(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			return
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
			return
		}
	}
}

// [
// 	 "127.0.0.1:xxx",
// 	 "127.0.0.1:yyy"
// ]
func (m messaging) peerPost(w http.ResponseWriter, r *http.Request) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	res := types.AddPeerArgument{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal addPeerArgument: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	m.log.Info().Msgf("got the following peers: %v", res)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	m.node.AddPeer(res...)
}

func (m messaging) routingGet(w http.ResponseWriter, r *http.Request) {
	table := m.node.GetRoutingTable()

	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed tp parse form: %v", err),
			http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Form.Get("graphviz") == "on" {
		table.DisplayGraph(w)
	} else {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "\t")

		err = enc.Encode(&table)
		if err != nil {
			http.Error(w, "failed to marshal routing table", http.StatusInternalServerError)
			return
		}
	}
}

// types.SetRoutingEntryArgument:
// {
//     "Origin": "XXX",
//     "RelayAddr": "XXX"
// }
func (m messaging) routingPost(w http.ResponseWriter, r *http.Request) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	m.log.Info().Msgf("got the following message: %s", buf)

	res := types.SetRoutingEntryArgument{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal addPeerArgument: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	m.log.Info().Msgf("got the following message: %v", res)

	m.node.SetRoutingEntry(res.Origin, res.RelayAddr)
}

// "Msg" can be of any type, based on "Type". Here a chat message.
//
// {
//     "Dest": "127.0.0.1:xxxx",
//     "Type": "chat",
//     "Msg": {
//         "Message": "Hello, world"
//     }
// }
func (m messaging) unicastPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	m.log.Info().Msgf("got the following message: %s", buf)

	res := types.UnicastArgument{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal unicast argument: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	err = m.node.Unicast(res.Dest, res.Msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (m messaging) broadcastPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	m.log.Info().Msgf("broadcast got the following message: %s", buf)

	res := transport.Message{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal broadcast argument: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	err = m.node.Broadcast(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

//=============================================Friend request====================================================================

func (m messaging) FriendRequestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			m.friendRequestPost(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			return
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
			return
		}
	}
}

func (m messaging) EncryptedMessageHandler() http.HandlerFunc {
	m.log.Info().Msgf("handler ecnrypted msg")
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			m.msgToFriendPost(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			return
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
			return
		}
	}
}

func (m messaging) SendPositiveResponseHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			m.positiveResponsePost(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			return
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
			return
		}
	}
}

func (m messaging) SendNegativeResponseHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			m.negativeResponsePost(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			return
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
			return
		}
	}
}

func (m messaging) friendRequestPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	m.log.Info().Msgf("got the following message: %s", buf)

	res := types.SendFriendRequestArgument{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal unicast argument: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	err = m.node.SendFriendRequest(res.Dest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (m messaging) msgToFriendPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	m.log.Info().Msgf("got the following message: %s", buf)

	res := types.UnicastArgument{}

	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal unicast argument: "+err.Error(),
			http.StatusInternalServerError)
		return
	}
	err = m.node.EncryptedMessage(res.Dest, res.Msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (m messaging) positiveResponsePost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	m.log.Info().Msgf("got the following message: %s", buf)

	res := types.SendPositiveResponseArgument{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal unicast argument: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	err = m.node.AcceptFriendRequest(res.Dest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (m messaging) negativeResponsePost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	m.log.Info().Msgf("got the following message: %s", buf)

	res := types.SendNegativeResponseArgument{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal unicast argument: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	err = m.node.DeclineFriendRequest(res.Dest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}



//=================================Posts for friends=====================

func (m messaging) AddPersonalPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	m.log.Info().Msgf("got the following message: %s", buf)

	res := types.SetPersonalPost{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal addPeerArgument: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	m.log.Info().Msgf("got the following message: %v", res)

	m.node.SetPersonalPost(res.Message)
}

func (m messaging) PersonalPostGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	m.log.Info().Msgf("got the following message: %s", buf)

	res := types.AskPersonalPost{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal addPeerArgument: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	m.node.GetPersonalPost(res.Dest,res.Msg)
}

func (m messaging) AddPersonalPostHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			m.AddPersonalPost(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			return
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
			return
		}
	}
}

func (m messaging) GetPersonalPostHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			m.PersonalPostGet(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			return
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
			return
		}
	}
}