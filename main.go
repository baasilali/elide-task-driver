// Copyright (c) Elide Dev, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/elide-dev/elide-task-driver/driver"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/plugins"
)

func main() {
	// Serve the plugin
	plugins.Serve(factory)
}

// factory returns a new instance of the Elide task driver plugin
func factory(log hclog.Logger) interface{} {
	return driver.NewPlugin(log)
}

