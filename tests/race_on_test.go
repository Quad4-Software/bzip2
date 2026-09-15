// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

//go:build race

package bzip2_test

// raceDetectorEnabled lets the multi-megabyte sequential oracle tests shrink
// their inputs under the race detector, whose instrumentation makes the
// suffix-sort-heavy encoder many times slower. Full sizes still run in the
// normal test suite.
const raceDetectorEnabled = true
