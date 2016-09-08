## Secrets-Bridge
---
### Status: Beta

---
#### Purpose:
The Secrets Bridge service is a standardized way of integrating Rancher and Vault such that Docker containers at startup are securely connected with their secrets within Vault. The Secrets Bridge service is composed of a server and agents. At container startup, the service first validates the container's identity with Rancher, and then provides the container with access to Vault. Neither Rancher nor the service actually manages any secrets within Vault; that is still left to the user and Vault. What this service will do is create Vault Tokens which are assigned a subset of policies allowed by the initial grantor-default token provided to the Secrets Bridge server at startup. The app token obtained through this service is then used by the container to communicate directly with Vault. This allows a user to define a custom process in their containers that can inject the secrets it reads from Vault into the app that ultimately uses them, using whatever custom input methods required by the user's app.

#### How it works:

In Vault, a user will create a Role for this service; scoping to an environment is probably a good idea. This Role should be assigned all of the Vault policies you need it to create tokens for. Vault only lets you create tokens for a subset of your own assigned tokens.

To accomplish this, you need to create some default policies for the `secrets-bridge` along with a Vault Cubbyhole. See Vaults documentation for more details on Cubbyholes, as this service relies heavily on them.

A service would then be deployed into an operators/tools/non-application environment. This item will likely be launchable from the catalog.

Once the server side service is deployed, you would then deploy the agents into your application environment. These agents then listen for Docker container start events. If your container has the secrets.bridge.enabled and nameKey labels correctly set then the agent will send that container's start event to the service.

The service then verifies with Rancher (see notes/todos below) the container's Identity. If the identity can not be verified, then nothing else happens. If the container is verified, the service checks for a policy key set on the config service tokens config path. If a policy is found, the service then generates a temporary token to create a Cubbyhole and a permanent token with the applied policy. The permanent key is placed into the Cubbyhole with the temporary key, which will have a short TTL and 1 more use to get the permanent key.

The response to the agent contains: the Docker container ID (from Rancher), the Vault path of the Cubbyhole, and the Temporary token. From this, the Agent performs a Docker copy of the Cubbyhole information to: `/tmp/secrets.txt` inside the target container.

A process inside the container can then read those credentials to get the permanent key and use Vault as it normally would.

##### Getting Started

[Setup Guide](https://github.com/rancher/secrets-bridge/blob/master/docs/setup.md)

[Using Secrets Bridge](https://github.com/rancher/secrets-bridge/blob/master/docs/applications.md)

##### Notes

* Assumes reasonably high degree of trust in the environment.
* Vault temporary tokens are the only things transmitted via secrets-bridge. Temporary Vault tokens are also stored on disk. These will have a short TTL and a single use remaining. See Vault Cubbyhole documentation for a more detailed explanation.


#### Debugging:

To enable debug output the '-d' flag has to be placed before any of the other command line arguments.  For example:

```
secrets-bridge -d agent --bridge-url http://[IP Of Secrets Bridge Server]:8181
```