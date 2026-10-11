// Package skel registers Skel runtime descriptors and checks generator compatibility.
// Deprecated scalar aliases, actor marker types, schema types and
// RegisterDomainSchema remain for previously generated code.
// New contracts use go.yorun.ai/skel/types and
// go.yorun.ai/skel/descriptor with RegisterDomainDescriptor.
// Vine preserves Via metadata emitted by skelc for compatibility but ignores it
// when validating descriptors, selecting services, and admitting requests.
package skel
