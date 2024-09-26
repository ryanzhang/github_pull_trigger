# cr_viewer

`cr_viewer` is a command-line tool written in Go that generates a sample Custom Resource (CR) specification based on a given CustomResourceDefinition (CRD) name or CRD YAML file. It helps in visualizing the structure and default values of the CR spec.

## Features

- Generates a sample CR spec from a CRD YAML file.
- Outputs the spec in YAML format.
- Supports cross-platform usage (macOS, Linux, Windows).

## Usage:
Usage: cr_viewer <crd_name> or cr_viewer -f <crd.yaml>

## Prerequisites
- Go 1.16 or later
- `oc` CLI tool for fetching CRD (OpenShift CLI)
- this command assume you already have a oc login session

## Example
```
bin/cr_viewer configs.imageregistry.operator.openshift.io                                                                          
spec:
    affinity:
        nodeAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
                - preference:
                    matchExpressions:
                        - key: <insert_value>
                          operator: <insert_value>
                          values:
                            - {}
                    matchFields:
                        - key: <insert_value>
                          operator: <insert_value>
                          values:
                            - {}
                  weight: <insert_value>
            requiredDuringSchedulingIgnoredDuringExecution:
                nodeSelectorTerms:
                    - matchExpressions:
                        - key: <insert_value>
                          operator: <insert_value>
                          values:
                            - {}
                      matchFields:
                        - key: <insert_value>
                          operator: <insert_value>
                          values:
                            - {}
        podAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
                - podAffinityTerm:
                    labelSelector:
                        matchExpressions:
                            - key: <insert_value>
                              operator: <insert_value>
                              values:
                                - {}
                        matchLabels: {}
                    matchLabelKeys:
                        - {}
                    mismatchLabelKeys:
                        - {}
                    namespaceSelector:
                        matchExpressions:
                            - key: <insert_value>
                              operator: <insert_value>
                              values:
                                - {}
                        matchLabels: {}
                    namespaces:
                        - {}
                    topologyKey: <insert_value>
                  weight: <insert_value>
            requiredDuringSchedulingIgnoredDuringExecution:
                - labelSelector:
                    matchExpressions:
                        - key: <insert_value>
                          operator: <insert_value>
                          values:
                            - {}
                    matchLabels: {}
                  matchLabelKeys:
                    - {}
                  mismatchLabelKeys:
                    - {}
                  namespaceSelector:
                    matchExpressions:
                        - key: <insert_value>
                          operator: <insert_value>
                          values:
                            - {}
                    matchLabels: {}
                  namespaces:
                    - {}
                  topologyKey: <insert_value>
        podAntiAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
                - podAffinityTerm:
                    labelSelector:
                        matchExpressions:
                            - key: <insert_value>
                              operator: <insert_value>
                              values:
                                - {}
                        matchLabels: {}
                    matchLabelKeys:
                        - {}
                    mismatchLabelKeys:
                        - {}
                    namespaceSelector:
                        matchExpressions:
                            - key: <insert_value>
                              operator: <insert_value>
                              values:
                                - {}
                        matchLabels: {}
                    namespaces:
                        - {}
                    topologyKey: <insert_value>
                  weight: <insert_value>
            requiredDuringSchedulingIgnoredDuringExecution:
                - labelSelector:
                    matchExpressions:
                        - key: <insert_value>
                          operator: <insert_value>
                          values:
                            - {}
                    matchLabels: {}
                  matchLabelKeys:
                    - {}
                  mismatchLabelKeys:
                    - {}
                  namespaceSelector:
                    matchExpressions:
                        - key: <insert_value>
                          operator: <insert_value>
                          values:
                            - {}
                    matchLabels: {}
                  namespaces:
                    - {}
                  topologyKey: <insert_value>
    defaultRoute: <insert_value>
    disableRedirect: <insert_value>
    httpSecret: <insert_value>
    logLevel: <insert_value>
    logging: <insert_value>
    managementState: <insert_value>
    nodeSelector: {}
    observedConfig: {}
    operatorLogLevel: <insert_value>
    proxy:
        http: <insert_value>
        https: <insert_value>
        noProxy: <insert_value>
    readOnly: <insert_value>
    replicas: <insert_value>
    requests:
        read:
            maxInQueue: <insert_value>
            maxRunning: <insert_value>
            maxWaitInQueue: <insert_value>
        write:
            maxInQueue: <insert_value>
            maxRunning: <insert_value>
            maxWaitInQueue: <insert_value>
    resources:
        claims:
            - name: <insert_value>
        limits: {}
        requests: {}
    rolloutStrategy: <insert_value>
    routes:
        - hostname: <insert_value>
          name: <insert_value>
          secretName: <insert_value>
    storage:
        azure:
            accountName: <insert_value>
            cloudName: <insert_value>
            container: <insert_value>
            networkAccess:
                internal:
                    networkResourceGroupName: <insert_value>
                    privateEndpointName: <insert_value>
                    subnetName: <insert_value>
                    vnetName: <insert_value>
                type: <insert_value>
        emptyDir: {}
        gcs:
            bucket: <insert_value>
            keyID: <insert_value>
            projectID: <insert_value>
            region: <insert_value>
        ibmcos:
            bucket: <insert_value>
            location: <insert_value>
            resourceGroupName: <insert_value>
            resourceKeyCRN: <insert_value>
            serviceInstanceCRN: <insert_value>
        managementState: <insert_value>
        oss:
            bucket: <insert_value>
            encryption:
                kms:
                    keyID: <insert_value>
                method: <insert_value>
            endpointAccessibility: <insert_value>
            region: <insert_value>
        pvc:
            claim: <insert_value>
        s3:
            bucket: <insert_value>
            cloudFront:
                baseURL: <insert_value>
                duration: <insert_value>
                keypairID: <insert_value>
                privateKey:
                    key: <insert_value>
                    name: <insert_value>
                    optional: <insert_value>
            encrypt: <insert_value>
            keyID: <insert_value>
            region: <insert_value>
            regionEndpoint: <insert_value>
            trustedCA:
                name: <insert_value>
            virtualHostedStyle: <insert_value>
        swift:
            authURL: <insert_value>
            authVersion: <insert_value>
            container: <insert_value>
            domain: <insert_value>
            domainID: <insert_value>
            regionName: <insert_value>
            tenant: <insert_value>
            tenantID: <insert_value>
    tolerations:
        - effect: <insert_value>
          key: <insert_value>
          operator: <insert_value>
          tolerationSeconds: <insert_value>
          value: <insert_value>
    topologySpreadConstraints:
        - labelSelector:
            matchExpressions:
                - key: <insert_value>
                  operator: <insert_value>
                  values:
                    - {}
            matchLabels: {}
          matchLabelKeys:
            - {}
          maxSkew: <insert_value>
          minDomains: <insert_value>
          nodeAffinityPolicy: <insert_value>
          nodeTaintsPolicy: <insert_value>
          topologyKey: <insert_value>
          whenUnsatisfiable: <insert_value>
    unsupportedConfigOverrides: {}
```
