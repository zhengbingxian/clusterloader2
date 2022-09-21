# ClusterLoader

ClusterLoader2 (CL2) 是 "自带yaml" 的Kubernetes负载测试工具，是官方K8s scalability和performance测试框架。

The CL2 tests 使用半声明的方式写到yaml文件中。A test定义了集群应该处于的一组状态（例如我想运行10K个pod，2k个cluster-ip服务，5个daemonsets等）。并指定多快能达到给定的状态（例如pod throughput吞吐量）。此外，它还定义了应该测量哪些性能特征 [测量值列表](#3 Measurement)。
另外, CL2 使用[Prometheus](#4 prometheus-metrics)在测试期间提供了额外的可观察性。

CL2 test API 的描述见 [此][api].

## 1 Getting started

如果你是初学者，查看 [Getting started] 指导.

### 1.1 参数

#### 必填参数

以下参数必填.
 - kubeconfig - kubeconfig文件的路径.
 - testconfig - test config文件路径. 如果需要运行多个测试，则可以多次使用此参数。
 - provider - 集群提供者,选项为: gce, gke, kind, kubemark, aws, local, vsphere, skeleton

#### 可选参数

 - nodes - 集群中的nodes数量。如果不指定，测试将分配到所有可调度集群节点的数量。
 - report-dir - summaries总结文件存放的目录。如果不指定，总结将打印到标准日志中。
 - mastername - master节点的名字
 - masterip - master节点的DNS Name / IP
 - testoverrides - path to file with overrides.
 - kubelet-port - Port of the kubelet to use (*default: 10250*)

## 2 Tests

### 2.1 Test definition

测试定义是 [api] 的实例，以json或yaml形式。API的初衷和描述可以在 [design doc]查看.
测试的定义以及单个对象的定义都支持模板化。测试定义的模板带一个预定义的值- ```{{.Nodes}}```，表示集群中可调度节点的数量。

### 2.2 Modules

ClusterLoader2  通过 [Module API](https://github.com/zhengbingxian/clusterloader2/blob/master/api/types.go#:~:text=type%20Module%20struct%20%7B)来支持测试配置文件的模板化.(最新代码的结构体已经没有path和params参数了)
通过 Module API,你可以将一个测试文件分解成多个module文件. A module可以被test或其他module参数化和使用多次。
这提供一种方便的方法来避免复制粘贴和维护超长、不可读的测试配置（例如看这900行的localconfig--有什么好看的-  -）1](#f1)</sup>.

[TODO(mm4tt)]: <> "Point to the load config based on modules here once we migrate it"

### 2.3 Object template

Object template对象模板类似于标准的k8s对象定义，唯一的区别是模板机制。可以通过 ```templateFillMap``` 将参数从test definition传递到object template。两个使用可用的参数是```{{.Name}}``` and ```{{.Index}}```，用来指定对象名称和对象副本索引。
例子: [load deployment template].

### 2.4 Overrides

Overrides 允许向模板注入新的变量值。许多test都定义了输入参数。输入参数是一个可能由测试框架提供的变量。因为输入参数是可选的，如果给定变量不存在，会使用 ```DefaultParam```方法进行处理。
overrides例子见: [overrides]

#### 2.4.1 **传递环境变量**

是可以依赖环境变量，代替在文件中使用override。只有以 `CL2_`前缀开头的变量才会被解析并在脚本中可用。

环境变量可以与 `DefaultParam` 方法一起使用，以提供合理的默认值。

1、**在shell中设置变量**

```shell
export CL2_ACCESS_TOKENS_QPS=5
```

2、**在测试定义test definition中使用它**

```yaml
{{$qpsPerToken := DefaultParam .CL2_ACCESS_TOKENS_QPS 0.1}}
```

## 3 Measurement

目前可用的measurements 有:
- **APIAvailabilityMeasurement** \
  这个测量值收集集群控制平面可用性的信息。有两种略微不同的测试方法：
  - cluster-level availability, 通过定期向 `/readyz`发出一个API调用。
  - host-level availability, 通过定期轮询每个控制平面主机的 `/readyz` 端点.
    - 需要开启 [exec service](https://github.com/zhengbingxian/clusterloader2/tree/master/pkg/execservice) .
- **APIResponsivenessPrometheusSimple** \
这个测量值根据prometheus收集的数据，为服务器api调用，创建延迟和数量的百分位数值。Api calls按 resource, subresource, verb and scope划分。 \
这个测量值用来验证[API call latencies SLO]是否满足。如果prometheus不可用，会跳过。
- **APIResponsivenessPrometheus** \
同上，区别为延迟和数量的summary。
- **CPUProfile** \
这个测量值收集指定组件的cpu使用概况，通过pprof。
- **EtcdMetrics** \
这个测试量收集一组etcd指标及其数据库的大小。
- **MemoryProfile** \
这个测量值收集指定组件的内存使用概况，通过pprof。
- **MetricsForE2E** \
这个测量值收集metrics，从kube-apiserver, controller manager,scheduler 和可选择的所有kubelets.
- **PodStartupLatency** \
这个测量值验证 [pod startup SLO] 是否满足。
- **ResourceUsageSummary** \
这个测量值收集每个组件的资源使用情况。在收集执行期间，收集到的数据将被转换为 90th, 99th and 100th summary。
可选：可以为此测量值提供资源约束文件。 资源约束文件指定给定组件的cpu和/或内存的约束。如果违反任何约束，将返回错误，导致测试失败。
- **SchedulingMetrics** \
这个测量值收集一组scheduler metrics。
- **SchedulingThroughput** \
这个测量值收集scheduling throughput调度吞吐量.
- **Timer** \
Timer allows for measuring latencies of certain parts of the test
(single timer allows for independent measurements of different actions).
- **WaitForControlledPodsRunning** \
这个测量值作为屏障运行。等待指定的控制对象的所有pod running。(ReplicationController, ReplicaSet, Deployment, DaemonSet and Job)。controlling对象可以通过标签选择器label、字段选择器field，和namespace来指定。如果超时，测试将继续运行，并记录失败(记录原因)。
- **WaitForRunningPods** \
这个测量值作为屏障运行。等待指定数量的pod变成running。Pods可以通过 label selector, field selector and namespace来指定。
In case of timeout test continues to run, with error (causing marking test as failed) being logged.
- **Sleep** \
这个测量值作为屏障运行。sleep给定的时间。

## 4 Prometheus metrics

有两种方法可以从集群中的pod中抓取指标：
- **ServiceMonitor** \
允许从service中抓取所有pod的指标。例子参考： [Service monitor]
- **PodMonitor** \
允许从给定的label中抓取所有pod的指标。例子参考： [Pod monitor]

## 5 Vendor

Vendor is created using [Go modules].

---

<sup><b id="f1">1.</b> As an example and anti-pattern see the 900 line [load test config.yaml](https://github.com/kubernetes/perf-tests/blob/92cc27ff529ae3702c87e8f154ea62f3f2d8e837/clusterloader2/testing/load/config.yaml) we ended up maintaining at some point. [↩](#a1)</sup>


[api]: https://github.com/zhengbingxian/clusterloader2/blob/master/api/types.go
[API call latencies SLO]: https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md
[exec service]: https://github.com/zhengbingxian/clusterloader2/tree/master/pkg/execservice
[design doc]: https://github.com/zhengbingxian/clusterloader2/blob/master/docs/design.md
[Go modules]: https://blog.golang.org/using-go-modules
[Getting started]: https://github.com/zhengbingxian/clusterloader2/blob/master/docs/GETTING_STARTED.md
[load deployment template]: https://github.com/zhengbingxian/clusterloader2/blob/master/testing/load/deployment.yaml
[load test]: https://github.com/zhengbingxian/clusterloader2/blob/master/testing/load/config.yaml
[overrides]: https://github.com/zhengbingxian/clusterloader2/blob/master/testing/density/scheduler/pod-affinity/overrides.yaml
[pod startup SLO]: https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md
[Service monitor]: https://github.com/zhengbingxian/clusterloader2/blob/master/pkg/prometheus/manifests/default/prometheus-serviceMonitorKubeProxy.yaml
[Pod monitor]: https://github.com/zhengbingxian/clusterloader2/blob/master/pkg/prometheus/manifests/default/prometheus-podMonitorNodeLocalDNS.yaml
