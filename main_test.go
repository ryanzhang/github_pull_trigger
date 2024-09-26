package main

import (
    "os"
    "os/exec"
    "testing"
)

func TestCrdName(t *testing.T) {
    cmd := exec.Command("./bin/cr_viewer", "configs.imageregistry.operator.openshift.io")
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Errorf("Error running cr_viewer: %v", err)
    }
    if len(output) == 0 {
        t.Errorf("Expected output, but got none")
    }
}

func TestCrdFile(t *testing.T) {
    cmd := exec.Command("./bin/cr_viewer", "-f", "./test-crd.yaml")
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Errorf("Error running cr_viewer: %v", err)
    }
    if len(output) == 0 {
        t.Errorf("Expected output, but got none")
    }
}

func TestMain(m *testing.M) {
    os.Mkdir("bin", 0755)
    exec.Command("go", "build", "-o", "./bin/cr_viewer", "main.go").Run()
    code := m.Run()
    os.RemoveAll("bin")
    os.Exit(code)
}
