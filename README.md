# Nephe

## Overview

Nephe supports security policy enforcement across different Public
Clouds. It translates and enforces Antrea NetworkPolicies on Public Cloud
Virtual Machines using Cloud Network Security Groups. The project deploys a
`nephe-controller` Pod in a Kubernetes cluster. Antrea must be the CNI of the
Kubernetes cluster and provides Antrea NetworkPolicy (ANP) CRD.

## Dependencies

* [Golang](https://go.dev/dl/): Nephe is developed and tested with go
  version 1.17.
* [Docker](https://docs.docker.com/install/): Nephe is developed and tested
  with Docker 20.10.17.
* [Antrea](https://github.com/antrea-io/antrea/): Provides Antrea
  NetworkPolicy (ANP) CRD, a controller that computes ANP spans, and an agent as
  K8s CNI.
* [cert-manager](https://github.com/jetstack/cert-manager): Provides in cluster
  authentication for Nephe Controller CR webhook servers.

## Getting Started

Getting started with Nephe is simple and fast. You can follow the
[Getting Started](docs/getting-started.md) guide to try it out.

## Contributing

The Antrea community welcomes new contributors. We are waiting for your PRs!

* Before contributing, please get familiar with our [Code of Conduct](CODE_OF_CONDUCT.md).
* Check out the [Developer Guide](docs/developers-guide.md) for information
  about setting up your development environment and our contribution workflow.
* Learn about Nephe's [Architecture and Design](docs/architecture.md).
  Your feedback is more than welcome!
* Check out [Open Issues](TBD).

## License

Nephe is licensed under the [Apache License, version 2.0](LICENSE)

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fantrea-io%2Fantrea.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fantrea-io%2Fantrea?ref=badge_large)
