# ping-exporter

Prometheus exporter for network round trip time in cluster. Deploy with kubernetes DaemonSet, A pod will ping every
other pod belonging to the same DaemonSet.

## Metrics

| Name | Type | Help |
| ---- | ---- | ---- |
| ping_loss_percent | gauge | Packet loss in percent |
| ping_rtt_best_seconds | gauge | Best round trip time in seconds |
| ping_rtt_mean_seconds | gauge | Mean round trip time in seconds |
| ping_rtt_std_deviation_seconds | gauge | Standard deviation in seconds |
| ping_rtt_worst_seconds | gauge | Worst round trip time in seconds |

## Installation

```
kubectl apply -f manifests
```
