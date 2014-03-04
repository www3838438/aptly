package debian

import (
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/utils"
	. "launchpad.net/gocheck"
)

type PackageCollectionSuite struct {
	collection *PackageCollection
	p          *Package
	db         database.Storage
}

var _ = Suite(&PackageCollectionSuite{})

func (s *PackageCollectionSuite) SetUpTest(c *C) {
	s.p = NewPackageFromControlFile(packageStanza.Copy())
	s.db, _ = database.OpenDB(c.MkDir())
	s.collection = NewPackageCollection(s.db)
}

func (s *PackageCollectionSuite) TearDownTest(c *C) {
	s.db.Close()
}

func (s *PackageCollectionSuite) TestUpdate(c *C) {
	// package doesn't exist, update ok
	err := s.collection.Update(s.p)
	c.Assert(err, IsNil)
	res, err := s.collection.ByKey(s.p.Key(""))
	c.Assert(err, IsNil)
	c.Assert(res.Equals(s.p), Equals, true)

	// same package, ok
	p2 := NewPackageFromControlFile(packageStanza.Copy())
	err = s.collection.Update(p2)
	c.Assert(err, IsNil)
	res, err = s.collection.ByKey(p2.Key(""))
	c.Assert(err, IsNil)
	c.Assert(res.Equals(s.p), Equals, true)

	// change some metadata
	p2.Source = "lala"
	err = s.collection.Update(p2)
	c.Assert(err, IsNil)
	res, err = s.collection.ByKey(p2.Key(""))
	c.Assert(err, IsNil)
	c.Assert(res.Equals(s.p), Equals, false)
	c.Assert(res.Equals(p2), Equals, true)

	// change file info
	p2 = NewPackageFromControlFile(packageStanza.Copy())
	p2.UpdateFiles(nil)
	res, err = s.collection.ByKey(p2.Key(""))
	err = s.collection.Update(p2)
	c.Assert(err, ErrorMatches, ".*conflict with existing packge")
	p2 = NewPackageFromControlFile(packageStanza.Copy())
	files := p2.Files()
	files[0].Checksums.MD5 = "abcdef"
	p2.UpdateFiles(files)
	res, err = s.collection.ByKey(p2.Key(""))
	err = s.collection.Update(p2)
	c.Assert(err, ErrorMatches, ".*conflict with existing packge")
}

func (s *PackageCollectionSuite) TestByKey(c *C) {
	err := s.collection.Update(s.p)
	c.Assert(err, IsNil)

	p2, err := s.collection.ByKey(s.p.Key(""))
	c.Assert(err, IsNil)
	c.Assert(p2.Equals(s.p), Equals, true)

	c.Check(p2.GetDependencies(0), DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)", "dpkg (>= 1.6)"})
	c.Check(p2.Extra()["Priority"], Equals, "extra")
	c.Check(p2.Files()[0].Filename, Equals, "alien-arena-common_7.40-2_i386.deb")
}

func (s *PackageCollectionSuite) TestByKeyOld_0_3(c *C) {
	key := []byte("Pi386 vmware-view-open-client 4.5.0-297975+dfsg-4+b1")
	s.db.Put(key, old_0_3_Package)

	p, err := s.collection.ByKey(key)
	c.Check(err, IsNil)
	c.Check(p.Name, Equals, "vmware-view-open-client")
	c.Check(p.Version, Equals, "4.5.0-297975+dfsg-4+b1")
	c.Check(p.Architecture, Equals, "i386")
	c.Check(p.Files(), DeepEquals, PackageFiles{
		PackageFile{Filename: "vmware-view-open-client_4.5.0-297975+dfsg-4+b1_i386.deb",
			Checksums: utils.ChecksumInfo{
				Size:   520080,
				MD5:    "9c61b54e2638a18f955a695b9162d6af",
				SHA1:   "5b7c99e64a70f4f509bfa3a674088ff9cef68163",
				SHA256: "4a9e4b2d9b3db13f9a29e522f3ffbb34eee96fc6f34a0647042ab1b5b0f2e04d"}}})
	c.Check(p.GetDependencies(0), DeepEquals, []string{"libatk1.0-0 (>= 1.12.4)", "libboost-signals1.49.0 (>= 1.49.0-1)",
		"libc6 (>= 2.3.6-6~)", "libcairo2 (>= 1.2.4)", "libcurl3 (>= 7.18.0)", "libfontconfig1 (>= 2.8.0)", "libfreetype6 (>= 2.2.1)",
		"libgcc1 (>= 1:4.1.1)", "libgdk-pixbuf2.0-0 (>= 2.22.0)", "libglib2.0-0 (>= 2.24.0)", "libgtk2.0-0 (>= 2.24.0)",
		"libicu48 (>= 4.8-1)", "libpango1.0-0 (>= 1.14.0)", "libssl1.0.0 (>= 1.0.0)", "libstdc++6 (>= 4.6)", "libx11-6",
		"libxml2 (>= 2.7.4)", "rdesktop"})
	c.Check(p.Extra()["Priority"], Equals, "optional")
}

func (s *PackageCollectionSuite) TestAllPackageRefs(c *C) {
	err := s.collection.Update(s.p)
	c.Assert(err, IsNil)

	refs := s.collection.AllPackageRefs()
	c.Check(refs.Len(), Equals, 1)
	c.Check(refs.Refs[0], DeepEquals, s.p.Key(""))
}

func (s *PackageCollectionSuite) TestDeleteByKey(c *C) {
	err := s.collection.Update(s.p)
	c.Assert(err, IsNil)

	_, err = s.db.Get(s.p.Key(""))
	c.Check(err, IsNil)

	_, err = s.db.Get(s.p.Key("xD"))
	c.Check(err, IsNil)

	_, err = s.db.Get(s.p.Key("xE"))
	c.Check(err, IsNil)

	_, err = s.db.Get(s.p.Key("xF"))
	c.Check(err, IsNil)

	err = s.collection.DeleteByKey(s.p.Key(""))
	c.Check(err, IsNil)

	_, err = s.collection.ByKey(s.p.Key(""))
	c.Check(err, ErrorMatches, "key not found")

	_, err = s.db.Get(s.p.Key(""))
	c.Check(err, ErrorMatches, "key not found")

	_, err = s.db.Get(s.p.Key("xD"))
	c.Check(err, ErrorMatches, "key not found")

	_, err = s.db.Get(s.p.Key("xE"))
	c.Check(err, ErrorMatches, "key not found")

	_, err = s.db.Get(s.p.Key("xF"))
	c.Check(err, ErrorMatches, "key not found")
}

// This is old package (pre-0.4) that would habe to be converted
var old_0_3_Package = []byte{0x8f, 0xac, 0x41, 0x72, 0x63, 0x68, 0x69, 0x74, 0x65, 0x63, 0x74, 0x75, 0x72, 0x65, 0xa4, 0x69, 0x33, 0x38, 0x36,
	0xac, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x44, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x73, 0xc0, 0xb1, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x44, 0x65,
	0x70, 0x65, 0x6e, 0x64, 0x73, 0x49, 0x6e, 0x44, 0x65, 0x70, 0xc0, 0xa7, 0x44, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x73, 0xdc, 0x0, 0x12,
	0xb7, 0x6c, 0x69, 0x62, 0x61, 0x74, 0x6b, 0x31, 0x2e, 0x30, 0x2d, 0x30, 0x20, 0x28, 0x3e, 0x3d, 0x20, 0x31, 0x2e, 0x31, 0x32, 0x2e,
	0x34, 0x29, 0xda, 0x0, 0x24, 0x6c, 0x69, 0x62, 0x62, 0x6f, 0x6f, 0x73, 0x74, 0x2d, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x6c, 0x73, 0x31,
	0x2e, 0x34, 0x39, 0x2e, 0x30, 0x20, 0x28, 0x3e, 0x3d, 0x20, 0x31, 0x2e, 0x34, 0x39, 0x2e, 0x30, 0x2d, 0x31, 0x29, 0xb3, 0x6c, 0x69,
	0x62, 0x63, 0x36, 0x20, 0x28, 0x3e, 0x3d, 0x20, 0x32, 0x2e, 0x33, 0x2e, 0x36, 0x2d, 0x36, 0x7e, 0x29, 0xb4, 0x6c, 0x69, 0x62, 0x63,
	0x61, 0x69, 0x72, 0x6f, 0x32, 0x20, 0x28, 0x3e, 0x3d, 0x20, 0x31, 0x2e, 0x32, 0x2e, 0x34, 0x29, 0xb4, 0x6c, 0x69, 0x62, 0x63, 0x75,
	0x72, 0x6c, 0x33, 0x20, 0x28, 0x3e, 0x3d, 0x20, 0x37, 0x2e, 0x31, 0x38, 0x2e, 0x30, 0x29, 0xb9, 0x6c, 0x69, 0x62, 0x66, 0x6f, 0x6e,
	0x74, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x31, 0x20, 0x28, 0x3e, 0x3d, 0x20, 0x32, 0x2e, 0x38, 0x2e, 0x30, 0x29, 0xb7, 0x6c, 0x69,
	0x62, 0x66, 0x72, 0x65, 0x65, 0x74, 0x79, 0x70, 0x65, 0x36, 0x20, 0x28, 0x3e, 0x3d, 0x20, 0x32, 0x2e, 0x32, 0x2e, 0x31, 0x29, 0xb4,
	0x6c, 0x69, 0x62, 0x67, 0x63, 0x63, 0x31, 0x20, 0x28, 0x3e, 0x3d, 0x20, 0x31, 0x3a, 0x34, 0x2e, 0x31, 0x2e, 0x31, 0x29, 0xbe, 0x6c,
	0x69, 0x62, 0x67, 0x64, 0x6b, 0x2d, 0x70, 0x69, 0x78, 0x62, 0x75, 0x66, 0x32, 0x2e, 0x30, 0x2d, 0x30, 0x20, 0x28, 0x3e, 0x3d, 0x20,
	0x32, 0x2e, 0x32, 0x32, 0x2e, 0x30, 0x29, 0xb8, 0x6c, 0x69, 0x62, 0x67, 0x6c, 0x69, 0x62, 0x32, 0x2e, 0x30, 0x2d, 0x30, 0x20, 0x28,
	0x3e, 0x3d, 0x20, 0x32, 0x2e, 0x32, 0x34, 0x2e, 0x30, 0x29, 0xb7, 0x6c, 0x69, 0x62, 0x67, 0x74, 0x6b, 0x32, 0x2e, 0x30, 0x2d, 0x30,
	0x20, 0x28, 0x3e, 0x3d, 0x20, 0x32, 0x2e, 0x32, 0x34, 0x2e, 0x30, 0x29, 0xb3, 0x6c, 0x69, 0x62, 0x69, 0x63, 0x75, 0x34, 0x38, 0x20,
	0x28, 0x3e, 0x3d, 0x20, 0x34, 0x2e, 0x38, 0x2d, 0x31, 0x29, 0xb9, 0x6c, 0x69, 0x62, 0x70, 0x61, 0x6e, 0x67, 0x6f, 0x31, 0x2e, 0x30,
	0x2d, 0x30, 0x20, 0x28, 0x3e, 0x3d, 0x20, 0x31, 0x2e, 0x31, 0x34, 0x2e, 0x30, 0x29, 0xb6, 0x6c, 0x69, 0x62, 0x73, 0x73, 0x6c, 0x31,
	0x2e, 0x30, 0x2e, 0x30, 0x20, 0x28, 0x3e, 0x3d, 0x20, 0x31, 0x2e, 0x30, 0x2e, 0x30, 0x29, 0xb3, 0x6c, 0x69, 0x62, 0x73, 0x74, 0x64,
	0x63, 0x2b, 0x2b, 0x36, 0x20, 0x28, 0x3e, 0x3d, 0x20, 0x34, 0x2e, 0x36, 0x29, 0xa8, 0x6c, 0x69, 0x62, 0x78, 0x31, 0x31, 0x2d, 0x36,
	0xb2, 0x6c, 0x69, 0x62, 0x78, 0x6d, 0x6c, 0x32, 0x20, 0x28, 0x3e, 0x3d, 0x20, 0x32, 0x2e, 0x37, 0x2e, 0x34, 0x29, 0xa8, 0x72, 0x64,
	0x65, 0x73, 0x6b, 0x74, 0x6f, 0x70, 0xa5, 0x45, 0x78, 0x74, 0x72, 0x61, 0x88, 0xa3, 0x54, 0x61, 0x67, 0xbd, 0x72, 0x6f, 0x6c, 0x65,
	0x3a, 0x3a, 0x70, 0x72, 0x6f, 0x67, 0x72, 0x61, 0x6d, 0x2c, 0x20, 0x75, 0x69, 0x74, 0x6f, 0x6f, 0x6c, 0x6b, 0x69, 0x74, 0x3a, 0x3a,
	0x67, 0x74, 0x6b, 0xa8, 0x50, 0x72, 0x69, 0x6f, 0x72, 0x69, 0x74, 0x79, 0xa8, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0xaa,
	0x4d, 0x61, 0x69, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0xda, 0x0, 0x28, 0x44, 0x65, 0x62, 0x69, 0x61, 0x6e, 0x20, 0x51, 0x41,
	0x20, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x20, 0x3c, 0x70, 0x61, 0x63, 0x6b, 0x61, 0x67, 0x65, 0x73, 0x40, 0x71, 0x61, 0x2e, 0x64, 0x65,
	0x62, 0x69, 0x61, 0x6e, 0x2e, 0x6f, 0x72, 0x67, 0x3e, 0xa8, 0x48, 0x6f, 0x6d, 0x65, 0x70, 0x61, 0x67, 0x65, 0xda, 0x0, 0x30, 0x68,
	0x74, 0x74, 0x70, 0x3a, 0x2f, 0x2f, 0x63, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x70, 0x2f, 0x76, 0x6d, 0x77, 0x61, 0x72, 0x65, 0x2d, 0x76, 0x69, 0x65, 0x77, 0x2d, 0x6f, 0x70, 0x65, 0x6e, 0x2d, 0x63, 0x6c, 0x69,
	0x65, 0x6e, 0x74, 0xaf, 0x44, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x2d, 0x6d, 0x64, 0x35, 0xda, 0x0, 0x20,
	0x62, 0x34, 0x34, 0x64, 0x34, 0x39, 0x62, 0x34, 0x37, 0x61, 0x65, 0x30, 0x35, 0x35, 0x32, 0x63, 0x62, 0x66, 0x61, 0x64, 0x32, 0x31,
	0x30, 0x64, 0x65, 0x32, 0x31, 0x63, 0x64, 0x65, 0x31, 0x39, 0xa7, 0x53, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0xab, 0x63, 0x6f, 0x6e,
	0x74, 0x72, 0x69, 0x62, 0x2f, 0x78, 0x31, 0x31, 0xae, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6c, 0x6c, 0x65, 0x64, 0x2d, 0x53, 0x69, 0x7a,
	0x65, 0xa4, 0x31, 0x34, 0x35, 0x39, 0xab, 0x44, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0xb9, 0x20, 0x56, 0x4d,
	0x77, 0x61, 0x72, 0x65, 0x20, 0x56, 0x69, 0x65, 0x77, 0x20, 0x4f, 0x70, 0x65, 0x6e, 0x20, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0xa,
	0xa5, 0x46, 0x69, 0x6c, 0x65, 0x73, 0x91, 0x82, 0xa9, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x75, 0x6d, 0x73, 0x84, 0xa3, 0x4d, 0x44,
	0x35, 0xda, 0x0, 0x20, 0x39, 0x63, 0x36, 0x31, 0x62, 0x35, 0x34, 0x65, 0x32, 0x36, 0x33, 0x38, 0x61, 0x31, 0x38, 0x66, 0x39, 0x35,
	0x35, 0x61, 0x36, 0x39, 0x35, 0x62, 0x39, 0x31, 0x36, 0x32, 0x64, 0x36, 0x61, 0x66, 0xa4, 0x53, 0x48, 0x41, 0x31, 0xda, 0x0, 0x28,
	0x35, 0x62, 0x37, 0x63, 0x39, 0x39, 0x65, 0x36, 0x34, 0x61, 0x37, 0x30, 0x66, 0x34, 0x66, 0x35, 0x30, 0x39, 0x62, 0x66, 0x61, 0x33,
	0x61, 0x36, 0x37, 0x34, 0x30, 0x38, 0x38, 0x66, 0x66, 0x39, 0x63, 0x65, 0x66, 0x36, 0x38, 0x31, 0x36, 0x33, 0xa6, 0x53, 0x48, 0x41,
	0x32, 0x35, 0x36, 0xda, 0x0, 0x40, 0x34, 0x61, 0x39, 0x65, 0x34, 0x62, 0x32, 0x64, 0x39, 0x62, 0x33, 0x64, 0x62, 0x31, 0x33, 0x66,
	0x39, 0x61, 0x32, 0x39, 0x65, 0x35, 0x32, 0x32, 0x66, 0x33, 0x66, 0x66, 0x62, 0x62, 0x33, 0x34, 0x65, 0x65, 0x65, 0x39, 0x36, 0x66,
	0x63, 0x36, 0x66, 0x33, 0x34, 0x61, 0x30, 0x36, 0x34, 0x37, 0x30, 0x34, 0x32, 0x61, 0x62, 0x31, 0x62, 0x35, 0x62, 0x30, 0x66, 0x32,
	0x65, 0x30, 0x34, 0x64, 0xa4, 0x53, 0x69, 0x7a, 0x65, 0xce, 0x0, 0x7, 0xef, 0x90, 0xa8, 0x46, 0x69, 0x6c, 0x65, 0x6e, 0x61, 0x6d,
	0x65, 0xda, 0x0, 0x5e, 0x70, 0x6f, 0x6f, 0x6c, 0x2f, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x69, 0x62, 0x2f, 0x76, 0x2f, 0x76, 0x6d, 0x77,
	0x61, 0x72, 0x65, 0x2d, 0x76, 0x69, 0x65, 0x77, 0x2d, 0x6f, 0x70, 0x65, 0x6e, 0x2d, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x2f, 0x76,
	0x6d, 0x77, 0x61, 0x72, 0x65, 0x2d, 0x76, 0x69, 0x65, 0x77, 0x2d, 0x6f, 0x70, 0x65, 0x6e, 0x2d, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74,
	0x5f, 0x34, 0x2e, 0x35, 0x2e, 0x30, 0x2d, 0x32, 0x39, 0x37, 0x39, 0x37, 0x35, 0x2b, 0x64, 0x66, 0x73, 0x67, 0x2d, 0x34, 0x2b, 0x62,
	0x31, 0x5f, 0x69, 0x33, 0x38, 0x36, 0x2e, 0x64, 0x65, 0x62, 0xa8, 0x49, 0x73, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0xc2, 0xa4, 0x4e,
	0x61, 0x6d, 0x65, 0xb7, 0x76, 0x6d, 0x77, 0x61, 0x72, 0x65, 0x2d, 0x76, 0x69, 0x65, 0x77, 0x2d, 0x6f, 0x70, 0x65, 0x6e, 0x2d, 0x63,
	0x6c, 0x69, 0x65, 0x6e, 0x74, 0xaa, 0x50, 0x72, 0x65, 0x44, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x73, 0xc0, 0xa8, 0x50, 0x72, 0x6f, 0x76,
	0x69, 0x64, 0x65, 0x73, 0xc0, 0xaa, 0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x73, 0xc0, 0xa6, 0x53, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0xda, 0x0, 0x2d, 0x76, 0x6d, 0x77, 0x61, 0x72, 0x65, 0x2d, 0x76, 0x69, 0x65, 0x77, 0x2d, 0x6f, 0x70, 0x65, 0x6e, 0x2d,
	0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x20, 0x28, 0x34, 0x2e, 0x35, 0x2e, 0x30, 0x2d, 0x32, 0x39, 0x37, 0x39, 0x37, 0x35, 0x2b, 0x64,
	0x66, 0x73, 0x67, 0x2d, 0x34, 0x29, 0xb2, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x41, 0x72, 0x63, 0x68, 0x69, 0x74, 0x65, 0x63, 0x74,
	0x75, 0x72, 0x65, 0xa0, 0xa8, 0x53, 0x75, 0x67, 0x67, 0x65, 0x73, 0x74, 0x73, 0xc0, 0xa7, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e,
	0xb6, 0x34, 0x2e, 0x35, 0x2e, 0x30, 0x2d, 0x32, 0x39, 0x37, 0x39, 0x37, 0x35, 0x2b, 0x64, 0x66, 0x73, 0x67, 0x2d, 0x34, 0x2b, 0x62, 0x31}
