//go:build !verbose
// +build !verbose

package main

// WrapHlml returns the original implementation without any additional debug logging.
func getVerboseHlml(impl Hlml) Hlml {
	return impl
}
