## Potato CD

Kubernetes operator example for Potato CD.

### Task overview

We would like you to implement a Kubernetes Operator which synchronizes a Git repository with a Kubernetes cluster
when a GitOps Custom Resource is created - essentially, a stripped down “demo” version of Flux or Argo CD.

### Architecture overview

Built with Kubebuilder. The CRD is of API version `gitops.potato.io/v1` and called `Application`.

The `Application` CRD supports the following properties:
- `Repository`: This is a URL pointing to the repository that contains the Kubernetes manifests
- `Ref`: This is the branch name that will be used to initially clone the repository 

The following assumptions are made:
- The repository has to be public, authentication is not supported
- The repository must have a folder called `kubernetes` at the root that should contain all manifests
- Only `apps/v1/Deployment` and `v1/Service` resource types are supported
- The `default` namespaces is used

The following task items got implemented:
- The `Application` CRD that describes the repository
- Cloning the repository and periodically pulling changes that gets reconciled
- Manifests containing the supported resource types get created and updated on changes in the repository
- All resources owned by the `Application` resource gets cleaned up when itself is removed from cluster
- Some basic tests using `ginkgo` and `envtest`
 
The following constraints apply to the controller implementation:
- Changing the repository has no effect, it will be ignored by the controller
- The branch name is only taken into account at cloning, switching branches is not implemented
- Manifest removed from the repository are not cleaned up
