// Copyright 2024 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaferWriteFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file")
	want := "test-data"

	if err := SaferWriteFile([]byte(want), f, 0644); err != nil {
		t.Errorf("SaferWriteFile(%s, %s) failed unexpectedly with err: %+v", "test-data", f, err)
	}

	got, err := os.ReadFile(f)
	if err != nil {
		t.Errorf("os.ReadFile(%s) failed unexpectedly with err: %+v", f, err)
	}
	if string(got) != want {
		t.Errorf("os.ReadFile(%s) = %s, want %s", f, string(got), want)
	}

	i, err := os.Stat(f)
	if err != nil {
		t.Errorf("os.Stat(%s) failed unexpectedly with err: %+v", f, err)
	}

	if i.Mode().Perm() != 0o644 {
		t.Errorf("SaferWriteFile(%s) set incorrect permissions, os.Stat(%s) = %o, want %o", f, f, i.Mode().Perm(), 0o644)
	}
}

func TestCopyFile(t *testing.T) {
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "dst")
	src := filepath.Join(tmp, "src")
	want := "testdata"
	if err := os.WriteFile(src, []byte(want), 0777); err != nil {
		t.Fatalf("failed to write test source file: %v", err)
	}
	if err := CopyFile(src, dst, 0644); err != nil {
		t.Errorf("CopyFile(%s, %s) failed unexpectedly with error: %v", src, dst, err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Errorf("unable to read %q: %v", dst, err)
	}
	if string(got) != want {
		t.Errorf("CopyFile(%s, %s) copied %q, expected %q", src, dst, string(got), want)
	}

	i, err := os.Stat(dst)
	if err != nil {
		t.Errorf("os.Stat(%s) failed unexpectedly with err: %+v", dst, err)
	}

	if i.Mode().Perm() != 0o644 {
		t.Errorf("SaferWriteFile(%s) set incorrect permissions, os.Stat(%s) = %o, want %o", dst, dst, i.Mode().Perm(), 0o644)
	}
}

func TestCopyFileError(t *testing.T) {
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "dst")
	src := filepath.Join(tmp, "src")

	if err := CopyFile(src, dst, 0644); err == nil {
		t.Errorf("CopyFile(%s, %s) succeeded for non-existent file, want error", src, dst)
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	f, err := os.CreateTemp(dir, "file")
	if err != nil {
		t.Fatalf("os.CreateTemp(%s, file) failed unexpectedly with error: %v", dir, err)
	}
	defer f.Close()

	tests := []struct {
		name  string
		want  bool
		fType Type
		path  string
	}{
		{
			name:  "existing_file",
			want:  true,
			fType: TypeFile,
			path:  f.Name(),
		},
		{
			name:  "exists_file_want_dir",
			want:  false,
			fType: TypeDir,
			path:  f.Name(),
		},
		{
			name:  "existing_dir",
			want:  true,
			fType: TypeDir,
			path:  dir,
		},
		{
			name:  "exists_dir_want_file",
			want:  false,
			fType: TypeFile,
			path:  dir,
		},
		{
			name:  "non_existing_file",
			want:  false,
			fType: TypeFile,
			path:  filepath.Join(t.TempDir(), "random"),
		},
		{
			name:  "non_existing_dir",
			want:  false,
			fType: TypeDir,
			path:  filepath.Join(t.TempDir(), "random"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := FileExists(test.path, test.fType); got != test.want {
				t.Errorf("FileExists(%s, %d) = %t, want = %t", test.path, test.fType, got, test.want)
			}
		})
	}
}
