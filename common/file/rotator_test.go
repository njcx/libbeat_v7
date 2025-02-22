// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package file_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/njcx/libbeat_v7/common/file"
	"github.com/njcx/libbeat_v7/logp"
)

const logMessage = "Test file rotator.\n"

func TestFileRotator(t *testing.T) {
	logp.TestingSetup()

	dir := t.TempDir()

	filename := filepath.Join(dir, "sample.log")
	r, err := file.NewFileRotator(filename,
		file.MaxBackups(2),
		file.WithLogger(logp.NewLogger("rotator").With(logp.Namespace("rotator"))),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	WriteMsg(t, r)
	AssertDirContents(t, dir, "sample.log")

	Rotate(t, r)
	AssertDirContents(t, dir, "sample.log.1")

	WriteMsg(t, r)
	AssertDirContents(t, dir, "sample.log", "sample.log.1")

	Rotate(t, r)
	AssertDirContents(t, dir, "sample.log.1", "sample.log.2")

	WriteMsg(t, r)
	AssertDirContents(t, dir, "sample.log", "sample.log.1", "sample.log.2")

	Rotate(t, r)
	AssertDirContents(t, dir, "sample.log.1", "sample.log.2")

	Rotate(t, r)
	AssertDirContents(t, dir, "sample.log.2", "sample.log.3")
}

func TestFileRotatorConcurrently(t *testing.T) {
	dir := t.TempDir()

	filename := filepath.Join(dir, "sample.log")
	r, err := file.NewFileRotator(filename, file.MaxBackups(2))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	var wg sync.WaitGroup
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer wg.Done()
			WriteMsg(t, r)
		}()
	}
	wg.Wait()
}

func TestDailyRotation(t *testing.T) {
	dir := t.TempDir()

	logname := "daily"
	dateFormat := "2006-01-02"
	today := time.Now().Format(dateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(dateFormat)
	twoDaysAgo := time.Now().AddDate(0, 0, -2).Format(dateFormat)

	// seed directory with existing log files
	files := []string{
		logname + "-" + yesterday + "-1",
		logname + "-" + yesterday + "-2",
		logname + "-" + yesterday + "-3",
		logname + "-" + yesterday + "-4",
		logname + "-" + yesterday + "-5",
		logname + "-" + yesterday + "-6",
		logname + "-" + yesterday + "-7",
		logname + "-" + yesterday + "-8",
		logname + "-" + yesterday + "-9",
		logname + "-" + yesterday + "-10",
		logname + "-" + yesterday + "-11",
		logname + "-" + yesterday + "-12",
		logname + "-" + yesterday + "-13",
		logname + "-" + twoDaysAgo + "-1",
		logname + "-" + twoDaysAgo + "-2",
		logname + "-" + twoDaysAgo + "-3",
	}

	for _, f := range files {
		CreateFile(t, filepath.Join(dir, f))
	}

	maxSizeBytes := uint(500)
	filename := filepath.Join(dir, logname)
	r, err := file.NewFileRotator(filename, file.MaxBackups(2), file.Interval(24*time.Hour), file.MaxSizeBytes(maxSizeBytes))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	// The backups exceeding the max of 2 aren't deleted until the first rotation.
	AssertDirContents(t, dir, files...)

	Rotate(t, r)

	AssertDirContents(t, dir, logname+"-"+yesterday+"-12", logname+"-"+yesterday+"-13")

	WriteMsg(t, r)

	AssertDirContents(t, dir, logname+"-"+yesterday+"-12", logname+"-"+yesterday+"-13", logname)

	Rotate(t, r)

	AssertDirContents(t, dir, logname+"-"+yesterday+"-13", logname+"-"+today+"-1")

	WriteMsg(t, r)

	AssertDirContents(t, dir, logname+"-"+yesterday+"-13", logname+"-"+today+"-1", logname)

	for i := 0; i < (int(maxSizeBytes)/len(logMessage))+1; i++ {
		WriteMsg(t, r)
	}

	AssertDirContents(t, dir, logname+"-"+today+"-1", logname+"-"+today+"-2", logname)
}

// Tests the FileConfig.RotateOnStartup parameter
func TestRotateOnStartup(t *testing.T) {
	dir := t.TempDir()

	logname := "rotate_on_open"
	filename := filepath.Join(dir, logname)

	// Create an existing log file with this name.
	CreateFile(t, filename)
	AssertDirContents(t, dir, logname)

	r, err := file.NewFileRotator(filename, file.RotateOnStartup(false))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	WriteMsg(t, r)

	// The line should have been appended to the existing file without rotation.
	AssertDirContents(t, dir, logname)

	// Close the first rotator early (the deferred close will be a no-op if
	// we haven't hit an error by now), so it can't interfere with the second one.
	r.Close()

	// Create a second rotator with the default setting of rotateOnStartup=true
	r, err = file.NewFileRotator(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	// The directory contents shouldn't change until the first Write.
	AssertDirContents(t, dir, logname)

	WriteMsg(t, r)
	AssertDirContents(t, dir, logname, logname+".1")
}

func TestRotateDateSuffix(t *testing.T) {
	dir := t.TempDir()

	logname := "beatname"
	filename := filepath.Join(dir, logname)

	r, err := file.NewFileRotator(filename, file.Suffix(file.SuffixDate), file.MaxBackups(1))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	WriteMsg(t, r)

	firstExpectedPattern := fmt.Sprintf("%s-%s.*", logname, time.Now().Format("20060102150405"))
	AssertDirContentsPattern(t, dir, firstExpectedPattern)

	time.Sleep(2 * time.Second)
	secondExpectedPattern := fmt.Sprintf("%s-%s.*", logname, time.Now().Format("20060102150405"))

	Rotate(t, r)
	WriteMsg(t, r)

	AssertDirContentsPattern(t, dir, firstExpectedPattern, secondExpectedPattern)

	time.Sleep(2 * time.Second)
	thirdExpectedPattern := fmt.Sprintf("%s-%s.*", logname, time.Now().Format("20060102150405"))

	Rotate(t, r)
	WriteMsg(t, r)

	AssertDirContentsPattern(t, dir, secondExpectedPattern, thirdExpectedPattern)
}

func CreateFile(t *testing.T, filename string) {
	t.Helper()
	f, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func AssertDirContents(t *testing.T, dir string, files ...string) {
	t.Helper()

	f, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	names, err := f.Readdirnames(-1)
	if err != nil {
		t.Fatal(err)
	}

	sort.Strings(files)
	sort.Strings(names)
	assert.EqualValues(t, files, names)
}

func AssertDirContentsPattern(t *testing.T, dir string, patterns ...string) {
	t.Helper()

	f, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	names, err := f.Readdirnames(-1)
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != len(names) {
		t.Fatal("unexpected number of files")
	}

	sort.Strings(patterns)
	sort.Strings(names)
	for i := 0; i < len(patterns); i++ {
		matches, err := regexp.MatchString(patterns[i], names[i])
		if err != nil {
			t.Fatal(err)
		}
		assert.True(t, matches, "pattern: %s name: %s", patterns[i], names[i])
	}
}

func WriteMsg(t *testing.T, r *file.Rotator) {
	t.Helper()

	n, err := r.Write([]byte(logMessage))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(logMessage), n)
}

func Rotate(t *testing.T, r *file.Rotator) {
	t.Helper()

	if err := r.Rotate(); err != nil {
		t.Fatal(err)
	}
}
