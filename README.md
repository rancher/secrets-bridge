## Secrets-Bridge
---
###Status: Experimental POC (Read: Do NOT use for production) 

#### To Dos:
 * Create catalog entry
 * Make work with TLS production Vault setup.
		Only works with a Dev Vault configuration.
 * Add support for K8s and Swarm
 * Cattle needs signature verification call.


---
#### Purpose: 
  This is a first pass at a service that validates container identity from Rancher and then provides access to Vault. Rancher nor this service actually manage secrets in Vault, that is still left to the user and Vault. What this service will do is create Vault Tokens with policies allowed by the token provided to the service. The token obtained from this service is then used to communicate directly with Vault.
  
#### How it works:

In Vault, a user will create a Role for this service, scoping to an environment is probably a good idea. This Role should be assigned all of the Vault policies you need it to create tokens for. Vault only lets you create tokens for a subset of your own assigned tokens. 

To accomplish this, you need to create some default policies for the `secrets-bridge` along with a Vault CubbyHole. See Vaults documentation for more details on CubbyHoles, this service relies heavily on them.

A service would then be deployed into an operators/tools/non-application environment. This item will likely be launchable from the catalog.

Once the server side service is deployed, you would then deploy the agents into your application enviornment. These agents then listen for Docker events and send container start events to the service. 

The service then verifies with Rancher (see notes/todos below) the containers Identitiy. If the identity can not be verified, then nothing else happens. If the container is verified, the service checks for a policy key set on the config service tokens config path. If a policy is found, it then generates a temporary token to create a Cubbyhole and a permanent token with the applied policy. The permanent key is placed into the Cubbyhole with the temporary key, which will have a short TTL and 1 more use to get the permanent key.

The response to the agent contains, the Docker container ID (From Rancher), the Vault path of the Cubbyhole and the Temporary token. From this, the Agent does a Docker copy of the Cubbyhole information to: `/tmp/secrets.txt` on the target container.

A process inside the container can then read those credentials to get the permanent key and use Vault as it normally would.

#####Example:
If you have the Rancher Default enviroment, you might consider creating a role: `grantor-Default`. 

You should also have a `grantor` policy. This will give the token permission to grant tokens for the role. 

*NOTE: In general, tokens given to apps should not be able to create new tokens, though there is nothing stopping you from doing so.*

And in that environment you are going to have three stacks that are unique. `app1, app2, app3`

Create a unique Vault policy per app, and when you create the `grantor-Default` role the request should include all of the roles: `"default,grantor,app1,app2,app3"`

You should then create a permanent token with a reasonably long TTL with the grantor-Default role.

You should create a regular default policy with 2 uses and a TTL about as long as it would take an administrator to launch the secrets-bridge service.

Assume that we have grantor-Default and apps test1 and test2.

  1. Start Vault server in Dev mode
  1. Create roles

  ```
  vault policy-write grantor-Default ./policies/grantor-Default
  vault policy-write test1 ./policies/test1
  vault policy-write test2 ./policies/test2
  ```
  
  1. Create Roles

  ```
  curl -s -X POST -H "X-Vault-Token: ${VAULT_TOKEN}" -d '{"allowed_policies": "default,grantor,test1,test2"}' http://vault/v1/auth/token/roles/grantor-Default
  ```
  
  1. Assign policies to apps

  ```
  vault write secrets/secrets-bridge/Default/Default/test1 policies=test1,default
  vault write secrets/secrets-bridge/Default/Default/test2 policies=test2,default
  ```
  
  1. Create Grantor Token

  ```
  curl -s -H "X-Vault-Token: $ROOT_TOKEN" ${VAULT_URL}/v1/auth/token/create -d '{"policies": ["default"], "ttl": "15m", "num_uses": 2}'|jq -r '.auth.client_token')
PERM_TOKEN=$(curl -s -X POST -H "X-Vault-Token: $ROOT_TOKEN" ${VAULT_URL}/v1/auth/token/create/grantor-Default -d '{"policies": ["default", "grantor", "test1", "test2"], "ttl": "72h", "meta": {"configPath": "secret/secrets-bridge/Default"}}' | jq -r '.auth.client_token')
curl -X POST -H "X-Vault-Token: ${TEMP_TOKEN}" ${VAULT_URL}/v1/cubbyhole/Default -d "{\"permKey\": \"${PERM_TOKEN}\"}"
echo "${TEMP_TOKEN}"
```
  
  Above we are configuring the App to look in secret/secrets-bridge/Default for its configuration by assigning the configPath in the meta for the token. This can be whatever you want it to be, but this is where it will search for policies. 
  The Temp Token is what is needed to launch the secrets-bridge (server side) of the app. Note, it is only valid for 15m based on the above config. Once it is used, it will no longer be valid.
  
  1. Start secret service bridge

  ```
  secrets-bridge server --vault-url http://192.168.99.100:9000 --rancher-url http://192.168.99.100/v1 --vault-cubbypath cubbyhole/Default --vault-token 62c08fb4-e635-6a2d-f315-002e374e2ff1
  ```
  2. Start agent services

  ```
  secrets-bridge agent --bridge-url http://192.168.42.157:8181
  ```
  
 

