package skel

const minSkelcVersion = "v0.17.1"

// MinSkelcVersion returns the minimum skelc version required to generate code
// compatible with this Vine skel runtime.
func MinSkelcVersion() string {
	return minSkelcVersion
}
