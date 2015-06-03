// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2014-2015 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package helpers

import (
	"compress/gzip"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type HTestSuite struct{}

var _ = Suite(&HTestSuite{})

func (ts *HTestSuite) TestUnpack(c *C) {

	// setup tmpdir
	tmpdir := c.MkDir()
	tmpfile := filepath.Join(tmpdir, "foo.tar.gz")

	// ok, slightly silly
	path := "/etc/fstab"

	// create test dir, symlink, and also test file
	someDir := c.MkDir()
	c.Assert(os.Symlink(path, filepath.Join(someDir, "fstab")), IsNil)

	cmd := exec.Command("tar", "cvzf", tmpfile, path, someDir)
	output, err := cmd.CombinedOutput()
	c.Assert(err, IsNil)
	c.Check(string(output), Matches, `(?ms).*^/etc/fstab`,
		Commentf("Can not find expected output from tar"))

	// unpack
	unpackdir := filepath.Join(tmpdir, "t")
	f, err := os.Open(tmpfile)
	c.Assert(err, IsNil)

	f2, err := gzip.NewReader(f)
	c.Assert(err, IsNil)

	err = UnpackTar(f2, unpackdir, nil)
	c.Assert(err, IsNil)

	// we have the expected file
	_, err = os.Open(filepath.Join(tmpdir, "t/etc/fstab"))
	c.Assert(err, IsNil)

	// and the expected dir is there and has the right mode
	unpackedSomeDir := filepath.Join(tmpdir, "t", someDir)
	c.Assert(IsDirectory(unpackedSomeDir), Equals, true)
	st1, err := os.Stat(unpackedSomeDir)
	c.Assert(err, IsNil)
	st2, err := os.Stat(someDir)
	c.Assert(err, IsNil)
	c.Assert(st1.Mode(), Equals, st2.Mode())

	// and the symlink is there too
	fn, err := os.Readlink(filepath.Join(unpackedSomeDir, "fstab"))
	c.Check(err, IsNil)
	c.Check(fn, Equals, "/etc/fstab")
}

func (ts *HTestSuite) TestUbuntuArchitecture(c *C) {
	goarch = "arm"
	c.Check(UbuntuArchitecture(), Equals, "armhf")

	goarch = "amd64"
	c.Check(UbuntuArchitecture(), Equals, "amd64")

	goarch = "386"
	c.Check(UbuntuArchitecture(), Equals, "i386")
}

func (ts *HTestSuite) TestChdir(c *C) {
	tmpdir := c.MkDir()

	cwd, err := os.Getwd()
	c.Assert(err, IsNil)
	c.Assert(cwd, Not(Equals), tmpdir)
	ChDir(tmpdir, func() {
		cwd, err := os.Getwd()
		c.Assert(err, IsNil)
		c.Assert(cwd, Equals, tmpdir)
	})
}

func (ts *HTestSuite) TestExitCode(c *C) {
	cmd := exec.Command("true")
	err := cmd.Run()
	c.Assert(err, IsNil)

	cmd = exec.Command("false")
	err = cmd.Run()
	c.Assert(err, NotNil)
	e, err := ExitCode(err)
	c.Assert(err, IsNil)
	c.Assert(e, Equals, 1)

	cmd = exec.Command("sh", "-c", "exit 7")
	err = cmd.Run()
	e, err = ExitCode(err)
	c.Assert(e, Equals, 7)

	// ensure that non exec.ExitError values give a error
	_, err = os.Stat("/random/file/that/is/not/there")
	c.Assert(err, NotNil)
	_, err = ExitCode(err)
	c.Assert(err, NotNil)
}

func (ts *HTestSuite) TestMakeMapFromEnvList(c *C) {
	envList := []string{
		"PATH=/usr/bin:/bin",
		"DBUS_SESSION_BUS_ADDRESS=unix:abstract=something1234",
	}
	envMap := MakeMapFromEnvList(envList)
	c.Assert(envMap, DeepEquals, map[string]string{
		"PATH": "/usr/bin:/bin",
		"DBUS_SESSION_BUS_ADDRESS": "unix:abstract=something1234",
	})
}

func (ts *HTestSuite) TestMakeMapFromEnvListInvalidInput(c *C) {
	envList := []string{
		"nonsesne",
	}
	envMap := MakeMapFromEnvList(envList)
	c.Assert(envMap, DeepEquals, map[string]string(nil))
}

func (ts *HTestSuite) TestSha512sum(c *C) {
	tempdir := c.MkDir()

	p := filepath.Join(tempdir, "foo")
	err := ioutil.WriteFile(p, []byte("x"), 0644)
	c.Assert(err, IsNil)
	hashsum, err := Sha512sum(p)
	c.Assert(err, IsNil)
	c.Assert(hashsum, Equals, "a4abd4448c49562d828115d13a1fccea927f52b4d5459297f8b43e42da89238bc13626e43dcb38ddb082488927ec904fb42057443983e88585179d50551afe62")
}

func (ts *HTestSuite) TestFileDoesNotExist(c *C) {
	c.Assert(FileExists("/i-do-not-exist"), Equals, false)
}

func (ts *HTestSuite) TestFileExistsSimple(c *C) {
	fname := filepath.Join(c.MkDir(), "foo")
	err := ioutil.WriteFile(fname, []byte(fname), 0644)
	c.Assert(err, IsNil)

	c.Assert(FileExists(fname), Equals, true)
}

func (ts *HTestSuite) TestFileExistsExistsOddPermissions(c *C) {
	fname := filepath.Join(c.MkDir(), "foo")
	err := ioutil.WriteFile(fname, []byte(fname), 0100)
	c.Assert(err, IsNil)

	c.Assert(FileExists(fname), Equals, true)
}

func (ts *HTestSuite) TestIsDirectoryDoesNotExist(c *C) {
	c.Assert(IsDirectory("/i-do-not-exist"), Equals, false)
}

func (ts *HTestSuite) TestIsDirectorySimple(c *C) {
	dname := filepath.Join(c.MkDir(), "bar")
	err := os.Mkdir(dname, 0700)
	c.Assert(err, IsNil)

	c.Assert(IsDirectory(dname), Equals, true)
}

func (ts *HTestSuite) TestMakeRandomString(c *C) {
	// for our tests
	rand.Seed(1)

	s1 := MakeRandomString(10)
	c.Assert(s1, Equals, "GMWjGsAPga")

	s2 := MakeRandomString(5)
	c.Assert(s2, Equals, "TlmOD")
}

func (ts *HTestSuite) TestAtomicWriteFile(c *C) {
	tmpdir := c.MkDir()

	p := filepath.Join(tmpdir, "foo")
	err := AtomicWriteFile(p, []byte("canary"), 0644)
	c.Assert(err, IsNil)

	content, err := ioutil.ReadFile(p)
	c.Assert(err, IsNil)
	c.Assert(string(content), Equals, "canary")

	// no files left behind!
	d, err := ioutil.ReadDir(tmpdir)
	c.Assert(err, IsNil)
	c.Assert(len(d), Equals, 1)
}

func (ts *HTestSuite) TestAtomicWriteFilePermissions(c *C) {
	tmpdir := c.MkDir()

	p := filepath.Join(tmpdir, "foo")
	err := AtomicWriteFile(p, []byte(""), 0600)
	c.Assert(err, IsNil)

	st, err := os.Stat(p)
	c.Assert(err, IsNil)
	c.Assert(st.Mode()&os.ModePerm, Equals, os.FileMode(0600))
}

func (ts *HTestSuite) TestCurrentHomeDirHOMEenv(c *C) {
	tmpdir := c.MkDir()

	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)

	os.Setenv("HOME", tmpdir)
	home, err := CurrentHomeDir()
	c.Assert(err, IsNil)
	c.Assert(home, Equals, tmpdir)
}

func (ts *HTestSuite) TestCurrentHomeDirNoHomeEnv(c *C) {
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)

	os.Setenv("HOME", "")
	home, err := CurrentHomeDir()
	c.Assert(err, IsNil)
	c.Assert(home, Equals, oldHome)
}

func skipOnMissingDevKmsg(c *C) {
	_, err := os.Stat("/dev/kmsg")
	if err != nil {
		c.Skip("Can not stat /dev/kmsg")
	}
}

func (ts *HTestSuite) TestMajorMinorSimple(c *C) {
	skipOnMissingDevKmsg(c)

	stat, _ := os.Stat("/dev/kmsg")
	major, minor, err := MajorMinor(stat)
	c.Assert(err, IsNil)
	c.Assert(major, Equals, uint32(1))
	c.Assert(minor, Equals, uint32(11))
}

func (ts *HTestSuite) TestMajorMinorNoDevice(c *C) {
	stat, err := os.Stat(c.MkDir())
	c.Assert(err, IsNil)

	_, _, err = MajorMinor(stat)
	c.Assert(err, NotNil)
}

func (ts *HTestSuite) TestMakedev(c *C) {
	// $ python -c 'import os;print(os.makedev(1,11))'
	// 267
	c.Assert(Makedev(1, 11), Equals, uint32(267))
}

func (ts *HTestSuite) TestUnpacksMknod(c *C) {
	skipOnMissingDevKmsg(c)

	// mknod mock
	mknodWasCalled := false
	mknod = func(path string, mode uint32, dev int) error {
		mknodWasCalled = true
		return nil
	}

	// setup tmpdir
	tmpdir := c.MkDir()
	tmpfile := filepath.Join(tmpdir, "device.tar")

	cmd := exec.Command("tar", "cf", tmpfile, "/dev/kmsg")
	err := cmd.Run()
	c.Assert(err, IsNil)

	f, err := os.Open(tmpfile)
	c.Assert(err, IsNil)

	err = UnpackTar(f, c.MkDir(), nil)
	c.Assert(err, IsNil)
	c.Assert(mknodWasCalled, Equals, true)

}
