# Demo App Examples

This directory contains example manifests for testing BGswitch progressive deployments.

## Files

- `deployment.yaml` - Sample nginx deployment with service (the target deployment)
- `fast-rollout.yaml` - Quick 2-step progressive deployment (30 seconds total)
- `conservative-rollout.yaml` - Slow 6-step deployment with manual approval (12+ minutes)

## Prerequisites

- BGswitch operator installed and running
- Kubernetes cluster with kubectl configured

## Quick Start

### 1. Deploy the Demo App
```bash
kubectl apply -f examples/demo-app/deployment.yaml