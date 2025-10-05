# BGswitch - Epic & Implementation Plan

**Progressive Deployment Operator for Kubernetes**

## 🎯 Project Goals

Build a production-ready Kubernetes operator that automates safe, progressive deployments with:
- Zero-downtime deployments
- Automatic health validation via metrics
- Instant rollback on failures
- Simple, declarative API

**Target Completion:** 3 weeks (60-75 hours total)

**Success Metrics:**
- Working operator managing real deployments
- Prometheus integration functional
- Auto-rollback working correctly
- Complete documentation and examples
- Open-sourced on GitHub

---

## 🏗️ Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────────┐
│                        BGswitch Operator                     │
│                                                              │
│  ┌────────────────┐         ┌─────────────────┐            │
│  │ CRD Controller │────────▶│ Canary Manager  │            │
│  └────────────────┘         └─────────────────┘            │
│         │                            │                       │
│         │                            ▼                       │
│         │                   ┌─────────────────┐             │
│         │                   │ Traffic Shifter │             │
│         │                   └─────────────────┘             │
│         │                            │                       │
│         ▼                            ▼                       │
│  ┌────────────────┐         ┌─────────────────┐            │
│  │ Metrics Client │────────▶│ Health Analyzer │            │
│  └────────────────┘         └─────────────────┘            │
└─────────────────────────────────────────────────────────────┘
         │                              │
         ▼                              ▼
┌──────────────────┐          ┌──────────────────┐
│   Prometheus     │          │   Kubernetes API │
└──────────────────┘          └──────────────────┘
```

### Components

**1. CRD Controller**
- Watches ProgressiveDeployment CRDs
- Orchestrates deployment lifecycle
- Manages state transitions
- Reconciliation loop

**2. Canary Manager**
- Creates canary Deployment
- Manages canary replicas
- Calculates replica distribution
- Cleanup on completion

**3. Traffic Shifter**
- Adjusts replica counts for traffic distribution
- Updates Service selectors (if needed)
- Implements traffic shifting strategies

**4. Metrics Client**
- Queries Prometheus API
- Fetches error rates, latency
- Executes custom PromQL queries
- Caches metric data

**5. Health Analyzer**
- Evaluates metrics against thresholds
- Determines deployment health
- Triggers promotion or rollback
- Logs decisions

### State Machine

```
     [Created]
        │
        ▼
   [Initializing] ──────────────┐
        │                       │
        ▼                       │
   [Analyzing]                  │
        │                       │
   ┌────┴────┐                  │
   │         │                  │
Healthy   Unhealthy             │
   │         │                  │
   ▼         ▼                  │
[Promoting] [RollingBack]       │
   │         │                  │
   │         └──────────────────┤
   │                            │
   ▼                            ▼
Next Step?              [RolledBack]
   │
   ├─Yes─▶ [Analyzing]
   │
   └─No──▶ [Completed]
```

### CRD Structure

```go
type ProgressiveDeploymentSpec struct {
    // Target deployment to progressively update
    TargetDeployment string `json:"targetDeployment"`
    
    // Canary steps as percentages [10, 25, 50, 100]
    CanarySteps []int `json:"canarySteps"`
    
    // Duration to wait at each step before analysis
    StepDuration metav1.Duration `json:"stepDuration"`
    
    // Prometheus metrics for health validation
    Metrics MetricsConfig `json:"metrics"`
    
    // Auto-promote if metrics are healthy
    AutoPromote bool `json:"autoPromote"`
}

type MetricsConfig struct {
    // Prometheus endpoint (default: in-cluster)
    PrometheusURL string `json:"prometheusUrl,omitempty"`
    
    // Error rate threshold
    ErrorRate MetricThreshold `json:"errorRate,omitempty"`
    
    // Latency threshold
    Latency MetricThreshold `json:"latency,omitempty"`
}

type MetricThreshold struct {
    // PromQL query to execute
    Query string `json:"query"`
    
    // Threshold value
    Threshold float64 `json:"threshold"`
}

type ProgressiveDeploymentStatus struct {
    // Current phase: Initializing, Analyzing, Promoting, RollingBack, etc.
    Phase string `json:"phase"`
    
    // Current canary step (0-indexed)
    CurrentStep int `json:"currentStep"`
    
    // Current canary percentage
    CanaryPercentage int `json:"canaryPercentage"`
    
    // Canary deployment name
    CanaryDeployment string `json:"canaryDeployment,omitempty"`
    
    // Health status: Healthy, Unhealthy, Unknown
    HealthStatus string `json:"healthStatus"`
    
    // Last metric values
    Metrics map[string]float64 `json:"metrics,omitempty"`
    
    // Conditions
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

---

## 📅 Implementation Plan

### Week 1: Foundation & Core Logic

#### Day 1-2: Project Setup (6-8 hours)

**Tasks:**
- [x] Initialize Go module
- [x] Set up kubebuilder project
- [x] Create CRD scaffold
- [x] Define types Spec
- [x]  Define type Status
- [x] Set up Git repository
- [x] Create basic project structure

**Deliverables:**
```
bgswitch/
├── api/v1alpha1/
│   └── progressivedeployment_types.go
├── controllers/
│   └── progressivedeployment_controller.go
├── config/
│   ├── crd/
│   ├── rbac/
│   └── manager/
├── Makefile
├── go.mod
└── README.md
```

**Commands:**
```bash
kubebuilder init --repo github.com/ghanatava/bgswitch
kubebuilder create api --group apps --version v1alpha1 --kind ProgressiveDeployment --resource --controller
```

#### Day 3-4: Controller Logic (8-10 hours)

**Tasks:**
- [ ] Implement reconciliation loop
- [ ] Watch target Deployment
- [ ] Create canary Deployment on CRD creation
- [x] Implement basic state machine
- [x] Add logging

**Core Logic:**
```go
func (r *ProgressiveDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch ProgressiveDeployment CRD
    // 2. Fetch target Deployment
    // 3. Determine current state
    // 4. Execute state-specific logic:
    //    - Initializing: Create canary
    //    - Analyzing: Check metrics
    //    - Promoting: Adjust replicas
    //    - RollingBack: Restore stable
    // 5. Update Status
    // 6. Requeue if needed
}
```

**Test:**
```bash
# Run operator locally
make run

# Apply test CRD
kubectl apply -f config/samples/apps_v1alpha1_progressivedeployment.yaml

# Check logs
# Verify canary Deployment created
```

#### Day 5-7: Traffic Shifting (8-10 hours)

**Tasks:**
- [ ] Implement replica calculation logic
- [ ] Adjust stable and canary replica counts
- [ ] Handle edge cases (min replicas, rounding)
- [ ] Implement step progression
- [ ] Add step timer logic

**Traffic Distribution Math:**
```
Total Replicas = Stable Replicas + Canary Replicas
Canary % = 25%

If Total = 10:
  Canary Replicas = 10 * 0.25 = 2.5 → 3
  Stable Replicas = 10 - 3 = 7

Traffic split ≈ 70% stable, 30% canary
```

**Implementation:**
```go
func calculateReplicas(total int, canaryPercent int) (stable, canary int) {
    canary = int(math.Ceil(float64(total) * float64(canaryPercent) / 100.0))
    stable = total - canary
    return stable, canary
}
```

**Test:**
- Deploy sample app with 10 replicas
- Apply ProgressiveDeployment with steps [10, 25, 50, 100]
- Verify replica counts at each step
- Check traffic distribution

---

### Week 2: Metrics & Automation

#### Day 8-10: Prometheus Integration (10-12 hours)

**Tasks:**
- [ ] Add Prometheus client library
- [ ] Implement metrics client
- [ ] Query Prometheus API
- [ ] Parse PromQL results
- [ ] Handle connection errors

**Dependencies:**
```go
import (
    promapi "github.com/prometheus/client_golang/api"
    promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)
```

**Metrics Client:**
```go
type MetricsClient struct {
    prometheusAPI promv1.API
}

func (m *MetricsClient) QueryMetric(ctx context.Context, query string) (float64, error) {
    result, _, err := m.prometheusAPI.Query(ctx, query, time.Now())
    // Parse result
    // Return value
}
```

**Test:**
- Install Prometheus in test cluster
- Deploy sample app with metrics
- Query metrics from operator
- Verify values

#### Day 11-12: Health Analysis (8-10 hours)

**Tasks:**
- [ ] Implement health analyzer
- [ ] Compare metrics to thresholds
- [ ] Determine overall health status
- [ ] Log analysis results
- [ ] Handle missing metrics gracefully

**Health Logic:**
```go
type HealthAnalyzer struct {
    metricsClient *MetricsClient
}

func (h *HealthAnalyzer) Analyze(ctx context.Context, pd *ProgressiveDeployment) (bool, error) {
    // Query error rate
    errorRate := queryMetric(pd.Spec.Metrics.ErrorRate.Query)
    
    // Query latency
    latency := queryMetric(pd.Spec.Metrics.Latency.Query)
    
    // Compare to thresholds
    healthy := errorRate < pd.Spec.Metrics.ErrorRate.Threshold &&
               latency < pd.Spec.Metrics.Latency.Threshold
    
    return healthy, nil
}
```

**Test:**
- Inject errors in canary deployment
- Verify health analyzer detects failures
- Check that analysis is correct

#### Day 13-14: Auto-Promotion & Rollback (8-10 hours)

**Tasks:**
- [ ] Implement auto-promotion logic
- [ ] Wait for stepDuration
- [ ] Trigger health analysis
- [ ] Promote if healthy
- [ ] Implement rollback logic
- [ ] Restore stable deployment
- [ ] Clean up canary

**Promotion Flow:**
```go
// After stepDuration expires
if pd.Spec.AutoPromote {
    healthy, err := healthAnalyzer.Analyze(ctx, pd)
    if err != nil {
        // Handle error
    }
    
    if healthy {
        // Move to next step
        promoteToNextStep(pd)
    } else {
        // Rollback
        initiateRollback(pd)
    }
}
```

**Rollback Flow:**
```go
func (r *Reconciler) Rollback(ctx context.Context, pd *ProgressiveDeployment) error {
    // Set stable replicas to original count
    // Set canary replicas to 0
    // Update status
    // Mark as RolledBack
}
```

**Test:**
- Successful promotion path (all metrics healthy)
- Failed promotion path (bad metrics trigger rollback)
- Manual rollback command

---

### Week 3: Polish & Release

#### Day 15-17: CLI & Manual Controls (6-8 hours)

**Tasks:**
- [ ] Create CLI tool (`bgswitch`)
- [ ] Implement `promote` command
- [ ] Implement `rollback` command
- [ ] Implement `pause` command
- [ ] Implement `status` command

**CLI Structure:**
```go
// cmd/bgswitch/main.go
func main() {
    rootCmd := &cobra.Command{
        Use:   "bgswitch",
        Short: "BGswitch CLI",
    }
    
    rootCmd.AddCommand(promoteCmd)
    rootCmd.AddCommand(rollbackCmd)
    rootCmd.AddCommand(statusCmd)
    rootCmd.AddCommand(pauseCmd)
    
    rootCmd.Execute()
}
```

**Test:**
```bash
bgswitch promote my-app
bgswitch rollback my-app
bgswitch status my-app
```

#### Day 18-19: Documentation (8-10 hours)

**Tasks:**
- [ ] Update README with examples
- [ ] Write architecture documentation
- [ ] Create configuration reference
- [ ] Write metrics guide
- [ ] Add diagrams
- [ ] Create example manifests

**Documents:**
- `docs/architecture.md` - System design
- `docs/installation.md` - Setup guide
- `docs/configuration.md` - CRD reference
- `docs/metrics.md` - Prometheus setup
- `examples/` - Sample applications

#### Day 20-21: Testing & Release (8-10 hours)

**Tasks:**
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] E2E test with sample app
- [ ] Fix bugs
- [ ] Build Docker image
- [ ] Push to registry
- [ ] Tag v0.1.0 release

**E2E Test:**
```bash
# Setup
make cluster-up
make deploy

# Deploy sample app
kubectl apply -f examples/demo-app/

# Create progressive deployment
kubectl apply -f examples/demo-app/progressive.yaml

# Watch progression
kubectl get progressivedeployment demo-app -w

# Inject failure
kubectl apply -f examples/demo-app/bad-version.yaml

# Verify rollback
kubectl get pods

# Cleanup
make cluster-down
```

**Release Checklist:**
- [ ] All tests passing
- [ ] Documentation complete
- [ ] Examples working
- [ ] Docker image built
- [ ] Git tags created
- [ ] GitHub release published

---

## 🎯 Milestones

### Milestone 1: MVP (End of Week 1)
**Goal:** Basic operator that creates canary deployments and shifts traffic

**Definition of Done:**
- CRD defined and installed
- Controller creates canary Deployment
- Traffic shifting works (replica adjustment)
- State machine implemented
- Can be run locally

### Milestone 2: Smart Automation (End of Week 2)
**Goal:** Metrics-based decisions and auto-rollback

**Definition of Done:**
- Prometheus integration working
- Health analysis functional
- Auto-promotion implemented
- Auto-rollback on failures
- Status reporting accurate

### Milestone 3: Production Ready (End of Week 3)
**Goal:** Polished, documented, released

**Definition of Done:**
- CLI tool functional
- Comprehensive documentation
- Working examples
- Tests passing
- Docker image published
- GitHub release v0.1.0

---

## 🧪 Testing Strategy

### Unit Tests
```go
// Test replica calculation
func TestCalculateReplicas(t *testing.T) {
    tests := []struct {
        total, percent int
        wantStable, wantCanary int
    }{
        {10, 10, 9, 1},
        {10, 25, 7, 3},
        {10, 50, 5, 5},
    }
    // Run tests
}
```

### Integration Tests
```go
// Test controller reconciliation
func TestReconcile(t *testing.T) {
    // Setup fake K8s client
    // Create ProgressiveDeployment
    // Trigger reconcile
    // Assert canary created
}
```

### E2E Tests
```bash
# Full deployment lifecycle test
./test/e2e/test-progressive-deployment.sh
```

---

## 📊 Success Criteria

**Technical:**
- ✅ Operator deploys successfully
- ✅ Canary deployments work correctly
- ✅ Traffic shifting is accurate (±5%)
- ✅ Metrics integration functional
- ✅ Auto-rollback triggers correctly
- ✅ No resource leaks (deployments cleaned up)

**Quality:**
- ✅ Code coverage >70%
- ✅ All E2E tests pass
- ✅ Documentation complete
- ✅ Examples work out of the box

**Adoption:**
- 🎯 GitHub stars: 10+ (first week)
- 🎯 Usable by others without asking questions
- 🎯 Mentioned in job interviews as portfolio piece

---

## 🚀 Future Enhancements (Post v0.1.0)

### v0.2.0 - Advanced Features
- Istio VirtualService integration (precise traffic control)
- Webhook notifications (Slack, Discord, PagerDuty)
- Advanced statistical analysis (t-tests, confidence intervals)
- Multi-metric weighted scoring

### v0.3.0 - Enterprise Features
- Blue-Green deployment mode
- A/B testing with traffic splitting by headers
- Custom webhook integrations
- Web dashboard UI
- Multi-cluster support

### v1.0.0 - Production Grade
- High availability operator
- Performance optimizations
- Security hardening
- Comprehensive observability
- Enterprise support options

---

## 📝 Notes & Decisions

### Technology Choices

**Why Kubernetes Operator?**
- Native K8s integration
- Declarative API
- Built-in RBAC and security
- Scalable and reliable

**Why Go + kubebuilder?**
- Standard for K8s operators
- Excellent K8s client libraries
- Fast, compiled, small binaries
- Great tooling (kubebuilder)

**Why Replica-based Traffic Shifting?**
- Works without service mesh
- Simple and predictable
- No external dependencies
- Good enough for v0.1.0

**Why Prometheus?**
- Industry standard for metrics
- PromQL is powerful
- Easy to integrate
- Most companies already use it

### Open Questions

- **Q:** Should we support Istio in v0.1.0?
  **A:** No, keep it simple. Replica-based is enough. Add Istio in v0.2.0.

- **Q:** Should we support multiple deployment strategies (canary, blue-green)?
  **A:** Start with canary only. Add blue-green in v0.3.0.

- **Q:** How to handle manual approvals?
  **A:** Set `autoPromote: false` and use CLI commands.

---

**Let's build this! 🚀**

Time to start coding. Follow the plan, ship fast, iterate based on usage.
