package utils

import (
	"sort"

	"project-template/pkg/code"
)

// PrecomputeErrorCodes returns a sorted list of error code names referenced by
// the given function in the specified package. It reuses the same AST-based
// inference used at runtime so the compile-time generator does not need to
// duplicate that logic.
func PrecomputeErrorCodes(pkgPath, fnName string) []string {
	codes := map[string]*code.Code{}
	inferFrom(pkgPath, fnName, map[string]bool{}, codes)
	names := make([]string, 0, len(codes))
	for n := range codes {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
