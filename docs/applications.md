# Using Secrets Bridge
---

#### Cattle Environments
1. In Vault setup a policy for your application. Depending on the scope, the secrets bridge will look for a policy in this order:
	* `<configPath>/<environment_name>/<stack_name>/<service_name>/<container_name>`
	* `<configPath>/<environment_name>/<stack_name>/<service_name>`
	* `<configPath>/<environment_name>/<stack_name>`
	* `<configPath>/<environment_name>`

	If no policy is found in any of those paths in Vault then no keys will be generated for the container. The key must be policy=
	
2. When launching applications, the `secrets.bridge.enabled=true` label should be used.


#### Kubernetes Environments

1. In Vault setup a policy for the application. Depending on the scope you need, the secrets bridge will look for a policy in this order:
	* `<configPath>/<environment_name>/<k8s_namespace>/<label_based_path>`
	* `<configPath>/<environment_name>/<k8s_namespace>`
	* `<configPath>/<environment_name>`

 The label path is specified by the user when launching the containers via the `secrets.bridge.k8s.path` label.
 
2. When launching an application the following labels can be used:
	* secrets.bridge.enabled=true (required)
	* secrets.bridge.k8s.path=policy/path/in/vault (optional)


### Secrets

In both orchestration engines your secrets will be writen to /tmp/secrets.txt.