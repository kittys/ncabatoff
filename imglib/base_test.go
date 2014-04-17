package imglib

import . "gopkg.in/check.v1"
import "testing"

// Hook up gocheck into the gotest runner.
func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})
