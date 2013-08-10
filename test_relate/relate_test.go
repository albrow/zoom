package test_relate

import (
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/redis"
	. "launchpad.net/gocheck"
	"testing"
)

// Gocheck setup...
func Test(t *testing.T) {
	TestingT(t)
}

type RelateSuite struct{}

var _ = Suite(&RelateSuite{})

func (s *RelateSuite) SetUpSuite(c *C) {

	zoom.Init(&zoom.Configuration{Database: 9})

	conn := zoom.GetConn()
	defer conn.Close()

	n, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
		c.Error(err)
	}

	if n != 0 {
		c.Errorf("Database #9 is not empty, test can not continue")
	}

	err = zoom.Register(&Person{}, "person")
	if err != nil {
		c.Error(err)
	}
}

func (s *RelateSuite) TearDownSuite(c *C) {
	zoom.UnregisterName("person")
	conn := zoom.GetConn()
	_, err := conn.Do("flushdb")
	if err != nil {
		c.Error(err)
	}
	conn.Close()
	zoom.Close()
}
