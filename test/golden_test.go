package golden_test

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/aep-dev/aepc/cmd"
	"github.com/google/go-cmp/cmp"
)

const (
	goldenDir = "examples"
)

// TestGolden output relative to input
func TestGolden(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := path.Join(currentDir, "..", goldenDir)
	files, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".output.proto") {
			continue
		}
		inputFile := path.Join(root, f.Name())
		outputFile := path.Join(root, f.Name()+".output.proto")
		t.Run(f.Name(), func(t *testing.T) {
			testOutputMatchesGolden(t, inputFile, outputFile)
		})
	}
}

// testOutputMatchesGolden:
// 1. reads in a file located at inputFile.
// 2. generated a random path randomPath
// 2. runs cmd.ProcessInput(inputFile, randomPath)
func testOutputMatchesGolden(t *testing.T, inputFile, outputFile string) {
	tempDir := os.TempDir()
	randomFile := path.Join(tempDir, "output.proto")
	err := cmd.ProcessInput(inputFile, randomFile)
	if err != nil {
		t.Fatalf("unable to process input: %v", err)
	}
	outputFileBytes, err := cmd.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("unable to open file: %v", err)
	}
	randomFileBytes, err := cmd.ReadFile(randomFile)
	if err != nil {
		t.Fatalf("unable to open file: %v", err)
	}
	if diff := cmp.Diff(string(outputFileBytes), string(randomFileBytes)); diff != "" {
		t.Errorf("ProcessInput() mismatch (-want +got):\n%s", diff)
	}
}
