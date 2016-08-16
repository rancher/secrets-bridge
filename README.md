## Secrets-Bridge
---
### Status: Experimental POC (Read: Do NOT use for production)

#### To Dos:
 * Create catalog entry.
 * Make work with TLS production Vault setup (currently only works with a Dev Vault configuration).
 * Add support for K8s and Swarm.
 * Cattle needs signature verification call.


---
#### Purpose:
The Secrets Bridge service is a standardized way of integrating Rancher and Vault such that Docker containers at startup are securely connected with their secrets within Vault. The Secrets Bridge service is composed of a server and agents. At container startup, the service first validates the container's identity with Rancher, and then provides the container with access to Vault. Neither Rancher nor the service actually manages any secrets within Vault; that is still left to the user and Vault. What this service will do is create Vault Tokens which are assigned a subset of policies allowed by the initial grantor-default token provided to the Secrets Bridge server at startup. The app token obtained through this service is then used by the container to communicate directly with Vault. This allows a user to define a custom process in their containers that can inject the secrets it reads from Vault into the app that ultimately uses them, using whatever custom input methods required by the user's app.

#### How it works:

In Vault, a user will create a Role for this service; scoping to an environment is probably a good idea. This Role should be assigned all of the Vault policies you need it to create tokens for. Vault only lets you create tokens for a subset of your own assigned tokens.

To accomplish this, you need to create some default policies for the `secrets-bridge` along with a Vault Cubbyhole. See Vaults documentation for more details on Cubbyholes, as this service relies heavily on them.

A service would then be deployed into an operators/tools/non-application environment. This item will likely be launchable from the catalog.

Once the server side service is deployed, you would then deploy the agents into your application environment. These agents then listen for Docker events and send container start events to the service.

The service then verifies with Rancher (see notes/todos below) the container's Identity. If the identity can not be verified, then nothing else happens. If the container is verified, the service checks for a policy key set on the config service tokens config path. If a policy is found, the service then generates a temporary token to create a Cubbyhole and a permanent token with the applied policy. The permanent key is placed into the Cubbyhole with the temporary key, which will have a short TTL and 1 more use to get the permanent key.

The response to the agent contains: the Docker container ID (from Rancher), the Vault path of the Cubbyhole, and the Temporary token. From this, the Agent performs a Docker copy of the Cubbyhole information to: `/tmp/secrets.txt` inside the target container.

A process inside the container can then read those credentials to get the permanent key and use Vault as it normally would.

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
export VAULT_ADDR=http://xxx.xxx.xxx.xxx:$VAULT_PORT
export ROOT_TOKEN=62c08fb4-e635-6a2d-f315-002e374e2ff1
export RANCHER_ENVIRONMENT_API_URL=http://xxx.xxx.xxx.xxy:XXXX/v1/projects/YYY
```
Set RANCHER_ENVIRONMENT_API_URL to the URL of API key for the Rancher Environment being used. For example, RANCHER_ENVIRONMENT_API_URL=http://192.168.101.128:8080/v1/projects/1a5

##### Step 4: Create grantor-default role

```
curl -s -X POST -H "X-Vault-Token: ${VAULT_TOKEN}" -d '{"allowed_policies": "default,grantor-default,app1,app2"}' http://vault/v1/auth/token/roles/grantor-default
```

##### Step 5: Assign policies to applications

```
vault write secrets/secrets-bridge/Default/Stack1/app1 policies=default,app1
vault write secrets/secrets-bridge/Default/Stack2/app2 policies=default,app2
```

##### Step 6: Configure Vault for Secrets-Bridge startup

Start by creating a permanent token for the grantor-default role. This token will be used by the secrets-bridge to interact with Vault and create temp tokens for applications.

```
PERM_TOKEN=$(curl -s -X POST -H "X-Vault-Token: $ROOT_TOKEN" ${VAULT_URL}/v1/auth/token/create/grantor-default -d '{"policies": ["default", "grantor-default", "app1", "app2"], "ttl": "72h", "meta": {"configPath": "secret/secrets-bridge/Default"}}' | jq -r '.auth.client_token')
```

Then create a temporary token with a TTL of 15m and a max usage of 2 attempts (1st used to place permanent token within cubbyhole; 2nd usage is when secrets-bridge starts up and contacts Vault to get permanent token)

```
TEMP_TOKEN=$(curl -s -H "X-Vault-Token: $ROOT_TOKEN" ${VAULT_URL}/v1/auth/token/create -d '{"policies": ["default"], "ttl": "15m", "num_uses": 2}' | jq -r '.auth.client_token')
```

Finally, place the permanent token within a cubbyhole using the temporary token. The secrets bridge will use the temporary token (2nd and final usage) to retrieve the permanent token from the cubbyhole when it starts up.

```
curl -X POST -H "X-Vault-Token: ${TEMP_TOKEN}" ${VAULT_URL}/v1/cubbyhole/Default -d "{\"permKey\": \"${PERM_TOKEN}\"}"
echo "${TEMP_TOKEN}"
```

##### Step 7: Start Server

```
secrets-bridge server --vault-url $VAULT_ADDR --rancher-url $RANCHER_ENVIRONMENT_API_URL --vault-cubbypath cubbyhole/Default --vault-token $TEMP_TOKEN
```

##### Step 8: Start the Agent

```
secrets-bridge agent --bridge-url http://[IP Of Secrets Bridge Server]:8181
```
