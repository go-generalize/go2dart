package generator

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-generalize/go2ts/pkg/parser"
	"github.com/google/go-cmp/cmp"
)

const runnerDart = `import 'dart:convert';
import 'dart:io';
import 'gen.dart';

void main(List<String> arguments) async {
  final input = await stdin.transform(utf8.decoder).join('');

  print(jsonEncode(%s.fromJson(jsonDecode(input)).toJson()));
}
`

func parseJson(t *testing.T, v string) map[string]interface{} {
	t.Helper()
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(v), &m); err != nil {
		t.Fatal(err)
	}

	return m
}

func testWithDatasets(t *testing.T, dir, structName string) {
	psr, err := parser.NewParser(dir, parser.All)
	if err != nil {
		t.Fatal(err)
	}

	res, err := psr.Parse()

	if err != nil {
		t.Fatal(err)
	}
	gen := NewGenerator(res, []string{})
	b, err := gen.Generate()

	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "/gen.dart"), []byte(b), 0744); err != nil {
		t.Fatal(err)
	}

	runnerDart := fmt.Sprintf(
		runnerDart,
		structName,
	)

	runnerDartPath := filepath.Join(dir, "/runner.dart")

	if err := os.WriteFile(runnerDartPath, []byte(runnerDart), 0744); err != nil {
		t.Fatal(err)
	}

	datasetsDir := filepath.Join(dir, "datasets")

	des, err := os.ReadDir(datasetsDir)

	if err != nil {
		t.Fatal(err)
	}

	for _, d := range des {
		t.Run(d.Name(), func(t *testing.T) {
			fp, err := os.Open(filepath.Join(datasetsDir, d.Name(), "input.json"))

			if err != nil {
				t.Fatal(err)
			}
			defer fp.Close()

			cmd := exec.Command("dart", "run", runnerDartPath)
			cmd.Stdin = fp
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatal(err, string(output))
			}

			expectedBytes, err := os.ReadFile(filepath.Join(datasetsDir, d.Name(), "output.json"))
			if err != nil {
				t.Fatal(err)
			}

			t.Log("output", string(output))

			expectedJson := parseJson(t, string(expectedBytes))
			outputJson := parseJson(t, string(output))

			if diff := cmp.Diff(expectedJson, outputJson); diff != "" {
				t.Errorf("diff: %s", diff)
			}
		})
	}
}

func TestGenerator_Generate(t *testing.T) {
	testWithDatasets(t, "testfiles/standard", "PostUserRequest")
}
