package bridge

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

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

type Error interface {
	error
	Status() int
}

type StatusError struct {
	Code int
	Err  error
}

func (se *StatusError) Error() string {
	return se.Err.Error()
}

func (se *StatusError) Status() int {
	return se.Code
}

func HTTPHandlerWrapper(t func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logrus.Debugf("Processing Request")
		defer logrus.Debugf("Finished Processing Request")

		header, _ := base64.StdEncoding.DecodeString(r.Header.Get("X-Agent-Signature"))

		verifiedAuth, err := actors.authVerifier.VerifyAuth(string(header))
		if err != nil || !verifiedAuth {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}

		if err := t(w, r); err != nil {
			switch e := err.(type) {
			case Error:
				http.Error(w, e.Error(), e.Status())
			default:
				http.Error(w, http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError)
			}
		}
	}
}

func StartServer(c *cli.Context) {
	var err error
	actors, err = initActors(c)
	if err != nil {
		logrus.Fatalf("Could not initialize: %s", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/v1/message", HTTPHandlerWrapper(messageHandler)).Methods("POST")

	logrus.Infof("Listening on port: 8181")
	s := &http.Server{
		Addr:         ":8181",
		Handler:      r,
		ReadTimeout:  45 * time.Second,
		WriteTimeout: 45 * time.Second,
	}
	s.ListenAndServe()
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

func messageHandler(w http.ResponseWriter, r *http.Request) error {
	var response *SecretResponse

	decoder := json.NewDecoder(r.Body)

	t := &types.Message{}
	err := decoder.Decode(&t)
	if err != nil {
		logrus.Errorf("BadReqeust..%s", err)
		return &StatusError{http.StatusBadRequest, err}
	}

	logrus.Debugf("MSG Decoded: %#v", t)
	if t.Action == "start" && t.UUID != "" {
		logrus.Debugf("Received start event for container UUID: %s", t.UUID)
		if response, err = ContainerStart(w, t); err != nil {
			logrus.Errorf("Unverified: %s", err)
			return &StatusError{http.StatusNotFound, err}
		}
		jsonSuccessResponse(response, w)

		logrus.Debugf("Responded successful")

		return nil
	}

	return &StatusError{http.StatusNotImplemented, nil}
}

func jsonSuccessResponse(body *SecretResponse, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(body)
	return
}

func ContainerStart(w http.ResponseWriter, msg *types.Message) (*SecretResponse, error) {
	var tempKey string

	verifiedObj, err := actors.verifier.Verify(msg)
	if err != nil {
		return &SecretResponse{}, err
	}

	if verifiedObj.Verified() {
		logrus.Debugf("Verified")
		tempKey, err = actors.secretStore.CreateSecretKey(verifiedObj)
		if err != nil {
			return &SecretResponse{}, err
		}
	}

	logrus.Debugf("VerifiedObj: %#v", verifiedObj)
	logrus.Debugf("VerifiedObj Path: %s", verifiedObj.Path())
	logrus.Debugf("VerifiedObj ID: %s", verifiedObj.ID())
	logrus.Debugf("TempKey ID: %s", tempKey)

	// ToDo: get a verified container object
	// This is not very generic...
	return &SecretResponse{
		TempToken:  tempKey,
		ExternalID: verifiedObj.ID(),
		CubbyPath:  actors.secretStore.GetSecretStoreURL() + "/cubbyhole/" + verifiedObj.Path(),
	}, nil
}
