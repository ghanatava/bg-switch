# BGswitch 🚀

**Progressive Deployment Operator for Kubernetes**

BGswitch is a Kubernetes operator that automates safe, progressive deployments with automatic traffic shifting, health monitoring, and rollback capabilities. Deploy with confidence using canary analysis and blue-green strategies.

## 🎯 Problem

Kubernetes rolling updates are risky:
- No gradual traffic shifting
- Limited health validation
- Manual rollback process
- No metric-based decision making

**Result:** Failed deployments cause downtime and impact users.

## ✨ Solution

BGswitch automates progressive deployments:
- 🔄 Gradual traffic shifting (10% → 25% → 50% → 100%)
- 📊 Metric-based health validation (Prometheus integration)
- ⚡ Automatic rollback on failures
- 🎛️ Manual promotion controls
- 📈 Real-time deployment status

## 🚀 Quick Start

### Installation

```bash
# Install the operator
kubectl apply -f https://raw.githubusercontent.com/yourusername/bgswitch/main/deploy/operator.yaml

# Verify installation
kubectl get pods -n bgswitch-system
```

### Deploy Your First Progressive Deployment

```yaml
apiVersion: bgswitch.dev/v1alpha1
kind: ProgressiveDeployment
metadata:
  name: my-app
spec:
  targetDeployment: my-app
  canarySteps: [10, 25, 50, 100]
  stepDuration: 5m
  metrics:
    errorRate:
      threshold: 1.0
      query: "rate(http_requests_errors_total{app='my-app'}[5m])"
    latency:
      threshold: 500
      query: "histogram_quantile(0.95, http_request_duration_seconds{app='my-app'})"
  autoPromote: true
```

```bash
kubectl apply -f my-progressive-deployment.yaml

# Watch progress
kubectl get progressivedeployment my-app -w
```

## 📊 How It Works

```
Stable (v1.0)     [████████████] 100%
                        ↓
Step 1 - Analyze  [███████████░] 90% stable, 10% canary
                        ↓ Metrics OK?
Step 2 - Promote  [████████░░░░] 75% stable, 25% canary
                        ↓ Metrics OK?
Step 3 - Promote  [█████░░░░░░░] 50% stable, 50% canary
                        ↓ Metrics OK?
Step 4 - Complete [░░░░░░░░░░░░] 100% canary → new stable
```

**If metrics degrade at any step:**
```
[██████░░] Rollback initiated → [████████████] Stable restored
```

## 🎛️ Features

### Progressive Traffic Shifting
- Define custom canary steps (e.g., 5%, 10%, 25%, 50%, 100%)
- Configurable duration per step
- Replica-based traffic distribution

### Health Monitoring
- Prometheus metric integration
- Custom PromQL queries
- Error rate tracking
- Latency monitoring
- Custom metric thresholds

### Automatic Rollback
- Detects metric degradation
- Instant rollback to stable version
- Preserves original deployment
- Detailed rollback reasons

### Manual Controls
```bash
# Promote to next step
kubectl bgswitch promote my-app

# Rollback immediately
kubectl bgswitch rollback my-app

# Pause progression
kubectl bgswitch pause my-app

# Get status
kubectl bgswitch status my-app
```

## 📖 Documentation

- [Architecture](docs/architecture.md)
- [Installation Guide](docs/installation.md)
- [Configuration Reference](docs/configuration.md)
- [Metrics Guide](docs/metrics.md)
- [Examples](examples/)

## 🛠️ Development

### Prerequisites
- Go 1.21+
- Docker
- kubectl
- kind (for local testing)

### Build from Source

```bash
# Clone repository
git clone https://github.com/yourusername/bgswitch.git
cd bgswitch

# Install dependencies
make install

# Run locally
make run

# Build operator image
make docker-build

# Run tests
make test
```

### Local Testing

```bash
# Create local cluster
make cluster-up

# Install operator
make deploy

# Run example
kubectl apply -f examples/demo-app/

# Cleanup
make cluster-down
```

## 🎯 Roadmap

**v0.1.0 (Current)**
- [x] Basic canary deployments
- [x] Replica-based traffic shifting
- [x] Prometheus integration
- [x] Auto-rollback

**v0.2.0 (Planned)**
- [ ] Istio/Service Mesh integration for precise traffic control
- [ ] Webhook-based notifications (Slack, Discord)
- [ ] Advanced metric analysis (statistical tests)
- [ ] Multi-metric weighted decisions

**v0.3.0 (Future)**
- [ ] Blue-Green deployment strategy
- [ ] A/B testing support
- [ ] Custom webhook integrations
- [ ] Dashboard UI

## 🤝 Contributing

Contributions welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

## 📝 License

MIT License - see [LICENSE](LICENSE)

## 🙏 Acknowledgments

Inspired by:
- [Flagger](https://flagger.app/) - Progressive delivery toolkit
- [Argo Rollouts](https://argoproj.github.io/argo-rollouts/) - Advanced deployment strategies
- [Spinnaker](https://spinnaker.io/) - Continuous delivery platform

## 📬 Contact

- GitHub Issues: [Report bugs or request features](https://github.com/yourusername/bgswitch/issues)
- Discussions: [Ask questions](https://github.com/yourusername/bgswitch/discussions)

---

**Built with ❤️ by [Your Name]**

*Making Kubernetes deployments safe and simple.*
