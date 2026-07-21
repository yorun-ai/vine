package skel

import internalskel "go.yorun.ai/vine/internal/core/skel"

// MinSkelcVersion returns the minimum skelc version required to generate code
// compatible with this Vine skel runtime.
func MinSkelcVersion() string {
	return internalskel.MinSkelcVersion()
}
