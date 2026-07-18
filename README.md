# Greenbone Community Edition on Kubernetes

This repository provides a custom Helm chart for deploying the **Greenbone Community Edition (GVM)** on Kubernetes. It is designed specifically for enterprise environments with two major architectural requirements:
1. **CloudNativePG (CNPG) Cluster**: Utilizes a highly available, cloud-native PostgreSQL cluster for GVM's back-end database instead of a single PostgreSQL container.
2. **Airgapped Feed Updates**: Avoids internet dependency by hosting an in-pod Go API sidecar (`data-fetcher`) that receives feed archives (`gvm-feed.tar.gz`) locally via HTTP POST and extracts them directly into shared persistent volumes.

---

## Architecture Overview

```
                          [ Traefik IngressRoute ]
                                     |
                 +-------------------+-------------------+
                 | (Paths matching /upload)              | (Other paths)
                 v                                       v
         [ data-fetcher:8080 ]                    [ gsa-nginx:443 ]
                 |                                       |
                 | (extracts tarball)                    | (web UI)
                 v                                       v
        +-------------------------------------------------+
        |           Shared Persistent Volumes              |
        |  (vt-data, scap-data, cert-data, gvmd-data)    |
        +-------------------------------------------------+
                                 ^
                                 | (reads feed data)
                              [ gvmd ] <-------> [ CNPG PostgreSQL Cluster ]
```

### Key Components
* **GVM Helm Chart (`helm/greenbone/`)**:
  * Manages the deployment of the GVM Core pod (containing `gvmd`, `ospd-openvas`, `redis`, `openvasd`, `gsad`, and Nginx frontend containers).
  * Hooks in the CloudNativePG `cluster` sub-chart as a dependency.
  * Dynamically renders custom Kubernetes manifests (e.g. Traefik `IngressRoute` and `ServersTransport`) using the `.Values.extraObjects` array helper.
* **Data Fetcher API (`data-fetcher/`)**:
  * A lightweight Go application running as a sidecar container in the GVM Core pod.
  * Exposes `/upload` (POST) to receive the GVM feed archive.
  * Extracts archives directly to target persistent volumes:
    - Files in `plugins/` -> `vt-data`
    - Files in `scap-data/` -> `scap-data`
    - Files in `cert-data/` -> `cert-data`
    - Files in `data-objects/` -> `gvmd-data`
    - Files in `notus/` -> `notus-data`

---

## Getting Started

### 1. Build and Publish the Data Fetcher Container
Since this is an airgapped setup, build the `data-fetcher` container and push it to your internal container registry (e.g., Harbor):

```bash
# Build the Docker image
docker build -t private-registry.local/gvm-data-fetcher:latest .

# Push to your private registry
docker push private-registry.local/gvm-data-fetcher:latest
```

### 2. Configure Credentials in Kubernetes
The Helm chart is configured to use a single existing Kubernetes secret to retrieve database and application passwords.

Create a secret named `greenbone-secrets` in your target namespace with the following keys:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: greenbone-secrets
type: Opaque
stringData:
  db-user: "gvm"
  db-password: "StrongDatabasePassword"
  db-name: "gvmd"
  admin-user: "admin"
  admin-password: "StrongAdminPassword"
  data-fetcher-password: "StrongUploadAPIToken"
```

### 3. Deploy GVM using Helm
First, download and build the sub-chart dependencies (CloudNativePG):

```bash
helm dependency build helm/greenbone/
```

Configure your `values.yaml` (especially `images`, `existingSecret`, and `extraObjects`), and install the chart:

```bash
helm install greenbone helm/greenbone/ -f values.yaml
```

---

## Performing Feed Updates

To update GVM vulnerability tests, scap, cert, and notus data, stream your compiled `gvm-feed.tar.gz` to the `/upload` API endpoint using a `Bearer` token matching the value of `data-fetcher-password` in your secret:

```bash
curl -X POST \
  -H "Authorization: Bearer <data-fetcher-password>" \
  -F "file=@gvm-feed.tar.gz" \
  https://greenbone.yourdomain.com/upload
```

Once uploaded, the `data-fetcher` automatically:
1. Validates the authentication token.
2. Extracts files directly to GVM's persistent volumes.
3. Automatically maps target directories to separate PV mounts based on file prefixes.

GVM components (`gvmd`, `ospd-openvas`, etc.) share these mounts and can instantly read the updated files on the next scan/reload cycle.
