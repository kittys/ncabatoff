package imglib

import . "gopkg.in/check.v1"
import "image"

func (s *MySuite) TestRGBConvertors(c *C) {
	rgb := getTestRgbImage(image.Point{5, 5})
	rgba := StdImage{rgb}.GetRGBA()
	c.Check(rgb, DeepEquals, NewRGBFromRGBADropAlpha(rgba))
}
