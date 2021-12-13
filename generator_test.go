package generator

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	parser "github.com/go-generalize/go-easyparser"
	"github.com/go-generalize/go-easyparser/types"
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

func parseAndGenerate(t *testing.T, dir string, ei func(o *types.Object) *ExternalImporter) {
	t.Helper()

	psr, err := parser.NewParser(dir, parser.All)
	if err != nil {
		t.Fatal(err)
	}

	res, err := psr.Parse()

	if err != nil {
		t.Fatal(err)
	}
	gen := NewGenerator(res, []string{})
	gen.ExternalImporter = ei

	b, err := gen.Generate()

	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "/gen.dart"), []byte(b), 0744); err != nil {
		t.Fatal(err)
	}
}

func testWithDatasets(t *testing.T, dir, structName string, ei func(o *types.Object) *ExternalImporter) {
	parseAndGenerate(t, dir, ei)

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

func getStructName(path string) string {
	split := strings.Split(path, ".")

	return split[len(split)-1]
}

func TestGenerator_Generate(t *testing.T) {
	wd, err := os.Getwd()

	if err != nil {
		t.Fatal(err)
	}

	testWithDatasets(t, "testfiles/standard", "PostUserRequest", nil)

	testWithDatasets(t, "testfiles/external", "Struct", func(o *types.Object) *ExternalImporter {
		rel, err := filepath.Rel(filepath.Join(wd, "testfiles/external"), o.Position.Filename)

		if err != nil {
			t.Fatal(err)
		}

		if strings.HasPrefix(rel, "../") || filepath.Dir(rel) == "." {
			return nil
		}

		return &ExternalImporter{
			Path: filepath.Dir(rel) + "/gen.dart",
			Name: getStructName(o.Name),
		}
	})
}
