// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// Package zoom is a blazing-fast datastore and querying engine for
// Go built on Redis. It supports models of any arbitrary struct
// type and provides basic querying functionality. It also supports
// atomic transactions, lua scripts, and running Redis commands
// directly if needed.
package zoom

// Init starts the Zoom library and creates a connection pool. It accepts
// a Configuration struct as an argument. Any zero values in the configuration
// will fallback to their default values. Init should be called once during
// application startup.
func Init(config *Configuration) error {
	config = parseConfig(config)
	initPool(config.Network, config.Address, config.Database, config.Password)
	if err := initScripts(config.ScriptsPath); err != nil {
		return err
	}
	return nil
}

// Close closes the connection pool and shuts down the Zoom library.
// It should be run when application exits, e.g. using defer.
func Close() error {
	return pool.Close()
}
