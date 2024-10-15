package store

import (
	"database/sql"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/redhatinsights/mbop/internal/config"
	l "github.com/redhatinsights/mbop/internal/logger"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
	db    *sql.DB
	store Store
}

func (suite *TestSuite) SetupSuite() {
	_ = l.Init()
	store, err := setupPostgresStore()
	if err != nil {
		suite.FailNow("Failed to get postgres store", "%v", err)
	}
	suite.db = store.db
	suite.store = store
}

func (suite *TestSuite) TearDownSuite() {
	// teardown after we're all done, using the same query we run before each test.
	suite.BeforeTest("", "teardown")
	err := suite.db.Close()
	if err != nil {
		suite.FailNow("Failed to close db")
	}
}

func (suite *TestSuite) BeforeTest(_, testName string) {
	_, err := suite.db.Exec(`delete from registrations`)
	if err != nil {
		suite.FailNow("failed to clear out table for test", "test %v, error: %v", testName, err)
	}

	_, err = suite.db.Exec(`delete from allowlist`)
	if err != nil {
		suite.FailNow("failed to clear out table for test", "test %v, error: %v", testName, err)
	}
}

func TestSuiteRun(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (suite *TestSuite) TestCreateWithoutExtra() {
	r := Registration{
		OrgID:    "1234",
		UID:      "1234",
		Username: "foobar",
	}
	id, err := suite.store.Create(&r)
	suite.Nil(err, "failed to insert without extra")
	suite.NotEqual("", id, "something funky with returning the id")
}

func (suite *TestSuite) TestCreateWithExtra() {
	r := Registration{
		OrgID:    "1234",
		Username: "foobar",
		UID:      "1234",
		Extra:    map[string]interface{}{"thing": true},
	}
	id, err := suite.store.Create(&r)
	suite.Nil(err, "failed to insert")
	suite.NotEqual("", id, "something funky with returning the id")
}

func (suite *TestSuite) TestCreateDuplicateDisplayNameSameOrg() {
	r := Registration{
		OrgID:       "1234",
		Username:    "foobar",
		UID:         "1234",
		DisplayName: "dupe",
	}
	_, err := suite.store.Create(&r)
	suite.Nil(err, "failed to insert")

	r2 := Registration{
		OrgID:       "1234",
		Username:    "foobar",
		UID:         "2345",
		DisplayName: "dupe",
	}
	_, err = suite.store.Create(&r2)
	suite.Error(err, "inserted successfully even when it shouldn't have")
}

func (suite *TestSuite) TestCreateDuplicateDisplayNameDifferentOrg() {
	r := Registration{
		OrgID:       "1234",
		Username:    "foobar",
		UID:         "1234",
		DisplayName: "dupe",
	}
	_, err := suite.store.Create(&r)
	suite.Nil(err, "failed to insert")

	r2 := Registration{
		OrgID:       "2345",
		Username:    "foobar",
		UID:         "2345",
		DisplayName: "dupe",
	}
	_, err = suite.store.Create(&r2)
	suite.Nil(err)
}

func (suite *TestSuite) TestDelete() {
	r := Registration{
		OrgID:    "1234",
		Username: "foobar",
		UID:      "1234",
		Extra:    map[string]interface{}{"thing": true},
	}
	_, err := suite.store.Create(&r)
	suite.Nil(err, "failed to setup for deletion")

	err = suite.store.Delete("1234", "1234")
	suite.Nil(err, "failed to delete item")
}

func (suite *TestSuite) TestDeleteNotExisting() {
	err := suite.store.Delete("1234", "1234")
	suite.Error(err, "failed to fail to delete item")
}

func (suite *TestSuite) TestFindOne() {
	r := Registration{
		OrgID:    "1234",
		Username: "foobar",
		UID:      "1234",
		Extra:    map[string]interface{}{"thing": true},
	}
	_, err := suite.store.Create(&r)
	suite.Nil(err, "failed to insert: %v", err)

	found, err := suite.store.Find("1234", "1234")
	suite.Nil(err, "failed to find one registration")
	suite.Equal(found.UID, "1234")
	suite.Equal(found.OrgID, "1234")
	suite.Equal(found.Username, "foobar")
	suite.WithinDuration(found.CreatedAt, time.Now(), 5*time.Second)
}

func (suite *TestSuite) TestFindByUID() {
	r := Registration{
		OrgID:    "1234",
		Username: "foobar",
		UID:      "1234",
		Extra:    map[string]interface{}{"thing": true},
	}
	_, err := suite.store.Create(&r)
	suite.Nil(err, "failed to insert: %v", err)

	found, err := suite.store.FindByUID("1234")
	suite.Nil(err, "failed to find one registration")
	suite.Equal(found.UID, "1234")
	suite.Equal(found.OrgID, "1234")
	suite.Equal(found.Username, "foobar")
	suite.WithinDuration(found.CreatedAt, time.Now(), 5*time.Second)
}

func (suite *TestSuite) TestFindByUIDNotThere() {
	_, err := suite.store.FindByUID("1234")
	suite.Error(err, "failed to not find one registration")
}

func (suite *TestSuite) TestFindOneNotThere() {
	_, err := suite.store.Find("1234", "1234")
	suite.Error(err, "failed to not find one registration")
}

func (suite *TestSuite) TestFindAll() {
	r := Registration{OrgID: "1234", UID: "1234", DisplayName: "one"}
	_, err := suite.store.Create(&r)
	suite.Nil(err, "failed to insert")

	r.OrgID = "1234"
	r.UID = "2345"
	r.DisplayName = "two"
	_, err = suite.store.Create(&r)
	suite.Nil(err, "failed to insert")

	_, count, err := suite.store.All("1234", 0, 0)
	suite.Nil(err, "failed to list all registrations")
	suite.Equal(count, 2)
}

func (suite *TestSuite) TestUpdate() {
	r := Registration{OrgID: "1234", UID: "1234"}
	_, err := suite.store.Create(&r)
	suite.Nil(err, "failed to insert")

	err = suite.store.Update(
		&r,
		&RegistrationUpdate{Extra: &map[string]interface{}{"thing": true}},
	)
	suite.Nil(err, "failed to update registration")
}

func (suite *TestSuite) TestFindAllWithPagination() {
	for i := 0; i < 10; i++ {
		s := strconv.Itoa(i)
		_, err := suite.store.Create(&Registration{
			OrgID:       "a",
			UID:         s,
			DisplayName: s,
		})
		suite.Nil(err)
	}

	// stepping through the pages ensuring they start/end with where it's expected
	regs, count, err := suite.store.All("a", 5, 0)
	suite.Nil(err)
	suite.Equal(10, count)
	suite.Equal(5, len(regs))
	suite.Equal("9", regs[0].UID)
	suite.Equal("5", regs[len(regs)-1].UID)

	regs, count, err = suite.store.All("a", 5, 5)
	suite.Nil(err)
	suite.Equal(10, count)
	suite.Equal(5, len(regs))
	suite.Equal("4", regs[0].UID)
	suite.Equal("0", regs[len(regs)-1].UID)

	regs, count, err = suite.store.All("a", 5, 10)
	suite.Nil(err)
	suite.Equal(10, count)
	suite.Equal(0, len(regs))
}

func (suite *TestSuite) TestIPAllowedHappyPath() {
	config.Reset()
	defer config.Reset()
	os.Setenv("ALLOWLIST_ENABLED", "true")
	defer os.Setenv("ALLOWLIST_ENABLED", "false")

	suite.Nil(suite.store.AllowAddress(&AllowlistBlock{
		IPBlock: "10.0.0.1/24",
		OrgID:   "1234",
	}))

	allowed, err := suite.store.AllowedIP("10.0.0.100", "1234")
	suite.True(allowed)
	suite.Nil(err)
}

func (suite *TestSuite) TestIPAllowedBadPath() {
	config.Reset()
	defer config.Reset()
	os.Setenv("ALLOWLIST_ENABLED", "true")
	defer os.Setenv("ALLOWLIST_ENABLED", "false")

	suite.Nil(suite.store.AllowAddress(&AllowlistBlock{
		IPBlock: "10.0.0.1/24",
		OrgID:   "1234",
	}))

	allowed, err := suite.store.AllowedIP("8.8.8.8", "1234")
	suite.False(allowed)
	suite.Nil(err)
}

func (suite *TestSuite) TestIPAllowedHappyMultiple() {
	config.Reset()
	defer config.Reset()
	os.Setenv("ALLOWLIST_ENABLED", "true")
	defer os.Setenv("ALLOWLIST_ENABLED", "false")

	suite.Nil(suite.store.AllowAddress(&AllowlistBlock{
		IPBlock: "10.0.0.1/24",
		OrgID:   "1234",
	}))
	suite.Nil(suite.store.AllowAddress(&AllowlistBlock{
		IPBlock: "192.168.1.1/24",
		OrgID:   "1234",
	}))

	for _, ip := range []string{"10.0.0.100", "192.168.1.100", "10.0.0.20/32", "192.168.1.20/32"} {
		allowed, err := suite.store.AllowedIP(ip, "1234")
		suite.True(allowed)
		suite.Nil(err)
	}
}

func (suite *TestSuite) TestIPAllowedWithSystem() {
	config.Reset()
	defer config.Reset()
	os.Setenv("ALLOWLIST_ENABLED", "true")
	defer os.Setenv("ALLOWLIST_ENABLED", "false")

	suite.Nil(suite.store.AllowAddress(&AllowlistBlock{
		IPBlock: "10.0.0.1/24",
		OrgID:   "1234",
	}))
	suite.Nil(suite.store.AllowAddress(&AllowlistBlock{
		IPBlock: "192.168.1.1/24",
		OrgID:   "system",
	}))

	for _, ip := range []string{"10.0.0.100", "192.168.1.100", "10.0.0.20/32", "192.168.1.20/32"} {
		allowed, err := suite.store.AllowedIP(ip, "1234")
		suite.True(allowed)
		suite.Nil(err)
	}
}
