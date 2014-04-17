package imglib

import . "gopkg.in/check.v1"
import "image"

func (s *MySuite) TestRgbConvertors(c *C) {
	rgb := getTestRgbImage(image.Point{128, 128})
	rgba := rgb.ToRGBA()
	c.Check(rgb, DeepEquals, NewRGBFromRGBADropAlpha(rgba))
}
