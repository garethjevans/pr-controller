# pr-controller 

a pull request webhook handler for TAP supply chains.

This is a proof of concept and should not be depended on in any way!!

## Installation

To install `pr-controller` on a local k8s cluster, use: 

```shell
kubectl apply -f resources/server-it.yaml
```

This will create a `pr-controller` namespace and install the application there.

To run a post deployment tekton task to validate that the server is functioning.

```shell
kubectl create -f resources/test-taskrun.yaml
tkn taskrun logs -f
```

You should see logs that look something like:

```shell
taskrun.tekton.dev/pr-controller-kcfpz created
? Select taskrun: pr-controller-kcfpz started 1 second ago
[upload] Generating test content...
[upload] test content
[upload] Uploading file...
[upload]   % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
[upload]                                  Dload  Upload   Total   Spent    Left  Speed
100   235  100    36  100   199   6888  38078 --:--:-- --:--:-- --:--:-- 47000
[upload] {"ok":true,"path":"/files/test.txt"}

[download] Downloading file...
[download]   % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
[download]                                  Dload  Upload   Total   Spent    Left  Speed
100    13  100    13    0     0   5922      0 --:--:-- --:--:-- --:--:--  6500
[download] test content
```

## Build this in TAP

```shell
tanzu apps workload create pr-controller \
  --namespace dev \
  --git-branch main \
  --git-repo https://github.com/garethjevans/pr-controller \
  --label apps.tanzu.vmware.com/has-tests=true \
  --label app.kubernetes.io/part-of=pr-controller \
  --param-yaml testing_pipeline_matching_labels='{"apps.tanzu.vmware.com/pipeline":"golang-pipeline"}' \
  --type web \
  --yes
```
