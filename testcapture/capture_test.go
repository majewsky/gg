// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package testcapture_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/testcapture"
)

func TestCaptureArtifactDir(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		// test that writing a regular file works
		fooPath := filepath.Join(t.ArtifactDir(), "foo.txt")
		err := os.WriteFile(fooPath, []byte("Hello World."), 0666)
		if err != nil {
			t.Fatal(err)
		}

		// test that nesting regular files into directories works
		// (and also, implicitly, that multiple calls to ArtifactDir() return the same path)
		barPath := filepath.Join(t.ArtifactDir(), "bar/a/b/c/d/data.json")
		err = os.MkdirAll(filepath.Dir(barPath), 0777)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(barPath, []byte(`{"bar":42}`), 0666)
		if err != nil {
			t.Fatal(err)
		}

		// test that empty directories are ignored by the capture
		err = os.MkdirAll(filepath.Join(t.ArtifactDir(), "unused"), 0777)
		if err != nil {
			t.Fatal(err)
		}
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeFinished,
		Artifacts: map[string]string{
			"foo.txt":               "Hello World.",
			"bar/a/b/c/d/data.json": `{"bar":42}`,
		},
	})
}

func TestCaptureAttr(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		t.Attr("foo", "bar")
		t.Attr("foo", "baz")
		t.Attr("hello", "world")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeFinished,
		Attrs: map[string]string{
			"foo":   "baz",
			"hello": "world",
		},
	})
}

func TestCaptureChdir(t *testing.T) {
	var cwdInCapture string
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		// ArtifactDir() is used as a target for Chdir()...
		dir := t.ArtifactDir()
		t.Chdir(dir)
		// ...because we have an easy way to check if we are actually in that dir
		err := os.WriteFile("foo.txt", []byte("foo"), 0666)
		if err != nil {
			t.Error(err)
		}

		// smuggle the cwd out of the capture for the cleanup test below
		cwdInCapture, err = os.Getwd()
		if err != nil {
			t.Error(err)
		}

		// t.Chdir() should have set $PWD to an absolute path
		assert.Equal(t, os.Getenv("PWD"), cwdInCapture)
		assert.Equal(t, filepath.IsAbs(cwdInCapture), true)
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome:   testcapture.OutcomeFinished,
		Artifacts: map[string]string{"foo.txt": "foo"},
	})

	// check that we reset the working directory at the end of the test
	cwdAfterCapture, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	if cwdAfterCapture == cwdInCapture {
		t.Error("cwd should have been reset, but still is", cwdInCapture)
	}
}

func TestCaptureCleanup(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		// test that cleanups run in reverse order of registration, and only after the test itself is done
		t.Log("starting up")
		t.Cleanup(func() { t.Log("first cleanup") })
		t.Cleanup(func() { t.Log("second cleanup") })
		t.Cleanup(func() { t.Log("third cleanup") })
		t.Log("shutting down")

		// test that cleanups run even after a panic
		panic("kaboom")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomePanicked,
		Messages: []testcapture.Message{
			testcapture.Log("starting up"),
			testcapture.Log("shutting down"),
			testcapture.Log("third cleanup"),
			testcapture.Log("second cleanup"),
			testcapture.Log("first cleanup"),
		},
		Panic: "kaboom",
	})
}

func TestCaptureContext(t *testing.T) {
	ctx := context.WithValue(t.Context(), "foo", "bar") //nolint:staticcheck // we do not care about type collision risks for this simple test
	result := testcapture.Capture(ctx, t.Name(), func(t assert.TestingTB) {
		// test that the context is live within the test
		err := t.Context().Err()
		if err != nil {
			t.Error(err)
		}

		// test that the context is expired at cleanup time
		t.Cleanup(func() {
			err := t.Context().Err()
			if err == nil {
				t.Error("still alive!?")
			}
		})

		// test that the context is derived from the one passed to Capture()
		value := ctx.Value("foo")
		if value != "bar" {
			t.Error("did not see the value")
		}
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeFinished,
	})
}

func TestCaptureError(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		simpleError := errors.New("bar")
		t.Error("foo: ", simpleError)
		// check that the test keeps going, and that t.Log() does not reset the Outcome
		t.Log("still going")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeFailed,
		Messages: []testcapture.Message{
			testcapture.Log("foo: bar"),
			testcapture.Log("still going"),
		},
	})
}

func TestCaptureErrorf(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		t.Errorf("foo = %d", 42)
		// check that the test keeps going, and that t.Log() does not reset the Outcome
		t.Log("still going")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeFailed,
		Messages: []testcapture.Message{
			testcapture.Log("foo = 42"),
			testcapture.Log("still going"),
		},
	})
}

func TestCaptureFail(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		if !t.Failed() {
			t.Log("looking good so far")
		}
		t.Fail()
		if t.Failed() {
			t.Log("still going")
		}
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeFailed,
		Messages: []testcapture.Message{
			testcapture.Log("looking good so far"),
			testcapture.Log("still going"),
		},
	})
}

func TestCaptureFailNow(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		if !t.Failed() {
			t.Log("looking good so far")
		}
		t.FailNow()
		t.Log("still going")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeFailed,
		Messages: []testcapture.Message{
			testcapture.Log("looking good so far"),
			// "still going" is not logged because FailNow() bails
		},
	})
}

func TestCaptureFatal(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		t.Log("looking good so far")
		t.Fatal("kaboom")
		t.Log("still going")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeFailed,
		Messages: []testcapture.Message{
			testcapture.Log("looking good so far"),
			testcapture.Log("kaboom"),
			// "still going" is not logged because Fatal() bails
		},
	})
}

func TestCaptureFatalf(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		t.Log("looking good so far")
		t.Fatalf("kaboom %d", 42)
		t.Log("still going")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeFailed,
		Messages: []testcapture.Message{
			testcapture.Log("looking good so far"),
			testcapture.Log("kaboom 42"),
			// "still going" is not logged because Fatalf() bails
		},
	})
}

func TestCaptureName(t *testing.T) {
	result := testcapture.Capture(t.Context(), "Harold", func(t assert.TestingTB) {
		panic(t.Name() + " died")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomePanicked,
		Panic:   "Harold died",
	})
}

func TestCaptureOutput(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		t.Log("hello 1")
		for range 10 {
			fmt.Fprintln(t.Output(), "a")
		}
		t.Log("hello 2")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeFinished,
		Messages: []testcapture.Message{
			testcapture.Log("hello 1"),
			testcapture.Output(strings.Repeat("a\n", 10)),
			testcapture.Log("hello 2"),
		},
	})
}

func TestCapturePanic(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		t.Error("we are going to blow up")
		panic("kaboom")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomePanicked, // Panicked takes precedence over Failed
		Messages: []testcapture.Message{
			testcapture.Log("we are going to blow up"),
		},
		Panic: "kaboom",
	})
}

func TestCaptureSetenv(t *testing.T) {
	t.Setenv("GG_TEST_SETENV_SCOPE", "outer")

	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		// test t.Setenv() overriding an existing variable
		assert.Equal(t, os.Getenv("GG_TEST_SETENV_SCOPE"), "outer")
		t.Setenv("GG_TEST_SETENV_SCOPE", "inner")
		assert.Equal(t, os.Getenv("GG_TEST_SETENV_SCOPE"), "inner")

		// test t.Setenv() setting a fresh variable
		// (this variable should be completely removed after the end of the test)
		t.Setenv("GG_TEST_SETENV_PAYLOAD", "42")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeFinished,
	})

	assert.Equal(t, os.Getenv("GG_TEST_SETENV_SCOPE"), "outer")
	_, ok := os.LookupEnv("GG_TEST_SETENV_PAYLOAD")
	assert.Equal(t, ok, false)
}

func TestCaptureSkip(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		if !t.Skipped() {
			t.Skip("this looks uninteresting")
		}
		t.Log("still going")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeSkipped,
		Messages: []testcapture.Message{
			testcapture.Log("this looks uninteresting"),
			// "still going" is not logged because Skip() bails
		},
	})
}

func TestCaptureSkipf(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		if !t.Skipped() {
			t.Skipf("pretty sure the answer is %d", 42)
		}
		t.Log("still going")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeSkipped,
		Messages: []testcapture.Message{
			testcapture.Log("pretty sure the answer is 42"),
			// "still going" is not logged because Skipf() bails
		},
	})
}

func TestCaptureSkipNow(t *testing.T) {
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		if !t.Skipped() {
			t.Log("this looks uninteresting")
		}
		t.SkipNow()
		t.Log("still going")
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeSkipped,
		Messages: []testcapture.Message{
			testcapture.Log("this looks uninteresting"),
			// "still going" is not logged because Skip() bails
		},
	})
}

func TestCaptureTempDir(t *testing.T) {
	var path string
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		// fill a TempDir with some stuff
		path = t.TempDir()
		err := os.MkdirAll(filepath.Join(path, "emptydir"), 0777)
		if err != nil {
			t.Error(err)
		}
		err = os.WriteFile(filepath.Join(path, "data.json"), []byte(`{"username":"admin"}`), 0666)
		if err != nil {
			t.Error(err)
		}

		// check that each call to TempDir returns a new dir
		otherPath := t.TempDir()
		if otherPath == path {
			t.Error("should have returned a different TempDir")
		}
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomeFinished,
	})

	// check that TempDir was cleaned up
	_, err := os.Stat(path)
	if err == nil {
		t.Error("TempDir was not cleaned up")
	} else if !os.IsNotExist(err) {
		t.Error(err)
	}
}
