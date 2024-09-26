# cr_viewer

`cr_viewer` is a command-line tool written in Go that generates a sample Custom Resource (CR) specification based on a given CustomResourceDefinition (CRD) YAML file. It helps in visualizing the structure and default values of the CR spec.

## Features

- Generates a sample CR spec from a CRD YAML file.
- Outputs the spec in YAML format.
- Supports cross-platform usage (macOS, Linux, Windows).

## Prerequisites

- Go 1.16 or later
- `oc` CLI tool for fetching CRD (OpenShift CLI)

