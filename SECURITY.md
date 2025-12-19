# Security Policy

## Reporting Security Issues

If you discover a security vulnerability, please report it responsibly.

**Do NOT** open a public GitHub issue for security vulnerabilities.

### How to Report

1. Email the maintainers directly, or
2. Open a private security advisory on GitHub

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

## Security Best Practices

When using KubeAI Autoscaler:

### RBAC

Use minimal RBAC permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kubeai-autoscaler
rules:
- apiGroups: ["apps"]
  resources: ["deployments/scale"]
  verbs: ["get", "update"]
- apiGroups: ["kubeai.io"]
  resources: ["aiinferenceautoscalerpolicies"]
  verbs: ["get", "list", "watch", "update"]
```

### Network Policies

Restrict controller network access:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kubeai-autoscaler
spec:
  podSelector:
    matchLabels:
      app: kubeai-autoscaler
  policyTypes:
  - Egress
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - port: 443  # Kubernetes API
    - port: 9090 # Prometheus
```

### Secrets

- Never hardcode credentials
- Use Kubernetes Secrets or external secret managers
- Rotate credentials regularly

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.1.x   | ✅ |

## Responsible Disclosure

We follow responsible disclosure practices and will acknowledge reports within 48 hours.
