package texture

import "image/color"

// John Carmack said the quake palette.lmp can be considered public domain
// because it is not an important asset to id.
var (
	// Palette is the quake palette with added alpha value.
	Palette = [256]color.RGBA{
		// White
		{0, 0, 0, 255}, {15, 15, 15, 255}, {31, 31, 31, 255}, {47, 47, 47, 255}, {63, 63, 63, 255}, {75, 75, 75, 255}, {91, 91, 91, 255}, {107, 107, 107, 255},
		{123, 123, 123, 255}, {139, 139, 139, 255}, {155, 155, 155, 255}, {171, 171, 171, 255}, {187, 187, 187, 255}, {203, 203, 203, 255}, {219, 219, 219, 255}, {235, 235, 235, 255},
		// Brown
		{15, 11, 7, 255}, {23, 15, 11, 255}, {31, 23, 11, 255}, {39, 27, 15, 255}, {47, 35, 19, 255}, {55, 43, 23, 255}, {63, 47, 23, 255}, {75, 55, 27, 255},
		{83, 59, 27, 255}, {91, 67, 31, 255}, {99, 75, 31, 255}, {107, 83, 31, 255}, {115, 87, 31, 255}, {123, 95, 35, 255}, {131, 103, 35, 255}, {143, 111, 35, 255},
		// Light Blue
		{11, 11, 15, 255}, {19, 19, 27, 255}, {27, 27, 39, 255}, {39, 39, 51, 255}, {47, 47, 63, 255}, {55, 55, 75, 255}, {63, 63, 87, 255}, {71, 71, 103, 255},
		{79, 79, 115, 255}, {91, 91, 127, 255}, {99, 99, 139, 255}, {107, 107, 151, 255}, {115, 115, 163, 255}, {123, 123, 175, 255}, {131, 131, 187, 255}, {139, 139, 203, 255},
		// Green
		{0, 0, 0, 255}, {7, 7, 0, 255}, {11, 11, 0, 255}, {19, 19, 0, 255}, {27, 27, 0, 255}, {35, 35, 0, 255}, {43, 43, 7, 255}, {47, 47, 7, 255},
		{55, 55, 7, 255}, {63, 63, 7, 255}, {71, 71, 7, 255}, {75, 75, 11, 255}, {83, 83, 11, 255}, {91, 91, 11, 255}, {99, 99, 11, 255}, {107, 107, 15, 255},
		// Red
		{7, 0, 0, 255}, {15, 0, 0, 255}, {23, 0, 0, 255}, {31, 0, 0, 255}, {39, 0, 0, 255}, {47, 0, 0, 255}, {55, 0, 0, 255}, {63, 0, 0, 255},
		{71, 0, 0, 255}, {79, 0, 0, 255}, {87, 0, 0, 255}, {95, 0, 0, 255}, {103, 0, 0, 255}, {111, 0, 0, 255}, {119, 0, 0, 255}, {127, 0, 0, 255},
		// Orange
		{19, 19, 0, 255}, {27, 27, 0, 255}, {35, 35, 0, 255}, {47, 43, 0, 255}, {55, 47, 0, 255}, {67, 55, 0, 255}, {75, 59, 7, 255}, {87, 67, 7, 255},
		{95, 71, 7, 255}, {107, 75, 11, 255}, {119, 83, 15, 255}, {131, 87, 19, 255}, {139, 91, 19, 255}, {151, 95, 27, 255}, {163, 99, 31, 255}, {175, 103, 35, 255},
		// Gold
		{35, 19, 7, 255}, {47, 23, 11, 255}, {59, 31, 15, 255}, {75, 35, 19, 255}, {87, 43, 23, 255}, {99, 47, 31, 255}, {115, 55, 35, 255}, {127, 59, 43, 255},
		{143, 67, 51, 255}, {159, 79, 51, 255}, {175, 99, 47, 255}, {191, 119, 47, 255}, {207, 143, 43, 255}, {223, 171, 39, 255}, {239, 203, 31, 255}, {255, 243, 27, 255},
		// Peach
		{11, 7, 0, 255}, {27, 19, 0, 255}, {43, 35, 15, 255}, {55, 43, 19, 255}, {71, 51, 27, 255}, {83, 55, 35, 255}, {99, 63, 43, 255}, {111, 71, 51, 255},
		{127, 83, 63, 255}, {139, 95, 71, 255}, {155, 107, 83, 255}, {167, 123, 95, 255}, {183, 135, 107, 255}, {195, 147, 123, 255}, {211, 163, 139, 255}, {227, 179, 151, 255},
		// Purple
		{171, 139, 163, 255}, {159, 127, 151, 255}, {147, 115, 135, 255}, {139, 103, 123, 255}, {127, 91, 111, 255}, {119, 83, 99, 255}, {107, 75, 87, 255}, {95, 63, 75, 255},
		{87, 55, 67, 255}, {75, 47, 55, 255}, {67, 39, 47, 255}, {55, 31, 35, 255}, {43, 23, 27, 255}, {35, 19, 19, 255}, {23, 11, 11, 255}, {15, 7, 7, 255},
		// Magenta
		{187, 115, 159, 255}, {175, 107, 143, 255}, {163, 95, 131, 255}, {151, 87, 119, 255}, {139, 79, 107, 255}, {127, 75, 95, 255}, {115, 67, 83, 255}, {107, 59, 75, 255},
		{95, 51, 63, 255}, {83, 43, 55, 255}, {71, 35, 43, 255}, {59, 31, 35, 255}, {47, 23, 27, 255}, {35, 19, 19, 255}, {23, 11, 11, 255}, {15, 7, 7, 255},
		// Tan
		{219, 195, 187, 255}, {203, 179, 167, 255}, {191, 163, 155, 255}, {175, 151, 139, 255}, {163, 135, 123, 255}, {151, 123, 111, 255}, {135, 111, 95, 255}, {123, 99, 83, 255},
		{107, 87, 71, 255}, {95, 75, 59, 255}, {83, 63, 51, 255}, {67, 51, 39, 255}, {55, 43, 31, 255}, {39, 31, 23, 255}, {27, 19, 15, 255}, {15, 11, 7, 255},
		// Light Green
		{111, 131, 123, 255}, {103, 123, 111, 255}, {95, 115, 103, 255}, {87, 107, 95, 255}, {79, 99, 87, 255}, {71, 91, 79, 255}, {63, 83, 71, 255}, {55, 75, 63, 255},
		{47, 67, 55, 255}, {43, 59, 47, 255}, {35, 51, 39, 255}, {31, 43, 31, 255}, {23, 35, 23, 255}, {15, 27, 19, 255}, {11, 19, 11, 255}, {7, 11, 7, 255},
		// Yellow
		{255, 243, 27, 255}, {239, 223, 23, 255}, {219, 203, 19, 255}, {203, 183, 15, 255}, {187, 167, 15, 255}, {171, 151, 11, 255}, {155, 131, 7, 255}, {139, 115, 7, 255},
		{123, 99, 7, 255}, {107, 83, 0, 255}, {91, 71, 0, 255}, {75, 55, 0, 255}, {59, 43, 0, 255}, {43, 31, 0, 255}, {27, 15, 0, 255}, {11, 7, 0, 255},
		// Blue
		{0, 0, 255, 255}, {11, 11, 239, 255}, {19, 19, 223, 255}, {27, 27, 207, 255}, {35, 35, 191, 255}, {43, 43, 175, 255}, {47, 47, 159, 255}, {47, 47, 143, 255},
		{47, 47, 127, 255}, {47, 47, 111, 255}, {47, 47, 95, 255}, {43, 43, 79, 255}, {35, 35, 63, 255}, {27, 27, 47, 255}, {19, 19, 31, 255}, {11, 11, 15, 255},
		// Fire
		{43, 0, 0, 255}, {59, 0, 0, 255}, {75, 7, 0, 255}, {95, 7, 0, 255}, {111, 15, 0, 255}, {127, 23, 7, 255}, {147, 31, 7, 255}, {163, 39, 11, 255},
		{183, 51, 15, 255}, {195, 75, 27, 255}, {207, 99, 43, 255}, {219, 127, 59, 255}, {227, 151, 79, 255}, {231, 171, 95, 255}, {239, 191, 119, 255}, {247, 211, 139, 255},
		// Brights
		{167, 123, 59, 255}, {183, 155, 55, 255}, {199, 195, 55, 255}, {231, 227, 87, 255}, {127, 191, 255, 255}, {171, 231, 255, 255}, {215, 255, 255, 255}, {103, 0, 0, 255},
		{139, 0, 0, 255}, {179, 0, 0, 255}, {215, 0, 0, 255}, {255, 0, 0, 255}, {255, 243, 147, 255}, {255, 247, 199, 255}, {255, 255, 255, 255}, {159, 91, 83, 255},
	}
)
