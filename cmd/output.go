package cmd

import (
	"fmt"

	"github.com/yaacov/kubectl-debug-queries/pkg/kube"
)

// outputResult prints the result to stdout. When format is json and an error
// occurred, it prints a JSON error object to stdout so the output stream
// remains valid JSON, then returns the original error for a non-zero exit code.
func outputResult(result string, err error, format string) error {
	if err != nil {
		if kube.IsJSONFormat(format) {
			fmt.Println(kube.JSONError(err))
		}
		return err
	}

	if result == "" && kube.IsJSONFormat(format) {
		fmt.Println(kube.JSONEmpty)
		return nil
	}

	fmt.Println(result)
	return nil
}
