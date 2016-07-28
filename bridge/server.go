package bridge

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/secrets-bridge/types"
	"github.com/rancher/secrets-bridge/vault"
	"github.com/rancher/secrets-bridge/verifier"
	"github.com/urfave/cli"
)

const default_verifier string = "rancher"

var actors *serverActors

type serverActors struct {
	verifier     verifier.Verifier
	secretStore  vault.SecureStore
	authVerifier verifier.AuthVerifier
}

type SecretResponse struct {
	ExternalID string `json:"externalId"`
	TempToken  string `json:"tempToken"`
	CubbyPath  string `json:"cubbyPath"`
}

func StartServer(c *cli.Context) {
	var err error
	actors, err = initActors(c)
	if err != nil {
		logrus.Fatalf("Could not initialize: %s", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/v1/message", tokenVerificationHandler).Methods("POST")

	logrus.Infof("Listening on port: 8181")
	http.ListenAndServe(":8181", r)
}

func initActors(c *cli.Context) (*serverActors, error) {
	verifierConfig := verifier.NewConfig(
		c.String("rancher-url"),
		c.String("rancher-access"),
		c.String("rancher-secret"))

	rVerify, err := verifier.NewVerifier(default_verifier, verifierConfig)
	if err != nil {
		logrus.Fatalf("Can not get verifier client")
		return nil, err
	}

	aVerify, err := verifier.NewAuthVerifier(default_verifier, verifierConfig)
	if err != nil {
		logrus.Fatalf("Can not get verifier client")
		return nil, err
	}

	secretStoreConfig := map[string]interface{}{
		"vault-token":     c.String("vault-token"),
		"vault-url":       c.String("vault-url"),
		"vault-cacert":    c.String("vault-cacert"),
		"vault-cubbypath": c.String("vault-cubbypath"),
	}

	sStore, err := vault.NewSecureStore(secretStoreConfig)
	if err != nil {
		logrus.Fatalf("Can not get secure store client: %s", err)
		return nil, err
	}

	return &serverActors{
		verifier:     rVerify,
		secretStore:  sStore,
		authVerifier: aVerify,
	}, nil

}

func tokenVerificationHandler(w http.ResponseWriter, r *http.Request) {
	header, _ := base64.StdEncoding.DecodeString(r.Header.Get("X-Agent-Signature"))
	verifiedAuth, err := actors.authVerifier.VerifyAuth(string(header))

	switch {
	case verifiedAuth && err == nil:
		messageHandler(w, r)
		return
	case err != nil:
		logrus.Errorf("Error Veifiying header: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	default:
		jsonUnauthorizedResponse(w)
		return
	}

	//just in case...
	jsonUnauthorizedResponse(w)
	return
}

func messageHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	t := &types.Message{}
	err := decoder.Decode(&t)
	if err != nil {
		logrus.Errorf("Bad..%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if t.Action == "start" && t.UUID != "" {
		logrus.Debugf("Received start event for container UUID: %s", t.UUID)
		if err := ContainerStart(w, t); err != nil {
			logrus.Errorf("Unverified: %s", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		return
	}

	w.WriteHeader(http.StatusNotImplemented)
	return
}

func jsonUnauthorizedResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
	return
}

func jsonSuccessResponse(body *SecretResponse, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(body)
	return
}

func ContainerStart(w http.ResponseWriter, msg *types.Message) error {
	var tempKey string

	verifiedObj, err := actors.verifier.Verify(msg)
	if err != nil {
		return err
	}

	if verifiedObj.Verified() {
		logrus.Debugf("Verified")
		tempKey, err = actors.secretStore.CreateSecretKey(verifiedObj)
		if err != nil {
			return err
		}
	}

	logrus.Debugf("VerifiedObj: %#v", verifiedObj)
	logrus.Debugf("VerifiedObj Path: %s", verifiedObj.Path())
	logrus.Debugf("VerifiedObj ID: %s", verifiedObj.ID())
	logrus.Debugf("TempKey ID: %s", tempKey)

	// ToDo: get a verified container object
	// This is not very generic...
	jsonSuccessResponse(&SecretResponse{
		TempToken:  tempKey,
		ExternalID: verifiedObj.ID(),
		CubbyPath:  actors.secretStore.GetSecretStoreURL() + "/cubbyhole/" + verifiedObj.Path(),
	}, w)

	return nil
}
