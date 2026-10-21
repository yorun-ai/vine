// Package web serves Vine routes, embedded assets, and reverse proxies.
//
// Use ServeReverseProxy for custom Gin proxies to distinguish upstream status
// codes from local transport failures. Existing raw ReverseProxy handlers retain
// status-based logging; no HTTP header can mark a response as upstream-owned.
package web
