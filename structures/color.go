package structures

// Color is an int32 which provides helper methods to get RGBA values.
type Color int32

// Creates a new color, each value of r, g, b, a are bytes between 0-255
func NewColor(r, g, b, a byte) Color {
	return Color((int32(r) << 24) | (int32(g) << 16) | (int32(b) << 8) | int32(a))
}

// Red returns the red part of the color (0-255)
func (c Color) Red() byte {
	return byte(int32(c)>>24) & 0xFF
}

// Green returns the green part of the color (0-255)
func (c Color) Green() byte {
	return byte(int32(c)>>16) & 0xFF
}

// Blue returns the blue part of the color (0-255)
func (c Color) Blue() byte {
	return byte(int32(c)>>8) & 0xFF
}

// Alpha returns the alpha part of the color (0-255)
func (c Color) Alpha() byte {
	return byte(int32(c)>>0) & 0xFF
}
