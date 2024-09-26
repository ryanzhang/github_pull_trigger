package main

import (
    "flag"
    "fmt"
    "os"
	"io/ioutil"
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

// Main function to handle both modes: fetching from `oc` or reading from a file.
func main() {
	// Define the `-f` flag for file input.
	filePath := flag.String("f", "", "Path to the CRD YAML file")
	flag.Parse()

	var crdYaml string
	var err error

	// Handle CRD from file or oc based on arguments.
	if *filePath != "" {
		// File mode: Load CRD from file.
		data, err := ioutil.ReadFile(*filePath)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}
		crdYaml = string(data)
	} else if len(flag.Args()) > 0 {
		// CRD name mode: Fetch CRD using oc.
		crdName := flag.Args()[0]
		crdYaml, err = getCRDYaml(crdName)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		// Error if neither argument is provided.
		fmt.Println(`Usage: cr_viewer <crd_name> or cr_viewer -f <crd.yaml>
Examples:
  cr_viewer configs.imageregistry.operator.openshift.io
  cr_viewer -f test-crd.yaml`)
		os.Exit(1)
	}

	// Parse the CRD YAML.
	var crd map[string]interface{}
	err = yaml.Unmarshal([]byte(crdYaml), &crd)
	if err != nil {
		fmt.Printf("Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	// Navigate to the OpenAPI schema.
	schema, ok := crd["spec"].(map[string]interface{})["versions"].([]interface{})[0].(map[string]interface{})["schema"].(map[string]interface{})["openAPIV3Schema"].(map[string]interface{})
	if !ok {
		fmt.Println("Error: Cannot find OpenAPI schema in CRD")
		os.Exit(1)
	}

	// Generate the spec from the OpenAPI schema.
	spec := generateSpec(schema["properties"].(map[string]interface{})["spec"].(map[string]interface{}))

	// Output the generated spec as YAML.
	specYaml, err := yaml.Marshal(map[string]interface{}{"spec": spec})
	if err != nil {
		fmt.Printf("Error generating YAML: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(specYaml))
}
