# ClusterLoader2

在本教程中，我们将：
- 准备本地开发环境和cl2代码仓库。
- 使用[Kind]创建单节点集群。
- 实现一个简单的CL2测试并运行它。
- 在100节点集群运行负载测试。

## 1 Clone仓库

```bash
mkdir -p  $GOPATH/src/github.com/zhengbingxian/  && cd !$
git clone https://github.com/zhengbingxian/clusterloader2.git
cd clusterloader2
```

## 2 安装golang环境，例如使用GVM
按照[GVM install]上的说明进行操作.
Install golang with specific version (1.15.12 was tested in this tutorial):

```bash
gvm install go1.15.12
gvm use go1.15.12
```
Next, add perf-tests repository to GOPATH:

```bash
gvm linkthis k8s.io/perf-tests
```

## 3 使用kind创建集群
按照 [kind 安装][Kind install] 指南操作. 例如linux：

```shell
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.15.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind
```

创建 v1.21.1 集群（单节点既是master又是worker）:
```bash
kind create cluster --image=kindest/node:v1.21.1 --name=test-cluster --wait=5m
```

此命令还会生成存储在 `${HOME}/.kube/config` 的以test-cluster命名的访问凭证上下文。

检查是否可以连接到集群:
```bash
# kind不包含kubectl，安装kubectl见https://kubernetes.io/docs/tasks/tools/
kubectl get nodes
```

## 4 准备运行简单的测试
让我们准备第一个测试配置 (config.yaml).该测试将:

- 创建一个namespace
- 并创建一个deployment with 10 pods inside。
- 测量这些pod的启动延迟

创建 `config.yaml`文件来描述此次测试。首先我们从定义测试名开始：
```yaml
name: test
```
CL2 会自动创建命名空间，但是我们需要指定需要多少命名空间：
```yaml
namespace:
  number: 1
```
接下来，我们需要指定 TuningSets. TuningSet 描述了如何执行操作。在我们的例子中，我们只有1 deployment。所以只有1个操作要执行。

在本例中，tuningSet 并不真正影响状态之间的转换。

```yaml
tuningSets:
- name: Uniform1qps
  qpsLoad:
    qps: 1
```
测试定义由一系列steps组成。 A step 可以是Phases 或Measurements的集合。
A Phase 定义了集群应该达到的状态。A Measurement 允许测量或等待某些东西。你可以在这里找到可用的测量列表 [Measurements].

第一个step，是开始两个measurements。 我们想要开始测量pod启动延迟，同时也要等待所有pod处于running（这也是一种measurement）。

设置action字段为```start```来开始执行测量。对于这两个测量，我们都需要指定labelSelectors来让它知道该考虑哪个pods。PodStartupLatency支持阈值。如果99th的延迟超过这个阈值时，测试就会失败。

```yaml
steps:
- name: Start measurements
  measurements:
  - Identifier: PodStartupLatency
    Method: PodStartupLatency
    Params:
      action: start
      labelSelector: group = test-pod
      threshold: 20s
  - Identifier: WaitForControlledPodsRunning
    Method: WaitForControlledPodsRunning
    Params:
      action: start
      apiVersion: apps/v1
      kind: Deployment
      labelSelector: group = test-deployment
      operationTimeout: 120s
```
一旦我们创建了这两个测量，我们就可以进行下一步创建deployment。
我们需要指定想让deployment在哪个namespace创建，每个namespace有多少个deployment。另外，我们需要为deployment指定模板。
现在，我们假设这个模板允许在deployment中指定replicas副本数。

```yaml
- name: Create deployment
  phases:
  - namespaceRange:
      min: 1
      max: 1
    replicasPerNamespace: 1
    tuningSet: Uniform1qps
    objectBundle:
    - basename: test-deployment
      objectTemplatePath: "deployment.yaml"
      templateFillMap:     # 表示deployment.yaml文件支持在外部定义Replicas参数，并填充进去。
        Replicas: 10
```
现在，我们需要等待这个deployment的pod处于running状态:
```yaml
- name: Wait for pods to be running
  measurements:
  - Identifier: WaitForControlledPodsRunning
    Method: WaitForControlledPodsRunning
    Params:
      action: gather
```
现在我们可以收集PodStartupLatency的结果:
```yaml
- name: Measure pod startup latency
  measurements:
  - Identifier: PodStartupLatency
    Method: PodStartupLatency
    Params:
      action: gather
```
整个 `config.yaml` 如下:
```yaml
name: test

namespace:
  number: 1

tuningSets:
- name: Uniform1qps
  qpsLoad:
    qps: 1

steps:
- name: Start measurements
  measurements:
  - Identifier: PodStartupLatency
    Method: PodStartupLatency
    Params:
      action: start
      labelSelector: group = test-pod
      threshold: 20s
  - Identifier: WaitForControlledPodsRunning
    Method: WaitForControlledPodsRunning
    Params:
      action: start
      apiVersion: apps/v1
      kind: Deployment
      labelSelector: group = test-deployment
      operationTimeout: 120s
- name: Create deployment
  phases:
  - namespaceRange:
      min: 1
      max: 1
    replicasPerNamespace: 1
    tuningSet: Uniform1qps
    objectBundle:
    - basename: test-deployment
      objectTemplatePath: "deployment.yaml"
      templateFillMap:
        Replicas: 10
- name: Wait for pods to be running
  measurements:
  - Identifier: WaitForControlledPodsRunning
    Method: WaitForControlledPodsRunning
    Params:
      action: gather
- name: Measure pod startup latency
  measurements:
  - Identifier: PodStartupLatency
    Method: PodStartupLatency
    Params:
      action: gather
```

默认情况下，clusterloader会删除自动创建的名称空间，所以我们不需要担心清理集群。

现在，为了完成我们的第一个测试，需要指定deployment 模板。你可以将它看做是普通的k8s对象，但是带模板。
CL2 默认添加 `Name`参数，可以在模板中使用。在我们的config中，还传递了 `Replicas` 参数。 我们必须要记住去设置正确的label，让 PodStartupLatency
和 WaitForControlledPodsRunning watch正确的pod。因此 (`deployment.yaml` 文件)如下:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Name}}
  labels:
    group: test-deployment
spec:
  replicas: {{.Replicas}}
  selector:
    matchLabels:
      group: test-pod
  template:
    metadata:
      labels:
        group: test-pod
    spec:
      containers:
      - image: zbxx/k8s.gcr.io_pause:3.1
        name: {{.Name}}
```
## 5 执行test
在运行test之前，确保 kubeconfig的当前上下文指向kind集群：

```bash
$ kubectl config current-context
> kind-test-cluster
```
运行以下，来执行test：
```bash
cd clusterloader2/
go run cmd/clusterloader.go --testconfig=config.yaml --provider=kind --kubeconfig=${HOME}/.kube/config --v=2
```

At the end of clusterloader output you should see pod startup latency:
```json
{
  "data": {
    "Perc50": 7100.534796,
    "Perc90": 8702.523037,
    "Perc99": 9122.894555
  },
  "unit": "ms",
  "labels": {
    "Metric": "pod_startup"
  }
},
```
`pod_startup` measures time since pod was created
until it was observed via watch as running.

You should also see that test succeeded:
```
--------------------------------------------------------------------------------
Test Finished
Test: ./config.yaml
Status: Success
--------------------------------------------------------------------------------
```
As an exercise you can modify threshold for PodStartupLatency
below values you've observed in your run and check if test fails.

## 6 Delete kind cluster
To delete kind cluster, run:
```bash
kind delete cluster --name test-cluster
```

## 7 Running 100-node scale test

Here you can find general purpose [Load test].
This test is release-blocking test we use to evaluate scalability of kubernetes.
It consists of 3 main phases:
- Creating objects
- Scaling objects to size between (50%, 150%) of their original size
- Deleting objects

It can be used to test clusters starting from 100 nodes up to 5k nodes.
Load test will create, roughly 30 * nodes pod objects. It will create:
- deployments
- jobs
- statefulsets
- services
- secrets
- configmaps

There are small (5 pods), medium (30 pods) and big (250 pods) versions of
deployments, jobs and statefulsets.

First, you need to create 100-nodes cluster
then you can run cluster scale test with this command:

```bash
./run-e2e-with-prometheus-fw-rule.sh cluster-loader2 --testconfig=./testing/load/config.yaml --nodes=100 --provider=gke --enable-prometheus-server=true --kubeconfig=${HOME}/.kube/config --v=2
```

`--enable-prometheus-server=true` deploys prometheus server
using prometheus-operator.

There are various measurements that depend on prometheus metrics, for example:
- API responsiveness - measures the latency of requests to kube-apiserver
- Scheduling throughput
- NodeLocalDNS latency

[Kind]: https://kind.sigs.k8s.io/
[GVM install]: https://github.com/moovweb/gvm#installing
[Kind config]: https://kind.sigs.k8s.io/docs/user/quick-start/#advanced
[Kind install]: https://kind.sigs.k8s.io/docs/user/quick-start#installation
[Load test]: https://github.com/zhengbingxian/clusterloader2/tree/master/testing/load
[Measurements]: https://github.com/zhengbingxian/clusterloader2/tree/master/#measurement
