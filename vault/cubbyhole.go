package vault

import (
	"errors"

	"github.com/Sirupsen/logrus"
	"github.com/hashicorp/vault/api"
)

type CubbyHoleConfig struct {
	TempTTL      string
	TempUseLimit int
	PermTTL      string
	PermUseLimit int
	PermPolicy   string
	Path         string
}

type CubbyHoleKeys struct {
	tempKey *api.Secret
}

func NewCubbyhole(client *VaultClient, cubbyConfig *CubbyHoleConfig) (*CubbyHoleKeys, error) {
	metadata := make(map[string]string)

	logrus.Infof("Getting Temp Token")
	tempToken, err := createVaultToken(client, &api.TokenCreateRequest{
		ID:              "",
		Policies:        []string{"default"},
		Metadata:        metadata,
		TTL:             cubbyConfig.TempTTL,
		NoParent:        false,
		NoDefaultPolicy: false,
		DisplayName:     "",
		NumUses:         cubbyConfig.TempUseLimit,
	})
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	policies, err := client.GetAppPolicies(cubbyConfig.Path)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Got policies: %s", policies)

	if len(policies) == 0 {
		return nil, errors.New("No policies to attach")
	}

	permToken, err := createVaultToken(client, &api.TokenCreateRequest{
		ID:              "",
		Policies:        policies,
		Metadata:        metadata,
		TTL:             cubbyConfig.PermTTL,
		NoParent:        false,
		NoDefaultPolicy: false,
		DisplayName:     "",
		NumUses:         cubbyConfig.PermUseLimit,
	})
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	if err := writePermanentKey(permToken, tempToken, "cubbyhole/"+cubbyConfig.Path, client); err != nil {
		return nil, err
	}

	return &CubbyHoleKeys{
		tempKey: tempToken,
	}, nil
}

func createVaultToken(client *VaultClient, tcr *api.TokenCreateRequest) (*api.Secret, error) {
	if client.tokenCreateRole != "" {
		return client.VClient.Auth().Token().CreateWithRole(tcr, client.tokenCreateRole)
	}
	logrus.Warn("You are probably running with Root keys...and thats probably not good")
	return client.VClient.Auth().Token().Create(tcr)
}

func (chk *CubbyHoleKeys) TempToken() *api.Secret {
	return chk.tempKey
}

func writePermanentKey(perm, temp *api.Secret, path string, client *VaultClient) error {
	client.VClient.SetToken(temp.Auth.ClientToken)
	defer client.VClient.SetToken(client.token)

	_, err := client.VClient.Logical().Write(path, map[string]interface{}{"permKey": perm.Auth.ClientToken})
	if err != nil {
		return err
	}

	return nil
}
