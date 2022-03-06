/* global Stimulus, Viz */
/* exported main */

// main is the entry point of the script, called when the DOM is loaded
//
// This script relies on the stimulus library to bind data between the HTML and
// the javascript. In short each HTML element with a special
// `data-XXX-target=yyy` attribute can be accessed in the corresponding `XXX`
// controller as the `this.yyyTarget` variable. See
// https://stimulus.hotwired.dev/ for a complete introduction.

const main = function () {
    const application = Stimulus.Application.start();

    application.register("flash", Flash);
    application.register("peerInfo", PeerInfo);
    application.register("messaging", Messaging);
    application.register("friendRequestMessaging", FriendRequestMessaging);
    application.register("unicast", Unicast);
    application.register("friendRequest", FriendRequest);
    application.register("encryptedMessage", EncryptedMessage);
    application.register("positiveResponse", PositiveResponse);
    application.register("negativeResponse", NegativeResponse);
    application.register("routing", Routing);
    application.register("packets", Packets);
    application.register("broadcast", Broadcast);
    application.register("catalog", Catalog);
    application.register("dataSharing", DataSharing);
    application.register("search", Search);
    application.register("naming", Naming);
    application.register("postFriends", PostFriends);
    application.register("postsFriend", PostsFriend);
    application.register("username", Username);
    application.register("avatar", Avatar);

    initCollapsible();
};

function initCollapsible() {
    // https://www.w3schools.com/howto/howto_js_collapsible.asp
    var coll = document.getElementsByClassName("collapsible");
    for (var i = 0; i < coll.length; i++) {
        coll[i].addEventListener("click", function () {
            var content = this.nextElementSibling;
            if (this.classList.contains("active")) {
                content.style.display = "none";
                this.classList.remove("active");
            } else {
                content.style.display = "block";
                this.classList.add("active");
            }
        });

        var content = coll[i].nextElementSibling;
        if (coll[i].classList.contains("active")) {
            content.style.display = "block";
        } else {
            content.style.display = "none";
        }
    }
}

// BaseElement is inherited by all the controllers. It provides common methods.
class BaseElement extends Stimulus.Controller {

    // checkInputs is a utility function to checks if form inputs are empty or
    // not. It takes care of displaying an appropriate message if a validation
    // fails. The caller should exit if this function returns false.
    checkInputs(...els) {
        for (let i = 0; i < els.length; i++) {
            const val = els[i].value;
            if (val == "" || val == undefined) {
                this.flash.printError(`form validation failed: "${els[i].name}" is empty or invalid`);
                return false;
            }
        }

        return true;
    }

    // fetch fetches the addr and checks for a 200 status in return.
    async fetch(addr, opts) {
        const resp = await fetch(addr, opts);
        if (resp.status != 200) {
            const text = await resp.text();
            throw `wrong status: ${resp.status} ${resp.statusText} - ${text}`;
        }

        return resp;
    }

    get flash() {
        const element = document.getElementById("flash");
        return this.application.getControllerForElementAndIdentifier(element, "flash");
    }

    get peerInfo() {
        const element = document.getElementById("peerInfo");
        return this.application.getControllerForElementAndIdentifier(element, "peerInfo");
    }
}

class Flash extends Stimulus.Controller {
    static get targets() {
        return ["wrapper"];
    }

    printError(e) {
        const flash = this.createFlash("error", e);
        this.wrapperTarget.appendChild(flash);
    }

    printSuccess(e) {
        const flash = this.createFlash("success", e);
        this.wrapperTarget.appendChild(flash);
    }

    createFlash(className, content) {
        content = content.replace(/\s+/g, ' ').trim();

        const newDiv = document.createElement("div");
        newDiv.classList.add("flash-msg", className);

        const closeBtn = document.createElement("span");

        closeBtn.innerHTML = "&times;";
        closeBtn.classList.add("closebtn");
        closeBtn.addEventListener("click", (e) => {
            e.target.parentElement.remove();
        });

        const progress = document.createElement("span");
        progress.classList.add("progress");

        newDiv.innerText = content;
        newDiv.appendChild(closeBtn);
        newDiv.appendChild(progress);

        setTimeout(() => {
            newDiv.remove();
        }, 4000);

        return newDiv;
    }
}

class PeerInfo extends Stimulus.Controller {

    static usernameHolder = null;
    static usernameEmpty = true;
    static avatarHolder = null;
    static avatarEmpty = true;

    static get targets() {
        return ["peerAddr", "socketAddr", "usernameValue", "avatarValue"];
    }

    async initialize() {
        const queryDict = this.getQueryArgs();

        const endpoint = "http://" + decodeURIComponent(queryDict["addr"]);
        this.endpoint = endpoint;

        this.peerAddrTarget.innerText = this.endpoint;
        console.log(this.endpoint);

        const addr = this.endpoint + "/socket/address";

        try {
            const resp = await fetch(addr);
            const text = await resp.text();

            this.socketAddrTarget.innerText = "tcp://" + text;
            this.socketAddr = text;
        } catch (e) {
            this.flash.printError("failed to fetch socket address: " + e);
        }

        // let's get the custom username for the user, if any
        const usernameEndpoint = this.endpoint + "/username/my"
        try {
            const resp = await fetch(usernameEndpoint);
            this.usernameValueTarget.innerText = await resp.text();
            if (resp.status !== 404) {
                PeerInfo.usernameEmpty = false;
            }
        } catch (e) {

        }

        PeerInfo.usernameHolder = this.usernameValueTarget;

        // We also need to find the avatar of the user, if any
        const avatarEndpoint = this.endpoint + "/avatar/my"
        try {
            const resp = await fetch(avatarEndpoint);
            if (resp.status === 404) {
                this.avatarValueTarget.innerText = await resp.text();
            } else {
                let imageData;
                resp.blob().then(blobResponse => {
                    let reader = new FileReader();
                    reader.readAsDataURL(blobResponse);
                    reader.onloadend = () => {
                        imageData = reader.result;
                        PeerInfo.usernameEmpty = false;
                        this.avatarValueTarget.innerHTML = `
                            <img src=${imageData} class="avatar" alt="avatar"/>
                        `
                    }
                });

            }
        } catch (e) {
        }

        PeerInfo.avatarHolder = this.avatarValueTarget;
    }

    getAPIURL(suffix) {
        return this.endpoint + suffix;
    }

    get flash() {
        const element = document.getElementById("flash");
        return this.application.getControllerForElementAndIdentifier(element, "flash");
    }

    // getQueryArgs returns a dictionary of key=value arguments from the URL
    getQueryArgs() {
        const queryDict = {};
        location.search.substr(1).split("&").forEach(function (item) {
            queryDict[item.split("=")[0]] = item.split("=")[1];
        });

        return queryDict;
    }

    static setUsernameValue(v) {
        PeerInfo.usernameHolder.innerText = v;
        PeerInfo.usernameEmpty = false;
    }

    static setAvatarValue(v) {
        PeerInfo.usernameHolder.innerHTML = v;
        PeerInfo.usernameEmpty = false;
    }
}

class Messaging extends BaseElement {
    static get targets() {
        return ["holder", "messages"];
    }

    addMsg(el) {
        console.log("unicast add msg")
        this.messagesTarget.append(el);
        this.holderTarget.scrollTop = this.holderTarget.scrollHeight;
    }
}

class FriendRequestMessaging extends BaseElement {
    static get targets() {
        return ["holder", "messages"];
    }

    addMsg(el) {
        console.log("friend request add msg")
        this.messagesTarget.append(el);
        this.holderTarget.scrollTop = this.holderTarget.scrollHeight;
    }
}


class Unicast extends BaseElement {
    static get targets() {
        return ["message", "destination"];
    }

    async send() {
        const addr = this.peerInfo.getAPIURL("/messaging/unicast");

        const ok = this.checkInputs(this.messageTarget, this.destinationTarget);
        if (!ok) {
            return;
        }

        const message = this.messageTarget.value;
        const destination = this.destinationTarget.value;

        const pkt = {
            "Dest": destination,
            "Msg": {
                "Type": "chat",
                "payload": {
                    "Message": message
                }
            }
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(pkt)
        };

        try {
            await this.fetch(addr, fetchArgs);

            const date = new Date();
            const el = document.createElement("div");

            el.classList.add("sent");
            el.innerHTML = `
                <div>
                    <p class="msg">${message}</p>
                    <p class="details">
                        to ${destination} 
                        at ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}
                    </p>
                </div>`;

            this.messagingController.addMsg(el);
        } catch (e) {
            this.flash.printError("failed to send message: " + e);
        }
    }

    get messagingController() {
        const element = document.getElementById("messaging");
        return this.application.getControllerForElementAndIdentifier(element, "messaging");
    }
}

class PostFriends extends BaseElement {
    static get targets() {
        return ["message", "holder", "messages", "address"];
    }

    async send() {
        const addr = this.peerInfo.getAPIURL("/messaging/postFriends");

        const ok = this.checkInputs(this.messageTarget);
        if (!ok) {
            return;
        }

        const message = this.messageTarget.value;
        const pkt = {
            "Message": message,
        }
        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(pkt)
        };

        try {
            await this.fetch(addr, fetchArgs);
            this.flash.printSuccess("chat message postFriends");
        } catch (e) {
            this.flash.printError("failed to send message: " + e);
        }

        const el = document.createElement("div");
        el.classList.add("sent");
        el.innerHTML = ` <div>
            <p class="msg">${message}</p> 
            </div>`;
        this.addMsg(el)
    }

    addMsg(el) {
        this.messagesTarget.append(el);
        this.holderTarget.scrollTop = this.holderTarget.scrollHeight;
    }
}

class PostsFriend extends BaseElement {
    static get targets() {
        return ["message", "holder", "messages"];
    }

    async send() {
        const highlightedItems = document.querySelectorAll("div.receivedPost");
        highlightedItems.forEach(function(userItem) {
            userItem.remove();
        });

        const addr = this.peerInfo.getAPIURL("/messaging/postsFriend");

        const ok = this.checkInputs(this.messageTarget);
        if (!ok) {
            return;
        }

        const message = this.messageTarget.value;
        const pkt = {
            "Dest": message,
            "Msg": {
                "Type": "AskProfilePost",
                "payload": {
                    "Address": message
                }
            }
        };
        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(pkt)
        };

        try {
            await this.fetch(addr, fetchArgs);
        } catch (e) {
            console.log("couldnt send msg")
            this.flash.printError("failed to send message: " + e);
        }
    }

    addMsg(el) {
        this.messagesTarget.append(el);
        this.holderTarget.scrollTop = this.holderTarget.scrollHeight;
    }
}

class FriendRequest extends BaseElement {

    static get targets() {
        return ["destination"];
    }

    async send() {
        const addr = this.peerInfo.getAPIURL("/messaging/friendRequest");

        const ok = this.checkInputs(this.destinationTarget);
        if (!ok) {
            return;
        }
        const destination = this.destinationTarget.value;
        console.log(destination)

        const pkt = {
            "Dest": destination,
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(pkt)
        };

        try {
            await this.fetch(addr, fetchArgs);

            const date = new Date();
            const el = document.createElement("div");

            el.classList.add("sent");
            el.innerHTML = `
                <div>
                    <p class="msg">You sent a friend request to ${destination}</p>
                    <p class="details">
                        to ${destination} 
                        at ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}
                    </p>
                </div>`;

            this.friendRequestMessagingController.addMsg(el);
        } catch (e) {
            this.flash.printError("failed to send message: " + e);
        }
    }

    get friendRequestMessagingController() {
        const element = document.getElementById("friendRequestMessaging");
        return this.application.getControllerForElementAndIdentifier(element, "friendRequestMessaging");
    }
}

class PositiveResponse extends BaseElement {

    static values = {
        destination: String
    }


    async send() {
        const addr = this.peerInfo.getAPIURL("/messaging/positiveResponse");

        const destination = this.destinationValue;
        console.log(destination)

        const pkt = {
            "Dest": destination,
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(pkt)
        };

        try {
            await this.fetch(addr, fetchArgs);

            const date = new Date();
            const el = document.createElement("div");

            el.classList.add("sent");
            el.innerHTML = `
                <div>
                    <p class="msg">You accepted ${destination}'s friend request</p>
                    <p class="details">
                        to ${destination} 
                        at ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}
                    </p>
                </div>`;

            this.friendRequestMessagingController.addMsg(el);
        } catch (e) {
            this.flash.printError("failed to send message: " + e);
        }
    }

    remove() {
        document.getElementById("friendRequestNotif").remove()
    }


    get friendRequestMessagingController() {
        const element = document.getElementById("friendRequestMessaging");
        return this.application.getControllerForElementAndIdentifier(element, "friendRequestMessaging");
    }
}

class NegativeResponse extends BaseElement {
    static values = {
        destination: String
    }

    async send() {
        const addr = this.peerInfo.getAPIURL("/messaging/negativeResponse");

        const destination = this.destinationValue;
        console.log(destination)

        const pkt = {
            "Dest": destination
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(pkt)
        };

        try {
            await this.fetch(addr, fetchArgs);

            const date = new Date();
            const el = document.createElement("div");

            el.classList.add("sent");
            el.innerHTML = `
                <div>
                <p class="msg">You declined ${destination}'s friend request</p>
                    <p class="details">
                        to ${destination} 
                        at ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}
                    </p>
                </div>`;

            this.friendRequestMessagingController.addMsg(el);
        } catch (e) {
            this.flash.printError("failed to send message: " + e);
        }
    }

    remove() {
        document.getElementById("friendRequestNotif").remove()
    }

    get friendRequestMessagingController() {
        const element = document.getElementById("friendRequestMessaging");
        return this.application.getControllerForElementAndIdentifier(element, "friendRequestMessaging");
    }
}



class EncryptedMessage extends BaseElement {

    static get targets() {
        return ["destination", "message"];
    }
    async send() {
        console.log("sending msg to friend")
        const addr = this.peerInfo.getAPIURL("/messaging/encryptedMessage");
        const ok = this.checkInputs(this.destinationTarget, this.messageTarget);
        if (!ok) {
            return;
        }

        const destination = this.destinationTarget.value;
        const message = this.messageTarget.value;
        const pkt = {
            "Dest": destination,
            "Msg": {
                "Type": "chat",
                "payload": {
                    "Message": message
                }
            }
        };
        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(pkt)
        };
        try {
            await this.fetch(addr, fetchArgs);
            const date = new Date();
            const el = document.createElement("div");
            el.classList.add("sent");
            el.innerHTML = `
                <div>
                    <p class="msg">${message}</p>
                    <p class="details">
                        to ${destination} 
                        at ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}
                    </p>
                </div>`;
            this.messagingController.addMsg(el);
        } catch (e) {
            console.log("couldnt send msg")
            this.flash.printError("failed to send message: " + e);
        }
    }

    get messagingController() {
        const element = document.getElementById("messaging");
        return this.application.getControllerForElementAndIdentifier(element, "messaging");
    }

}


class Routing extends BaseElement {
    static get targets() {
        return ["table", "graphviz", "peer", "origin", "relay"];
    }

    initialize() {
        this.update();
    }

    async update() {
        const addr = this.peerInfo.getAPIURL("/messaging/routing");

        try {
            const resp = await this.fetch(addr);
            const data = await resp.json();

            this.tableTarget.innerHTML = "";

            for (const [origin, relay] of Object.entries(data)) {
                const el = document.createElement("tr");

                el.innerHTML = `<td>${origin}</td><td>${relay}</td>`;
                this.tableTarget.appendChild(el);
            }

            this.flash.printSuccess("Routing table updated");

        } catch (e) {
            this.flash.printError("Failed to fetch routing: " + e);
        }

        const graphAddr = addr + "?graphviz=on";

        try {
            const resp = await this.fetch(graphAddr);
            const data = await resp.text();

            var viz = new Viz();

            const element = await viz.renderSVGElement(data);
            this.graphvizTarget.innerHTML = "";
            this.graphvizTarget.appendChild(element);
        } catch (e) {
            this.flash.printError("Failed to display routing: " + e);
        }
    }

    async addPeer() {
        const ok = this.checkInputs(this.peerTarget);
        if (!ok) {
            return;
        }

        const addr = this.peerInfo.getAPIURL("/messaging/peers");
        const peer = this.peerTarget.value;

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify([peer])
        };

        try {
            await this.fetch(addr, fetchArgs);
            this.flash.printSuccess("peer added");
            this.update();
        } catch (e) {
            this.flash.printError("failed to add peer: " + e);
        }
    }

    async setEntry() {
        const addr = this.peerInfo.getAPIURL("/messaging/routing");

        const ok = this.checkInputs(this.originTarget);
        if (!ok) {
            return;
        }

        const origin = this.originTarget.value;
        const relay = this.relayTarget.value;

        const message = {
            "Origin": origin,
            "RelayAddr": relay
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(message)
        };

        try {
            await this.fetch(addr, fetchArgs);

            if (relay == "") {
                this.flash.printSuccess("Entry deleted");
            } else {
                this.flash.printSuccess("Entry set");
            }

            this.update();

        } catch (e) {
            this.flash.printError("failed to set entry: " + e);
        }
    }
}

class Packets extends BaseElement {
    static get targets() {
        return ["follow", "holder", "scroll", "packets"];
    }

    initialize() {
        const addr = this.peerInfo.getAPIURL("/registry/pktnotify");
        const newPackets = new EventSource(addr);

        newPackets.onmessage = this.packetMessage.bind(this);
        newPackets.onerror = this.packetError.bind(this);

        this.holderTarget.addEventListener("scroll", this.packetsScroll.bind(this));
    }

    packetMessage(e) {
        const pkt = JSON.parse(e.data);
        console.log("type is " + pkt.Msg.Type)
        if (pkt.Msg.Type == "chat") {
            console.log("received chat msg")
            const date = new Date(pkt.Header.Timestamp / 1000000);

            const el = document.createElement("div");

            if (pkt.Header.Source == this.peerInfo.socketAddr) {
                el.classList.add("sent");
            } else {
                el.classList.add("received");
            }

            // note that this is not secure and prone to XSS attack.
            el.innerHTML = `<div><p class="msg">${pkt.Msg.Payload.Message}</p><p class="details">from ${pkt.Header.Source} at ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}</p></div>`;

            this.messagingController.addMsg(el);
        } else if (pkt.Msg.Type == "RespondProfilePostMessage") {
            for (const [id, post] of Object.entries(pkt.Msg.Payload.Messages)) {
                const el = document.createElement("div");
                const date = new Date(pkt.Header.Timestamp / 1000000);

                el.classList.add("receivedPost");

                el.innerHTML = ` <div id="profilePostReceived">
                 <p class="details">${post.Date}</p>
                <p class="msg">${post.Message}</p>
                </div>`;

                this.PostFriendsMessagingController.addMsg(el);
            }
        } else if (pkt.Msg.Type == "AlertNewPostMessage") {
            this.flash.printSuccess(pkt.Header.Source + " has posted a message");
        } else if (pkt.Msg.Type == "friendRequest") {
            console.log("received friend request")
            const el = document.createElement("div");
            const date = new Date(pkt.Header.Timestamp / 1000000);

            if (pkt.Header.Source == this.peerInfo.socketAddr) {
                el.classList.add("sent");
            } else {
                el.classList.add("received");
            }


            el.innerHTML = ` <div id="friendRequestNotif">
            <p class="msg">${"New friend request from " + pkt.Header.Source}</p>
            <div  data-controller="positiveResponse" data-positiveResponse-destination-value=${pkt.Header.Source}>
            <button id="acceptButton" data-action ="click->positiveResponse#send click->positiveResponse#remove">
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path d="M24 0l-6 22-8.129-7.239 7.802-8.234-10.458 7.227-7.215-1.754 24-12zm-15 16.668v7.332l3.258-4.431-3.258-2.901z"/></svg>
            Accept
            </button>
            </div>
            
            <div data-controller="negativeResponse" data-negativeResponse-destination-value=${pkt.Header.Source}>
            <button id="declineButton"  data-action ="click->negativeResponse#send click->negativeResponse#remove">
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path d="M24 0l-6 22-8.129-7.239 7.802-8.234-10.458 7.227-7.215-1.754 24-12zm-15 16.668v7.332l3.258-4.431-3.258-2.901z"/></svg>
            Decline
            </button>
            </div>
            <p class="details">from ${pkt.Header.Source} at ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}</p>
            </div>
            `; 


            this.friendRequestMessagingController.addMsg(el);

        } else if (pkt.Msg.Type == "positiveResponse") {

            const el = document.createElement("div");
            const date = new Date(pkt.Header.Timestamp / 1000000);

            if (pkt.Header.Source == this.peerInfo.socketAddr) {
                el.classList.add("sent");
            } else {
                el.classList.add("received");
            }


            el.innerHTML = ` <div>
            <p class="msg">${pkt.Header.Source + " has accepted your friend request"}</p>
            <p class="details">from ${pkt.Header.Source} at ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}</p>
            </div>`;

            this.friendRequestMessagingController.addMsg(el);

        } else if (pkt.Msg.Type == "negativeResponse") {

            const el = document.createElement("div");
            const date = new Date(pkt.Header.Timestamp / 1000000);

            if (pkt.Header.Source == this.peerInfo.socketAddr) {
                el.classList.add("sent");
            } else {
                el.classList.add("received");
            }

            el.innerHTML = ` <div>
            <p class="msg">${pkt.Header.Source + " has declined your friend request"}</p>
            <p class="details">from ${pkt.Header.Source} at ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}</p>
            </div>`;

            this.friendRequestMessagingController.addMsg(el);
        } 
        const el = document.createElement("div");
        el.innerHTML = `<pre>${JSON.stringify(pkt, null, 2)}</pre>`;

        this.packetsTarget.append(el);

        this.scrollTarget.style.width = this.packetsTarget.scrollWidth + "px";

        if (this.followTarget.checked) {
            this.packetsTarget.scrollLeft = this.packetsTarget.scrollWidth;
            this.holderTarget.scrollLeft = this.holderTarget.scrollWidth;
        }
    }

    packetError() {
        this.flash.printError("failed to listen pkt: stopped listening");
    }

    packetsScroll() {
        this.packetsTarget.scrollLeft = this.holderTarget.scrollLeft;
    }

    get messagingController() {
        return this.application.getControllerForElementAndIdentifier(document.getElementById("messaging"), "messaging");
    }

    get PostFriendsMessagingController() {
        return this.application.getControllerForElementAndIdentifier(document.getElementById("postsFriend"), "postsFriend");
    }
      
    get friendRequestMessagingController() {
        return this.application.getControllerForElementAndIdentifier(document.getElementById("friendRequestMessaging"), "friendRequestMessaging");
    }
}

class Broadcast extends BaseElement {
    static get targets() {
        return ["chatMessage", "privateMessage", "privateRecipients"];
    }

    async sendChat() {
        const addr = this.peerInfo.getAPIURL("/messaging/broadcast");

        const ok = this.checkInputs(this.chatMessageTarget);
        if (!ok) {
            return;
        }

        const message = this.chatMessageTarget.value;

        const msg = {
            "Type": "chat",
            "payload": {
                "Message": message
            }
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(msg)
        };

        try {
            await this.fetch(addr, fetchArgs);
            this.flash.printSuccess("chat message broadcasted");
        } catch (e) {
            this.flash.printError("failed to send message: " + e);
        }
    }


    async sendPrivate() {
        const addr = this.peerInfo.getAPIURL("/messaging/broadcast");

        const ok = this.checkInputs(this.privateMessageTarget, this.privateRecipientsTarget);
        if (!ok) {
            return;
        }

        const destination = this.privateRecipientsTarget.value;
        const message = this.privateMessageTarget.value;

        const recipients = {};
        destination.split(",").forEach(e => recipients[e.trim()] = {});

        const msg = {
            "Type": "private",
            "payload": {
                "Recipients": recipients,
                "Msg": {
                    "Type": "chat",
                    "payload": {
                        "Message": message
                    }
                }
            }
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(msg)
        };

        try {
            await this.fetch(addr, fetchArgs);
            this.flash.printSuccess("private message sent");
        } catch (e) {
            this.flash.printError("failed to send message: " + e);
        }
    }
}

class Catalog extends BaseElement {
    static get targets() {
        return ["content", "key", "value"];
    }

    initialize() {
        this.update();
    }

    async update() {
        const addr = this.peerInfo.getAPIURL("/datasharing/catalog");

        try {
            const resp = await this.fetch(addr);
            const data = await resp.json();

            this.contentTarget.innerHTML = "";

            // Expected format of data:
            //
            // {
            //     "chunk1": {
            //         "peerA": {}, "peerB": {},
            //     },
            //     "chunk2": {...},
            // }

            if (Object.keys(data).length === 0) {
                this.contentTarget.innerHTML = "<i>no elements</i>";
                // this.flash.printSuccess("Catalog updated, nothing found");
                return;
            }

            for (const [chunk, peersBag] of Object.entries(data)) {
                const entry = document.createElement("div");
                const chunkName = document.createElement("p");
                chunkName.innerHTML = chunk;

                entry.appendChild(chunkName);

                for (var peer in peersBag) {
                    const peerEl = document.createElement("p");
                    peerEl.innerHTML = peer;
                    entry.appendChild(peerEl);
                }

                this.contentTarget.appendChild(entry);
            }

            this.flash.printSuccess("Catalog updated");
        } catch (e) {
            this.flash.printError("Failed to fetch catalog: " + e);
        }
    }

    async add() {
        const addr = this.peerInfo.getAPIURL("/datasharing/catalog");

        const ok = this.checkInputs(this.keyTarget, this.valueTarget);
        if (!ok) {
            return;
        }

        const key = this.keyTarget.value;
        const value = this.valueTarget.value;

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify([key, value])
        };

        try {
            await this.fetch(addr, fetchArgs);
            this.update();
        } catch (e) {
            this.flash.printError("failed to add catalog entry: " + e);
        }
    }
}

class DataSharing extends BaseElement {
    static get targets() {
        return ["uploadResult", "fileUpload", "downloadMetahash"];
    }

    async upload() {
        const addr = this.peerInfo.getAPIURL("/datasharing/upload");

        const fileList = this.fileUploadTarget.files;

        if (fileList.length == 0) {
            this.flash.printError("No file found");
            return;
        }

        const file = fileList[0];

        const reader = new FileReader();
        reader.addEventListener('load', async (event) => {
            const result = event.target.result;

            const fetchArgs = {
                method: "POST",
                headers: {
                    "Content-Type": "multipart/form-data"
                },
                body: result
            };

            try {
                const resp = await this.fetch(addr, fetchArgs);
                const text = await resp.text();

                this.flash.printSuccess(`data uploaded, metahash: ${text}`);
                this.uploadResultTarget.innerHTML = `Metahash: ${text}`;
            } catch (e) {
                this.flash.printError("failed to upload data: " + e);
            }
        });

        reader.addEventListener('progress', (event) => {
            if (event.loaded && event.total) {
                const percent = (event.loaded / event.total) * 100;
                this.flash.printSuccess(`File upload progress: ${Math.round(percent)}`);
            }
        });
        reader.readAsArrayBuffer(file);
    }

    async download() {
        const ok = this.checkInputs(this.downloadMetahashTarget);
        if (!ok) {
            return;
        }

        const metahash = this.downloadMetahashTarget.value;

        const addr = this.peerInfo.getAPIURL("/datasharing/download?key=" + metahash);

        try {
            const resp = await this.fetch(addr);
            const blob = await resp.blob();

            this.triggerDownload(metahash, blob);
            this.flash.printSuccess("Data downloaded!");
        } catch (e) {
            this.flash.printError("Failed to download data: " + e);
        }
    }

    triggerDownload(metahash, blob) {
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement("a");

        a.style.display = "none";
        a.href = url;
        a.download = metahash;
        document.body.appendChild(a);

        a.click();
        window.URL.revokeObjectURL(url);
    }
}

class Search extends BaseElement {
    static get targets() {
        return ["searchAllResult", "searchAllPattern", "searchAllBudget", "searchAllTimeout",
            "searchFirstResult", "searchFirstPattern", "searchFirstInitialBudget",
            "searchFirstFactor", "searchFirstRetry", "searchFirstTimeout"];
    }

    async searchAll() {
        const addr = this.peerInfo.getAPIURL("/datasharing/searchAll");

        const ok = this.checkInputs(this.searchAllPatternTarget,
            this.searchAllBudgetTarget, this.searchAllTimeoutTarget);
        if (!ok) {
            return;
        }

        const pattern = this.searchAllPatternTarget.value;
        const budget = this.searchAllBudgetTarget.value;
        const timeout = this.searchAllTimeoutTarget.value;

        const pkt = {
            "Pattern": pattern,
            "Budget": parseInt(budget),
            "Timeout": timeout
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "multipart/form-data"
            },
            body: JSON.stringify(pkt)
        };

        try {
            this.searchAllResultTarget.innerHTML = `<i>searching all...</i>`;

            const resp = await this.fetch(addr, fetchArgs);
            const text = await resp.text();

            this.flash.printSuccess(`SearchAll done, result: ${text}`);
            this.searchAllResultTarget.innerHTML = `Names: ${text}`;
        } catch (e) {
            this.flash.printError("failed to searchAll: " + e);
        }
    }

    async searchFirst() {
        const addr = this.peerInfo.getAPIURL("/datasharing/searchFirst");

        const ok = this.checkInputs(this.searchFirstPatternTarget, this.searchFirstInitialBudgetTarget,
            this.searchFirstFactorTarget, this.searchFirstRetryTarget, this.searchFirstTimeoutTarget);
        if (!ok) {
            return;
        }

        const pattern = this.searchFirstPatternTarget.value;
        const initial = this.searchFirstInitialBudgetTarget.value;
        const factor = this.searchFirstFactorTarget.value;
        const retry = this.searchFirstRetryTarget.value;
        const timeout = this.searchFirstTimeoutTarget.value;

        const pkt = {
            "Pattern": pattern,
            "Initial": parseInt(initial),
            "Factor": parseInt(factor),
            "Retry": parseInt(retry),
            "Timeout": timeout
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "multipart/form-data"
            },
            body: JSON.stringify(pkt)
        };

        try {
            this.searchFirstResultTarget.innerHTML = `<i>searching first...</i>`;

            const resp = await this.fetch(addr, fetchArgs);
            const text = await resp.text();

            this.flash.printSuccess(`Search done, result: ${text}`);
            if (text == "") {
                this.searchFirstResultTarget.innerHTML = "<i>Nothing found</i>";
            } else {
                this.searchFirstResultTarget.innerHTML = `Names: ${text}`;
            }

            this.flash.printSuccess(`search first done, result: ${text}`);
        } catch (e) {
            this.flash.printError("failed to Search first: " + e);
        }
    }
}

class Naming extends BaseElement {
    static get targets() {
        return ["resolveResult", "resolveFilename", "tagFilename", "tagMetahash"];
    }

    async resolve() {
        this.checkInputs(this.resolveFilenameTarget);

        const filename = this.resolveFilenameTarget.value;

        const addr = this.peerInfo.getAPIURL("/datasharing/naming?name=" + filename);

        try {
            const resp = await this.fetch(addr);
            const text = await resp.text();

            if (text == "") {
                this.resolveResultTarget.innerHTML = "<i>nothing found</i>";
            } else {
                this.resolveResultTarget.innerHTML = text;
            }

            this.flash.printSuccess("filename resolved");
        } catch (e) {
            this.flash.printError("Failed to resolve filename: " + e);
        }
    }

    async tag() {
        const addr = this.peerInfo.getAPIURL("/datasharing/naming");

        const ok = this.checkInputs(this.tagFilenameTarget, this.tagMetahashTarget);
        if (!ok) {
            return;
        }

        const filename = this.tagFilenameTarget.value;
        const metahash = this.tagMetahashTarget.value;

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify([filename, metahash])
        };

        try {
            await this.fetch(addr, fetchArgs);
            this.flash.printSuccess("tagging done");
        } catch (e) {
            this.flash.printError("failed to tag filename: " + e);
        }
    }
}


class Username extends BaseElement {
    static get targets() {
        return [
            "setUsername",
            "setUsernameResult",
            "searchUserPattern",
            "searchUserResult",
        ];
    }

    async setup() {
        const addr = this.peerInfo.getAPIURL("/username/set");
        const ok = this.checkInputs(this.setUsernameTarget);
        if (!ok) {
            return;
        }
        const username = this.setUsernameTarget.value;
        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify([username])
        };

        try {
            await this.fetch(addr, fetchArgs);
            PeerInfo.setUsernameValue(username);
            this.flash.printSuccess("successfully setup username: " + username);
        } catch (e) {
            this.flash.printError("failed to setup username: " + e);
        }

    }


    async searchUsername() {
        const addr = this.peerInfo.getAPIURL("/username/search");
        const ok = this.checkInputs(this.searchUserPatternTarget);
        if (!ok) {
            return;
        }
        const pattern = this.searchUserPatternTarget.value;
        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify([pattern])
        };

        try {
            this.searchUserResultTarget.innerHTML = `<i>searching users...</i>`;

            const resp = await this.fetch(addr, fetchArgs);
            const text = await resp.json();

            this.flash.printSuccess(`Search user done.`);

            let content = '';
            // TODO
            for (let i in text) {
                content += `
                <table class="peer-info">
                    <tr>
                        <td>Username</td>
                        <td>${text[i].Username}</td>
                    </tr>
                    <tr>
                        <td>Address</td>
                        <td>${text[i].Address}</td>
                    </tr>
                    <tr>
                        <td>Avatar</td>
                        <td>
                            <button id="avatar-button-${i}">View avatar</button>
                        </td>
                        <td><div id="avatar-image-${i}" /></td>
                    </tr>
                    <tr>
                        <td>
                            <button id="addFriend-button-${i}">Add Friend</button>
                        </td>
                    </tr>
                </table>
                `
            }
            this.searchUserResultTarget.innerHTML = `${content}`;

            let peerInfo = this.peerInfo;
            let app = this.application;

            for (let i in text) {
                document.getElementById(`avatar-button-${i}`)
                    .addEventListener('click', (async function(){
                        const downloadAvatarEndpoint = peerInfo.getAPIURL("/avatar/download")
                        try {
                            const fetchArgs = {
                                method: "POST",
                                headers: {
                                    "Content-Type": "multipart/form-data"
                                },
                                body: JSON.stringify([text[i].Address])
                            };
                            const resp = await fetch(downloadAvatarEndpoint, fetchArgs);
                            if (resp.status === 200) {
                                resp.blob().then(blobResponse => {
                                    let reader = new FileReader();
                                    reader.readAsDataURL(blobResponse);
                                    reader.onloadend = () => {
                                        document.getElementById(`avatar-image-${i}`).innerHTML =
                                            `<img src=${reader.result} class="avatar" alt="avatar"/>`
                                    }
                                });

                            }
                        } catch (e) {
                        }
                    }));
                document.getElementById(`addFriend-button-${i}`)
                    .addEventListener('click', (async function(){
                        const friendRequestEndpoint = peerInfo.getAPIURL("/messaging/friendRequest");
                        try {
                            const pkt = {
                                "Dest": text[i].Address,
                            };

                            const fetchArgs = {
                                method: "POST",
                                headers: {
                                    "Content-Type": "application/json"
                                },
                                body: JSON.stringify(pkt)
                            };
                            await fetch(friendRequestEndpoint, fetchArgs);

                            const date = new Date();
                            const el = document.createElement("div");
                            el.classList.add("sent");
                            el.innerHTML = `
                                <div>
                                    <p class="msg">You sent a friend request to ${text[i].Address}</p>
                                    <p class="details">
                                        to ${text[i].Address} 
                                        at ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}
                                    </p>
                                </div>
                            `;
                            const element = document.getElementById("friendRequestMessaging")
                            const controller = app.getControllerForElementAndIdentifier(element, "friendRequestMessaging");
                            controller.addMsg(el);
                        } catch (e) {
                            this.flash.printError("failed to send message: " + e);
                        }
                    }));
            }
        } catch (e) {
            this.searchUserResultTarget.innerHTML = ``;
            this.flash.printError("failed to find results: " + e);
        }

    }
}


class Avatar extends BaseElement {
    static get targets() {
        return ["avatarUploadResult", "avatarUpload"];
    }

    async upload() {
        const addr = this.peerInfo.getAPIURL("/avatar/upload");

        const fileList = this.avatarUploadTarget.files;

        if (fileList.length !== 1) {
            this.flash.printError("You must choose one file");
            return;
        }

        const file = fileList[0];

        const reader = new FileReader();
        reader.addEventListener('load', async (event) => {
            const result = event.target.result;

            const fetchArgs = {
                method: "POST",
                headers: {
                    "Content-Type": "multipart/form-data"
                },
                body: result
            };

            try {
                const resp = await this.fetch(addr, fetchArgs);
                const text = await resp.text();

                this.avatarUploadResultTarget.innerHTML = `Metahash: ${text}`;
            } catch (e) {
                this.flash.printError("failed to upload data: " + e);
            }
        });

        reader.readAsArrayBuffer(file);
    }
}