package testing_test

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/influxdata/flux"
	"github.com/influxdata/flux/ast"
	"github.com/influxdata/flux/execute"
	"github.com/influxdata/flux/lang"
	"github.com/influxdata/flux/parser"
	"github.com/influxdata/flux/stdlib"

	platform "github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/cmd/influxd/launcher"
	"github.com/influxdata/influxdb/query"

	_ "github.com/influxdata/flux/stdlib"           // Import the built-in functions
	_ "github.com/influxdata/influxdb/query/stdlib" // Import the stdlib
)

// Default context.
var ctx = context.Background()

func init() {
	flux.FinalizeBuiltIns()
}

var skipTests = map[string]string{
	// TODO(adam) determine the reason for these test failures.
	"cov":                      "Reason TBD",
	"covariance":               "Reason TBD",
	"cumulative_sum":           "Reason TBD",
	"cumulative_sum_default":   "Reason TBD",
	"cumulative_sum_noop":      "Reason TBD",
	"difference_panic":         "Reason TBD",
	"drop_non_existent":        "Reason TBD",
	"filter_by_regex_function": "Reason TBD",
	"first":                    "Reason TBD",
	"group_by_irregular":       "Reason TBD",
	"highestAverage":           "Reason TBD",
	"highestMax":               "Reason TBD",
	"histogram":                "Reason TBD",
	"histogram_normalize":      "Reason TBD",
	"histogram_quantile":       "Reason TBD",
	"join":                     "Reason TBD",
	"join_across_measurements": "Reason TBD",
	"keep_non_existent":        "Reason TBD",
	"key_values":               "Reason TBD",
	"key_values_host_name":     "Reason TBD",
	"last":                     "Reason TBD",
	"lowestAverage":            "Reason TBD",
	"max":                      "Reason TBD",
	"meta_query_fields":        "Reason TBD",
	"meta_query_keys":          "Reason TBD",
	"meta_query_measurements":  "Reason TBD",
	"min":                      "Reason TBD",
	"multiple_range":           "Reason TBD",
	"sample":                   "Reason TBD",
	"selector_preserve_time":   "Reason TBD",
	"shift":                    "Reason TBD",
	"shift_negative_duration":  "Reason TBD",
	"show_all_tag_keys":        "Reason TBD",
	"sort":                     "Reason TBD",
	"task_per_line":            "Reason TBD",
	"top":                      "Reason TBD",
	"union":                    "Reason TBD",
	"union_heterogeneous":      "Reason TBD",
	"unique":                   "Reason TBD",
	"distinct":                 "Reason TBD",

	// it appears these occur when writing the input data.  `to` may not be null safe.
	"fill_bool":   "failed to read meta data: panic: interface conversion: interface {} is nil, not uint64",
	"fill_float":  "failed to read meta data: panic: interface conversion: interface {} is nil, not uint64",
	"fill_int":    "failed to read meta data: panic: interface conversion: interface {} is nil, not uint64",
	"fill_string": "failed to read meta data: panic: interface conversion: interface {} is nil, not uint64",
	"fill_time":   "failed to read meta data: panic: interface conversion: interface {} is nil, not uint64",
	"fill_uint":   "failed to read meta data: panic: interface conversion: interface {} is nil, not uint64",
	"window_null": "failed to read meta data: panic: interface conversion: interface {} is nil, not float64",

	// these may just be missing calls to range() in the tests.  easy to fix in a new PR.
	"group_nulls":      "unbounded test",
	"integral":         "unbounded test",
	"integral_columns": "unbounded test",
	"map":              "unbounded test",

	// the following tests have a difference between the CSV-decoded input table, and the storage-retrieved version of that table
	"columns":              "group key mismatch",
	"count":                "column order mismatch",
	"mean":                 "column order mismatch",
	"percentile_aggregate": "column order mismatch",
	"percentile_tdigest":   "column order mismatch",
	"set":                  "column order mismatch",
	"set_new_column":       "column order mismatch",
	"skew":                 "column order mismatch",
	"spread":               "column order mismatch",
	"stddev":               "column order mismatch",
	"sum":                  "column order mismatch",
	"simple_max":           "_stop missing from expected output",
	"derivative":           "time bounds mismatch (engine uses now() instead of bounds on input table)",
	"percentile":           "time bounds mismatch (engine uses now() instead of bounds on input table)",
	"difference_columns":   "data write/read path loses columns x and y",
	"keys":                 "group key mismatch",
	"pivot_task_test":      "possible group key or column order mismatch",

	// failed to read meta data errors: the CSV encoding is incomplete probably due to data schema errors.  needs more detailed investigation to find root cause of error
	"filter_by_regex":             "failed to read metadata",
	"filter_by_tags":              "failed to read metadata",
	"group":                       "failed to read metadata",
	"group_except":                "failed to read metadata",
	"group_ungroup":               "failed to read metadata",
	"pivot_mean":                  "failed to read metadata",
	"select_measurement":          "failed to read metadata",
	"select_measurement_field":    "failed to read metadata",
	"histogram_quantile_minvalue": "failed to read meta data: no column with label _measurement exists",
	"increase":                    "failed to read meta data: table has no _value column",

	"string_max":                  "error: invalid use of function: *functions.MaxSelector has no implementation for type string (https://github.com/influxdata/platform/issues/224)",
	"null_as_value":               "null not supported as value in influxql (https://github.com/influxdata/platform/issues/353)",
	"string_interp":               "string interpolation not working as expected in flux (https://github.com/influxdata/platform/issues/404)",
	"to":                          "to functions are not supported in the testing framework (https://github.com/influxdata/flux/issues/77)",
	"covariance_missing_column_1": "need to support known errors in new test framework (https://github.com/influxdata/flux/issues/536)",
	"covariance_missing_column_2": "need to support known errors in new test framework (https://github.com/influxdata/flux/issues/536)",
	"drop_before_rename":          "need to support known errors in new test framework (https://github.com/influxdata/flux/issues/536)",
	"drop_referenced":             "need to support known errors in new test framework (https://github.com/influxdata/flux/issues/536)",
	"yield":                       "yield requires special test case (https://github.com/influxdata/flux/issues/535)",
	"rowfn_with_import":           "imported libraries are not visible in user-defined functions (https://github.com/influxdata/flux/issues/1000)",
	"string_trim":                 "imported libraries are not visible in user-defined functions (https://github.com/influxdata/flux/issues/1000)",

	"window_group_mean_ungroup": "window trigger optimization modifies sort order of its output tables (https://github.com/influxdata/flux/issues/1067)",
}

func TestFluxEndToEnd(t *testing.T) {
	runEndToEnd(t, stdlib.FluxTestPackages)
}
func BenchmarkFluxEndToEnd(b *testing.B) {
	benchEndToEnd(b, stdlib.FluxTestPackages)
}

func runEndToEnd(t *testing.T, pkgs []*ast.Package) {
	l := launcher.RunTestLauncherOrFail(t, ctx)
	l.SetupOrFail(t)
	defer l.ShutdownOrFail(t, ctx)
	for _, pkg := range pkgs {
		pkg := pkg.Copy().(*ast.Package)
		name := pkg.Files[0].Name
		t.Run(name, func(t *testing.T) {
			if reason, ok := skipTests[strings.TrimSuffix(name, ".flux")]; ok {
				t.Skip(reason)
			}
			testFlux(t, l, pkg)
		})
	}
}

func benchEndToEnd(b *testing.B, pkgs []*ast.Package) {
	l := launcher.RunTestLauncherOrFail(b, ctx)
	l.SetupOrFail(b)
	defer l.ShutdownOrFail(b, ctx)
	for _, pkg := range pkgs {
		pkg := pkg.Copy().(*ast.Package)
		name := pkg.Files[0].Name
		b.Run(name, func(b *testing.B) {
			if reason, ok := skipTests[strings.TrimSuffix(name, ".flux")]; ok {
				b.Skip(reason)
			}
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				testFlux(b, l, pkg)
			}
		})
	}
}

var (
	optionsAST      *ast.File
	getInputDataAST *ast.File
)

const (
	optionsSource = `
import "testing"
import c "csv"

// Options bucket and org are defined dynamically per test

option testing.loadStorage = (csv) => {
	c.from(csv: csv) |> to(bucket: bucket, org: org)
	return from(bucket: bucket)
}
`
	// TODO(affo): due to flaky tests, we ensure that writes were successful before proceeding.
	//  So, before checking test errors, we validate that input data written to its bucket
	//  was successfully read back, in order to ensure that the test result is reliable.
	//  These and all related lines can be removed once https://github.com/influxdata/influxdb/issues/12891 gets fixed.
	inputDataResultName = "input-data"
	getInputDataSource  = `from(bucket: bucket) |> range(start: 1970-01-01T00:00:00Z) |> yield(name: "` + inputDataResultName + `")`
	flakyReason         = "flaky test (https://github.com/influxdata/influxdb/issues/12891)"
)

func init() {
	pkg := parser.ParseSource(optionsSource)
	if ast.Check(pkg) > 0 {
		panic(ast.GetError(pkg))
	}
	optionsAST = pkg.Files[0]
	pkg = parser.ParseSource(getInputDataSource)
	if ast.Check(pkg) > 0 {
		panic(ast.GetError(pkg))
	}
	getInputDataAST = pkg.Files[0]
}

func testFlux(t testing.TB, l *launcher.TestLauncher, pkg *ast.Package) {

	// Query server to ensure write persists.

	b := &platform.Bucket{
		OrgID:           l.Org.ID,
		Name:            t.Name(),
		RetentionPeriod: 0,
	}

	s := l.BucketService()
	if err := s.CreateBucket(context.Background(), b); err != nil {
		t.Fatal(err)
	}

	// Define bucket and org options
	bucketOpt := &ast.OptionStatement{
		Assignment: &ast.VariableAssignment{
			ID:   &ast.Identifier{Name: "bucket"},
			Init: &ast.StringLiteral{Value: b.Name},
		},
	}
	orgOpt := &ast.OptionStatement{
		Assignment: &ast.VariableAssignment{
			ID:   &ast.Identifier{Name: "org"},
			Init: &ast.StringLiteral{Value: l.Org.Name},
		},
	}
	options := optionsAST.Copy().(*ast.File)
	options.Body = append([]ast.Statement{bucketOpt, orgOpt}, options.Body...)

	// Add options to pkg
	pkg.Files = append(pkg.Files, options)

	// Add testing.inspect call to ensure the data is loaded
	inspectCalls := stdlib.TestingInspectCalls(pkg)
	pkg.Files = append(pkg.Files, inspectCalls)

	req := &query.Request{
		OrganizationID: l.Org.ID,
		Compiler:       lang.ASTCompiler{AST: pkg},
	}
	if r, err := l.FluxQueryService().Query(ctx, req); err != nil {
		t.Fatal(err)
	} else {
		defer r.Release()
		for r.More() {
			v := r.Next()
			if err := v.Tables().Do(func(tbl flux.Table) error {
				return nil
			}); err != nil {
				t.Error(err)
			}
		}
		if r.Err() != nil {
			t.Fatal(r.Err())
		}
	}

	// quirk: our execution engine doesn't guarantee the order of execution for disconnected DAGS
	// so that our function-with-side effects call to `to` may run _after_ the test instead of before.
	// running twice makes sure that `to` happens at least once before we run the test.
	// this time we use a call to `run` so that the assertion error is triggered
	runCalls := stdlib.TestingRunCalls(pkg)
	pkg.Files[len(pkg.Files)-1] = runCalls
	pkg.Files = append(pkg.Files, getInputDataAST)
	r, err := l.FluxQueryService().Query(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()

	for r.More() {
		v := r.Next()
		if v.Name() == inputDataResultName {
			if countTables(v) == 0 {
				t.Logf("test %s result is not reliable: no table returned from its bucket after writing to it", t.Name())
				t.Skip(flakyReason)
			}
		} else if doErr := v.Tables().Do(func(tbl flux.Table) error {
			return nil
		}); err == nil {
			// Keep only the first error encountered, do not fail.
			// We must iterate over every result in order to understand if the test is reliable or not.
			err = doErr
		}
	}
	if err != nil {
		t.Error(err)
	}

	if err := r.Err(); err != nil {
		t.Error(err)
		// Replace the testing.run calls with testing.inspect calls.
		pkg.Files[len(pkg.Files)-1] = inspectCalls
		r, err := l.FluxQueryService().Query(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		defer func() {
			if t.Failed() {
				scanner := bufio.NewScanner(&out)
				for scanner.Scan() {
					t.Log(scanner.Text())
				}
			}
		}()
		for r.More() {
			v := r.Next()
			err := execute.FormatResult(&out, v)
			if err != nil {
				t.Error(err)
			}
		}
		if err := r.Err(); err != nil {
			t.Error(err)
		}
	}
}

func countTables(result flux.Result) int {
	var count int
	_ = result.Tables().Do(func(tbl flux.Table) error {
		count++
		return nil
	})
	return count
}
