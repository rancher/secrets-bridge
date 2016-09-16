# Setup Guide
----

### Architecture

![Architecture Image](https://cdn.rawgit.com/rancher/secrets-bridge/master/docs/secrets_bridge_arch.svg)


### Configure Vault

Setting up a secure HA Vault installation is outside the scope of this document. We will assume that you have Vault running and have access to setup tokens and policies.


#### Example Config

#### Example:

This example will utilize:

* 1 grantor role
* 2 applications

For it to work, the Vault server must be started in DEV mode.

```
Note: these instructions use the convention of scoping roles and policies to a Rancher Environment.
```  

##### Step 1: Define polices

###### grantor-default.hcl

```
# grantor-default should not have access to sys
path "sys/*" {
  capabilities = ["deny"]
}

# Scope to Default environment
path "secret/secrets-bridge/Default/*" {
  capabilities = ["read", "list"]
}

path "auth/token/create/grantor-default" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

# Limit scope of grantor-default by denying access to all secrets
path "secret/*" {
  capabilities = ["deny"]
}
```
`Note: default is lower-cased within name grantor-default b/c the vault write command converts the name to lowercase. And the name within the auth/token/create/ must be consistent for path searches.`

This policy gives the `grantor-default` role the ability to grant tokens.

\*\*\* In general, tokens given to apps should not be able to create new tokens, though there is nothing stopping you from doing so

###### app1.hcl

```
path "sys/*" {
  capabilities = ["deny"]
}

path "secret/Default/Stack1/app1" {
  capabilities = ["read", "list"]
}

```

###### app2.hcl

```
path "sys/*" {
  capabilities = ["deny"]
}

path "secret/Default/Stack2/app2" {
  capabilities = ["read", "list"]
}

```
For demonstration of the Rancher use case using (i.e. environments, stacks, services and containers), the applications (i.e. services) are placed within their own stacks.

##### Step 2: Write policies to Vault

```
vault policy-write grantor-default ./grantor-default.hcl
vault policy-write app1 ./app1
vault policy-write app2 ./app2
```

##### Step 3: Set script variables

Obtain this information from your Vault startup log.

```
export PORT=8200
export VAULT_ADDR=http://xxx.xxx.xxx.xxx:$PORT
export ROOT_TOKEN=62c08fb4-e635-6a2d-f315-002e374e2ff1
export RANCHER_ENVIRONMENT_API_URL=http://xxx.xxx.xxx.xxy:XXXX/v1/projects/YYY
```
Set `RANCHER_ENVIRONMENT_API_URL` to the URL of API key for the Rancher Environment being used. For example, `RANCHER_ENVIRONMENT_API_URL=http://192.168.101.128:8080/v1/projects/1a5`

##### Step 4: Create grantor-default role

```
curl -s -X POST -H "X-Vault-Token: ${ROOT_TOKEN}" -d '{"allowed_policies": "default,grantor-default,app1,app2"}' http://vault/v1/auth/token/roles/grantor-default
```

##### Step 5: Assign policies to applications

```
vault write secret/secrets-bridge/Default/Stack1/app1 policies=default,app1
vault write secret/secrets-bridge/Default/Stack2/app2 policies=default,app2
```

##### Step 6: Configure Vault for Secrets-Bridge startup

Start by creating a permanent token for the grantor-default role. This token will be used by the secrets-bridge to interact with Vault and create temp tokens for applications.

```
PERM_TOKEN=$(curl -s -X POST -H "X-Vault-Token: $ROOT_TOKEN" ${VAULT_ADDR}/v1/auth/token/create/grantor-default -d '{"policies": ["default", "grantor-default", "app1", "app2"], "ttl": "72h", "meta": {"configPath": "secret/secrets-bridge/Default"}}' | jq -r '.auth.client_token')
```

**NOTE:** _If this key expires all tokens issued by key will also expire. It is recommended that this key is stored and retrievable via an administrative user for upgrades or service restarts of the secrets bridge server. It will always need to be retrieved via Cubbyhole._


Then create a temporary token with a TTL of 15m and a max usage of 2 attempts (1st used to place permanent token within cubbyhole; 2nd usage is when secrets-bridge starts up and contacts Vault to get permanent token)

```
TEMP_TOKEN=$(curl -s -H "X-Vault-Token: $ROOT_TOKEN" ${VAULT_ADDR}/v1/auth/token/create -d '{"policies": ["default"], "ttl": "15m", "num_uses": 2}' | jq -r '.auth.client_token')
```

Finally, place the permanent token within a cubbyhole using the temporary token. The secrets bridge will use the temporary token (2nd and final usage) to retrieve the permanent token from the cubbyhole when it starts up.

```
curl -X POST -H "X-Vault-Token: ${TEMP_TOKEN}" ${VAULT_ADDR}/v1/cubbyhole/Default -d "{\"permKey\": \"${PERM_TOKEN}\"}"
echo "${TEMP_TOKEN}"
```

### Configure Secrets Bridge Server

It is recommended that your run a secrets bridge server per Rancher environment. This limits the scope of a token expiration lapse and having a single token with broad access to Vault.

#### Command line
##### Step 1: Start Server

```
secrets-bridge server --vault-url $VAULT_ADDR --rancher-url $RANCHER_ENVIRONMENT_API_URL --vault-cubbypath cubbyhole/Default --vault-token $TEMP_TOKEN
```

#### Cattle

1. Deploy from secrets-bridge-server catalog entry.
	* you will need:
		* Vault URL
		* Vault Cubbyhole

### Configure Secrets Bridge Agents

 The agents are what listen on the host for Docker create events.

#### Command line
##### Step 1: Start the Agent

```
secrets-bridge agent --bridge-url http://[IP Of Secrets Bridge Server]:8181
```

#### Cattle

Launch from catalog secrets-bridge-agents.

#### Kubernetes

In K8s you have two choices.

A. Before turning K8s on in your environment deploy the secrets agent from the catalog entry. Then enable K8s on the manage environments page.

B. Deploy the agents as a system stack.

1. Go to system stacks page in UI.
2. Create new stack
3. Paste in compose file for secrets-bridge-agent. 
4. Launch stack




