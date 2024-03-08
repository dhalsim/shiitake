package utils

import (
	"math"
	"net/url"
	"strconv"

	"github.com/diamondburned/gotkit/gtkutil"
)

// InjectAvatarSize calls InjectSize with size being 64px.
func InjectAvatarSize(urlstr string) string {
	return InjectSize(urlstr, 64)
}

// InjectSize injects the size query parameter into the URL. Size is
// automatically scaled up to 2x or more.
func InjectSize(urlstr string, size int) string {
	if urlstr == "" {
		return ""
	}

	if scale := gtkutil.ScaleFactor(); scale > 2 {
		size *= scale
	} else {
		size *= 2
	}

	return InjectSizeUnscaled(urlstr, size)
}

// InjectSizeUnscaled is like InjectSize, except the size is not scaled
// according to the scale factor.
func InjectSizeUnscaled(urlstr string, size int) string {
	// Round size up to the nearest power of 2.
	size = roundSize(size)

	u, err := url.Parse(urlstr)
	if err != nil {
		return urlstr
	}

	q := u.Query()
	q.Set("size", strconv.Itoa(size))
	u.RawQuery = q.Encode()

	return u.String()
}

func roundSize(size int) int {
	// Round size up to the nearest power of 2.
	return int(math.Pow(2, math.Ceil(math.Log2(float64(size)))))
}
