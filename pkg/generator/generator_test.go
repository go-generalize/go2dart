package generator

import (
	"os"
	"testing"

	"github.com/go-generalize/go2ts/pkg/parser"
)

func TestGenerate(t *testing.T) {
	// filter := func(opt *parser.FilterOpt) bool {
	// 	if opt.Dependency {
	// 		return true
	// 	}
	// 	if !opt.BasePackage {
	// 		return false
	// 	}
	// 	if !opt.Exported {
	// 		return false
	// 	}

	// 	return strings.HasSuffix(opt.Name, "Request") || strings.HasSuffix(opt.Name, "Response")
	// }

	psr, err := parser.NewParser("./testdata/standard", parser.All)

	if err != nil {
		t.Fatal(err)
	}

	res, err := psr.Parse()

	if err != nil {
		t.Fatal(err)
	}
	gen := NewGenerator(res, []string{}, "a.dart")
	b, err := gen.Generate()

	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("./testdata/standard/gen.dart", []byte(b), 0744); err != nil {
		t.Fatal(err)
	}
}
