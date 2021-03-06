// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tfimport

import (
	"fmt"
	"os/exec"
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/healthcare-data-protection-suite/internal/terraform"
	"github.com/GoogleCloudPlatform/healthcare-data-protection-suite/internal/tfimport/importer"
	"github.com/google/go-cmp/cmp"
)

func TestImportable(t *testing.T) {
	tests := []struct {
		rc   terraform.ResourceChange
		pcv  map[string]interface{}
		want resourceImporter
	}{
		// Empty Kind - should return nil.
		{terraform.ResourceChange{}, nil, nil},

		// Unsupported Kind - should return nil.
		{
			terraform.ResourceChange{
				Kind: "unsupported",
			}, nil, nil,
		},

		// Bucket - should return resource with bucket importer
		{
			terraform.ResourceChange{
				Kind:    "google_storage_bucket",
				Address: "google_storage_bucket.gcs_tf_bucket",
				Change: terraform.Change{
					After: map[string]interface{}{
						"project": "project-from-resource",
						"name":    "mybucket",
					},
				},
			}, nil,
			&importer.StorageBucket{},
		},

		// GKE Cluster - should return resource with GKE clsuter importer
		{
			terraform.ResourceChange{
				Kind:    "google_container_cluster",
				Address: "google_container_cluster.my_cluster",
				Change: terraform.Change{
					After: map[string]interface{}{
						"project":  "project-from-resource",
						"location": "us-east1",
						"name":     "mybucket",
					},
				},
			}, nil,
			&importer.GKECluster{},
		},
	}
	for _, tc := range tests {
		got, ok := Importable(tc.rc, tc.pcv)

		// If we want nil, we should get nil.
		// If we don't want nil, then the address and importer should match.
		if got == nil {
			if tc.want != nil {
				t.Errorf("Importable(%v, %v) = nil; want %+v", tc.rc, tc.pcv, tc.want)
			}
		} else if reflect.TypeOf(got.Importer) != reflect.TypeOf(tc.want) {
			t.Errorf("Importable(%v, %v) = %+v; want %+v", tc.rc, tc.pcv, got.Importer, tc.want)
		} else if !ok {
			t.Errorf("Importable(%v, %v) unexpectedly failed", tc.rc, tc.pcv)
		}
	}
}

const (
	testAddress       = "test-address"
	testImportID      = "test-import-id"
	testInputDir      = "test-input-dir"
	testTerraformPath = "terraform"
)

var argsWant = []string{testTerraformPath, "import", testAddress, testImportID}

type testImporter struct{}

func (r *testImporter) ImportID(terraform.ResourceChange, importer.ProviderConfigMap, bool) (string, error) {
	return testImportID, nil
}

type testRunner struct {
	// This can be modified per test case to check different outcomes.
	output []byte
}

func (*testRunner) CmdRun(cmd *exec.Cmd) error              { return nil }
func (*testRunner) CmdOutput(cmd *exec.Cmd) ([]byte, error) { return nil, nil }
func (tr *testRunner) CmdCombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	if !cmp.Equal(cmd.Args, argsWant) {
		return nil, fmt.Errorf("args = %v; want %v", cmd.Args, argsWant)
	}
	return tr.output, nil
}

func TestImportArgs(t *testing.T) {
	testResource := &Resource{
		Change:         terraform.ResourceChange{Address: testAddress},
		ProviderConfig: importer.ProviderConfigMap{},
		Importer:       &testImporter{},
	}

	wantOutput := ""
	trn := &testRunner{
		output: []byte(wantOutput),
	}

	gotOutput, err := Import(trn, testResource, testInputDir, testTerraformPath, true)

	if err != nil {
		t.Errorf("TestImport(%v, %v, %v) %v", trn, testResource, testInputDir, err)
	}
	if !cmp.Equal(gotOutput, wantOutput) {
		t.Errorf("TestImport(%v, %v, %v) output = %v; want %v", trn, testResource, testInputDir, gotOutput, wantOutput)
	}
}

func TestNotImportable(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		// No output.
		{
			output: "",
			want:   false,
		},

		// Not importable error.
		{
			output: "Error: resource google_container_registry doesn't support import",
			want:   true,
		},

		// Importable and exists.
		{
			output: "Import successful!",
			want:   false,
		},
	}
	for _, tc := range tests {
		got := NotImportable(tc.output)
		if got != tc.want {
			t.Errorf("TestNotImportable(%v) = %v; want %v", tc.output, got, tc.want)
		}
	}
}

func TestDoesNotExist(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		// No output.
		{
			output: "",
			want:   false,
		},

		// Does not exist error.
		{
			output: "Error: Cannot import non-existent remote object",
			want:   true,
		},

		// Importable and exists.
		{
			output: "Import successful!",
			want:   false,
		},
	}
	for _, tc := range tests {
		got := DoesNotExist(tc.output)
		if got != tc.want {
			t.Errorf("TestDoesNotExist(%v) = %v; want %v", tc.output, got, tc.want)
		}
	}
}
