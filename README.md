## Potato CD

Kubernetes operator example for Potato CD.

### Task overview

We would like you to implement a Kubernetes Operator which synchronizes a Git repository with a Kubernetes cluster
when a GitOps Custom Resource is created - essentially, a stripped down “demo” version of Flux or Argo CD.

### Project overview

Built with Kubebuilder. The CRD is of API version `gitops.potato.io/v1` and called `Application`.

Used libraries:
- https://github.com/go-git/go-git/ to work with the repositories
- https://github.com/banzaicloud/k8s-objectmatcher for comparing actual and desired state

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

Beyond that, the following issues are known:
- There are some unnecessary reconciliations taking places. I first tried `reflect.DeepEqual` as suggested in the
  `kubebuilder` book, then `apiequality.Semantic.DeepEqual` with the same results. Then settled with the Banzai Cloud
  k8s-objectmatcher library

Beyond fixing, improving the above the following could be improved:
- Add some webhooks to handle defaults - e.g. branch name - and validation - for valid urls for repository
- Support private repositories by supporting different kinds of authentication and different repository URL formats
- The tests rely on an outside resource, the application-2 repository which makes writing more complex tests harder,
  and they are unstable / unreliable this way

### Testing

There are 2 sample applications in the following repositories:
- https://github.com/uvegla/potato-application-1 (Sample: config/samples/potato_application_1.yaml)
  - This contains a simple nginx deployment with 3 replicas and a service in from of them
- https://github.com/uvegla/potato-application-2 (Sample: config/samples/potato_application_2.yaml)
  - This contains a simple deployment of a cowsay webapp with a single replica

### Workflow

There was no time limit set, so I constrained myself to achieve and learn as much as I can within ~3days (24 hours).

I prioritised based on the fact that I wanted to achieve the following to at least some extent:
- Check out available technology and tooling and choose a stack to build on
- Get a working solution that implements at least some required functionality
- Try how testing works and implement the very least some dummy test case for the controller
- Add basic documentation of the project
- Prepare presentation
