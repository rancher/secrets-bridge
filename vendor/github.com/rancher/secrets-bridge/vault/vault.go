package vault

import (
	"errors"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/hashicorp/vault/api"
	"github.com/rancher/secrets-bridge/verifier"
)

type SecureStore interface {
	CreateSecretKey(verifier.VerifiedResponse) (string, error)
}

type VaultClient struct {
	VClient         *api.Client
	config          *api.Config
	envConfigPath   string // This is where to look for policy information.
	token           string
	tokenCreateRole string // The tokens can only create on this path...
}

func NewSecureStore(opts map[string]interface{}) (SecureStore, error) {
	return NewVaultSecureStore(opts)
}

func NewVaultSecureStore(opts map[string]interface{}) (*VaultClient, error) {
	config := api.DefaultConfig()

	if url, ok := opts["vault-url"]; ok {
		config.Address = url.(string)
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	permKey, err := unpackPermanentKey(client, opts)
	if err != nil {
		return nil, err
	}
	client.SetToken(permKey)

	tokenSecret, err := selfTokenSecret(client)
	if err != nil {
		return nil, err
	}

	configPath, err := inspectSelfTokenForConfigPath(tokenSecret)
	if err != nil {
		return nil, err
	}

	role := inspectSelfTokenForRole(tokenSecret)

	vaultClient := &VaultClient{
		VClient:         client,
		config:          config,
		envConfigPath:   configPath,
		token:           permKey,
		tokenCreateRole: role,
	}

	// handle refreshing the issuing token
	vaultClient.manageIssuingTokenRefresh()

	return vaultClient, nil

}

func (vc *VaultClient) manageIssuingTokenRefresh() {
	go func() {
		secret, err := selfTokenSecret(vc.VClient)
		if err != nil {
			logrus.Fatalf("Can not get Token secret: %s", err)
		}

		if secret.Data != nil {
			renewIncrement, ok := secret.Data["creation_ttl"]
			if !ok {
				logrus.Fatal("No creation_ttl set, likely can not renew")
			}

			renewalChannel := make(chan int)

			remainingTime, err := getSecretTTL(secret)
			if err != nil {
				logrus.Fatal("Issuing token has no TTL, has it expired")
			}

			go scheduleTimer(int(remainingTime), renewalChannel)

			for {
				select {
				case <-renewalChannel:
					logrus.Infof("Processing issuing token renewal")
					renewedSecret, err := vc.VClient.Auth().Token().RenewSelf(int(renewIncrement.(float64)))
					if err != nil {
						logrus.Errorf("Could not renew token: %s", err)
					}

					remainingTime, err = getSecretTTL(renewedSecret)
					if err != nil {
						logrus.Fatal("Issuing token has no TTL, has it expired")
					}

					go scheduleTimer(int(remainingTime), renewalChannel)
				}
			}
		}
		logrus.Fatalf("Issuing token didn't return TTL")
	}()
}

func getSecretTTL(secret *api.Secret) (float64, error) {
	remainingTime, ok := secret.Data["ttl"]
	if !ok {
		logrus.Fatal("Issuing token has no TTL Value.")
	}
	return remainingTime.(float64), nil
}

func calculateDuration(remainingTime int) int {
	if remainingTime > 300 {
		return remainingTime - 300
	}
	// this is hacky, but if its within 5 minutes... renew now
	return 1
}

func scheduleTimer(duration int, notify chan int) {
	time.Sleep(time.Duration(duration) * time.Second)
	notify <- 1
}

// We create cubbyholes in order to pass credentials
func (vClient *VaultClient) CreateSecretKey(verified verifier.VerifiedResponse) (string, error) {
	if !verified.Verified() {
		return "", errors.New("Secret creation aborted for unverified object")
	}
	cubbyConfig := &CubbyHoleConfig{
		TempTTL:      "300s",
		TempUseLimit: 2,
		PermTTL:      "1h",
		PermUseLimit: 0,
		Path:         verified.Path(),
	}

	cubbyHoleKeys, err := NewCubbyhole(vClient, cubbyConfig)
	if err != nil {
		return "", err
	}

	return cubbyHoleKeys.TempToken().Auth.ClientToken, nil
}

func (vClient *VaultClient) GetAppPolicies(appPath string) ([]string, error) {
	// OK, lets get the most specific...
	policies := []string{}

	splitPath := strings.Split(appPath, "/")
	for i := strings.Count(appPath, "/") + 1; i >= 0; i-- {
		fullPath := vClient.envConfigPath + "/" + strings.Join(splitPath[:i], "/")
		logrus.Infof("Trying path: %s", fullPath)
		secret, err := vClient.VClient.Logical().Read(fullPath)
		if err != nil && i != 0 {
			return policies, err
		}
		logrus.Infof("secret: %#v", secret)
		if secret != nil {
			if policies, ok := secret.Data["policies"]; ok {
				return strings.Split(policies.(string), ","), nil
			}
		}
	}

	return policies, nil
}

func selfTokenSecret(c *api.Client) (*api.Secret, error) {
	secret, err := c.Auth().Token().LookupSelf()
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func inspectSelfTokenForRole(secret *api.Secret) string {
	if secret.Data != nil {
		if role, ok := secret.Data["role"]; ok {
			return role.(string)
		}
	}
	return ""
}
func inspectSelfTokenForConfigPath(secret *api.Secret) (string, error) {
	if secret.Data != nil {
		if meta, ok := secret.Data["meta"].(map[string]interface{})["configPath"]; ok {
			return meta.(string), nil
		}
	}

	return "", errors.New("No configPath key found on token metadata")
}

func unpackPermanentKey(c *api.Client, opts map[string]interface{}) (string, error) {
	temp_token, ok := opts["vault-token"]
	if !ok {
		return "", errors.New("Vault token not set")
	}
	c.SetToken(temp_token.(string))

	keyPath, ok := opts["vault-cubbypath"]
	if !ok {
		return "", errors.New("Vault Cubby Path must be set.")
	}

	secretResp, err := c.Logical().Read(keyPath.(string))
	if err != nil {
		return "", err
	}

	// Started with a temp token, so we need to get the actual token
	permKey, ok := secretResp.Data["permKey"]
	if !ok {
		return "", errors.New("The key 'permKey' was not found at path: " + keyPath.(string))
	}

	return permKey.(string), nil
}
