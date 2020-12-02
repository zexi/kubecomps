package auth

import (
	"encoding/json"
	"fmt"
	"net/http"

	"yunion.io/x/log"
)

type userInfo struct {
	Username string              `json:"username"`
	UID      string              `json:"uid"`
	Groups   []string            `json:"groups"`
	Extra    map[string][]string `json:"extra"`
}

type status struct {
	Authenticated bool     `json:"authenticated"`
	User          userInfo `json:"user"`
}

type WebhookHandler struct {
	Authenticator *KeystoneAuthenticator
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err := decoder.Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var apiVersion = data["apiVersion"].(string)
	var kind = data["kind"].(string)

	if apiVersion != "authentication.k8s.io/v1beta1" && apiVersion != "authorization.k8s.io/v1beta1" {
		http.Error(w, fmt.Sprintf("unknown apiVersion %q", apiVersion),
			http.StatusBadRequest)
		return
	}
	if kind == "TokenReview" {
		var token = data["spec"].(map[string]interface{})["token"].(string)
		h.authenticateToken(w, r, token, data)
		return
	}
	http.Error(w, fmt.Sprintf("unknown kind/apiVersion %q %q", kind, apiVersion),
		http.StatusBadRequest)
}

func (h *WebhookHandler) authenticateToken(w http.ResponseWriter, r *http.Request, token string, data map[string]interface{}) {
	log.Infof(">>>> authenticateToken data: %#v\n", data)
	user, authenticated, err := h.Authenticator.AuthenticateToken(token)
	log.Infof("<<<< authenticateToken: %v, user: %v, authed: %v, err: %v", token, user, authenticated, err)

	if !authenticated {
		var response status
		response.Authenticated = false
		data["status"] = response

		output, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(output)
		return
	}

	var info userInfo
	info.Username = user.GetName()
	info.UID = user.GetUID()
	info.Groups = user.GetGroups()
	info.Extra = user.GetExtra()

	var response status
	response.Authenticated = true
	response.User = info

	data["status"] = response

	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(output)
}
