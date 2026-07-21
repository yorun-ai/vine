package vnet

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustParsePort(t *testing.T) {
	assert.Equal(t, 7079, MustParsePort(":7079"))
}

func TestMustParseHost(t *testing.T) {
	assert.Equal(t, "0.0.0.0", MustParseHost("0.0.0.0:7079"))
}

func ExampleParseHost() {
	host, err := ParseHost("127.0.0.1:7079")
	fmt.Println(host, err == nil)
	// Output:
	// 127.0.0.1 true
}

func ExampleParsePort() {
	port, err := ParsePort("127.0.0.1:7079")
	fmt.Println(port, err == nil)
	// Output:
	// 7079 true
}

func ExampleMustParseHost() {
	fmt.Println(MustParseHost("0.0.0.0:7079"))
	// Output:
	// 0.0.0.0
}

func ExampleMustParsePort() {
	fmt.Println(MustParsePort(":7079"))
	// Output:
	// 7079
}
