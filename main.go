package main

import (
    "flag"
    "fmt"
    "os"
    "os/exec"

    "gopkg.in/yaml.v3"
)

func getCRDYaml(crdName string) (string, error) {
    cmd := exec.Command("oc", "get", "crd", crdName, "-o", "yaml")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("error getting CRD: %s", string(output))
    }
    return string(output), nil
}

func generateSpec(schema map[string]interface{}) map[string]interface{} {
    spec := make(map[string]interface{})
    properties, ok := schema["properties"].(map[string]interface{})
    if !ok {
        return spec
    }

    for key, value := range properties {
        vMap, ok := value.(map[string]interface{})
        if !ok {
            continue
        }
        typ, ok := vMap["type"].(string)
        if !ok {
            continue
        }
        switch typ {
        case "object":
            spec[key] = generateSpec(vMap)
        case "array":
            items, ok := vMap["items"].(map[string]interface{})
            if ok {
                spec[key] = []interface{}{generateSpec(items)}
            }
        default:
            spec[key] = "<insert_value>"
        }
    }
    return spec
}

func main() {
    crdName := flag.String("crd", "", "The name of the CustomResourceDefinition")
    flag.Parse()

    if *crdName == "" {
        fmt.Println("Error: CRD name is required")
        os.Exit(1)
    }

    crdYaml, err := getCRDYaml(*crdName)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    var crd map[string]interface{}
    err = yaml.Unmarshal([]byte(crdYaml), &crd)
    if err != nil {
        fmt.Printf("Error parsing YAML: %v\n", err)
        os.Exit(1)
    }

    schema, ok := crd["spec"].(map[string]interface{})["versions"].([]interface{})[0].(map[string]interface{})["schema"].(map[string]interface{})["openAPIV3Schema"].(map[string]interface{})
    if !ok {
        fmt.Println("Error: Cannot find OpenAPI schema in CRD")
        os.Exit(1)
    }

    spec := generateSpec(schema["properties"].(map[string]interface{})["spec"].(map[string]interface{}))
    specYaml, err := yaml.Marshal(map[string]interface{}{"spec": spec})
    if err != nil {
        fmt.Printf("Error generating YAML: %v\n", err)
        os.Exit(1)
    }

    fmt.Println(string(specYaml))
}
